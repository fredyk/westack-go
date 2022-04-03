package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"regexp"
	"strings"

	"github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/gofiber/fiber/v2"
)

type Property struct {
	Type     interface{} `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default"`
}

type Relation struct {
	Type       string `json:"type"`
	Model      string `json:"model"`
	PrimaryKey string `json:"primaryKey"`
	ForeignKey string `json:"foreignKey"`
}

type ACL struct {
	AccessType    string `json:"accessType"`
	PrincipalType string `json:"principalType"`
	PrincipalId   string `json:"principalId"`
	Permission    string `json:"permission"`
	Property      string `json:"property"`
}

type CasbinConfig struct {
	RequestDefinition  string   `json:"requestDefinition"`
	PolicyDefinition   string   `json:"policyDefinition"`
	RoleDefinition     string   `json:"roleDefinition"`
	PolicyEffect       string   `json:"policyEffect"`
	MatchersDefinition string   `json:"matchersDefinition"`
	Policies           []string `json:"policies"`
}

type Config struct {
	Name       string              `json:"name"`
	Plural     string              `json:"plural"`
	Base       string              `json:"base"`
	Datasource string              `json:"dataSource"`
	Public     bool                `json:"public"`
	Properties map[string]Property `json:"properties"`
	Relations  map[string]Relation `json:"relations"`
	Acls       []ACL               `json:"acls"`
	Hidden     []string            `json:"hidden"`
	Casbin     CasbinConfig        `json:"casbin"`
}

type DataSourceConfig struct {
	Name      string `json:"name"`
	Connector string `json:"connector"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	User      string `json:"user"`
	Password  string `json:"password"`
}

type Model struct {
	Name             string                 `json:"name"`
	Config           *Config                `json:"-"`
	Datasource       *datasource.Datasource `json:"-"`
	Router           *fiber.Router          `json:"-"`
	App              *wst.IApp              `json:"-"`
	BaseUrl          string                 `json:"-"`
	CasbinModel      *casbinmodel.Model
	CasbinAdapter    **fileadapter.Adapter
	Enforcer         *casbin.Enforcer
	DisabledHandlers map[string]bool

	eventHandlers    map[string]func(eventContext *EventContext) error
	modelRegistry    *map[string]*Model
	remoteMethodsMap map[string]*OperationItem
}

func (loadedModel *Model) SendError(ctx *fiber.Ctx, err error) error {
	switch err.(type) {
	case *WeStackError:
		return ctx.Status((err).(*WeStackError).FiberError.Code).JSON(fiber.Map{
			"error": fiber.Map{
				"statusCode": (err).(*WeStackError).FiberError.Code,
				"name":       "Error",
				"code":       err.(*WeStackError).Code,
				"error":      err.(*WeStackError).FiberError.Error(),
				"message":    (err.(*WeStackError).Details)["message"],
				"details":    err.(*WeStackError).Details,
			},
		})
	}
	return err
}

func New(config *Config, modelRegistry *map[string]*Model) *Model {
	loadedModel := &Model{
		Name:             config.Name,
		Config:           config,
		DisabledHandlers: map[string]bool{},

		modelRegistry:    modelRegistry,
		eventHandlers:    map[string]func(eventContext *EventContext) error{},
		remoteMethodsMap: map[string]*OperationItem{},
	}

	(*modelRegistry)[config.Name] = loadedModel

	return loadedModel
}

type Instance struct {
	Model *Model
	Id    interface{}

	data  wst.M
	bytes []byte
}

type RegistryEntry struct {
	Name  string
	Model *Model
}

func (modelInstance *Instance) ToJSON() wst.M {

	if modelInstance == nil {
		return nil
	}

	var result wst.M
	result = wst.CopyMap(modelInstance.data)
	for relationName, relationConfig := range modelInstance.Model.Config.Relations {
		if modelInstance.data[relationName] != nil {
			if relationConfig.Type == "" {
				// relation not found
				continue
			}
			rawRelatedData := modelInstance.data[relationName]
			relatedModel, err := modelInstance.Model.App.FindModel(relationConfig.Model)
			if err != nil {
				return nil
			}
			if relatedModel != nil {
				switch relationConfig.Type {
				case "belongsTo", "hasOne":
					relatedInstance := rawRelatedData.(*Instance).ToJSON()
					result[relationName] = relatedInstance
				case "hasMany", "hasAndBelongsToMany":
					aux := make(wst.A, len(rawRelatedData.([]Instance)))
					for idx, v := range rawRelatedData.([]Instance) {
						aux[idx] = v.ToJSON()
					}
					result[relationName] = aux
				}
			}
		}
	}

	return result
}

func (modelInstance Instance) Get(relationName string) interface{} {
	result := modelInstance.data[relationName]
	switch modelInstance.Model.Config.Relations[relationName].Type {
	case "hasMany", "hasAndBelongsToMany":
		if result == nil {
			result = make([]Instance, 0)
		}
		break
	}
	return result
}

func (modelInstance Instance) GetOne(relationName string) *Instance {
	result := modelInstance.Get(relationName)
	if result == nil {
		return nil
	}
	return result.(*Instance)
}

func (modelInstance Instance) GetMany(relationName string) []Instance {
	return modelInstance.Get(relationName).([]Instance)
}

func (loadedModel *Model) Build(data wst.M, baseContext *EventContext) Instance {

	if data["id"] == nil {
		data["id"] = data["_id"]
		if data["id"] != nil {
			delete(data, "_id")
		}
	}

	_bytes, _ := bson.Marshal(data)

	var targetBaseContext = baseContext
	deepLevel := 0
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
		deepLevel++
	}

	for relationName, relationConfig := range loadedModel.Config.Relations {
		if data[relationName] != nil {
			if relationConfig.Type == "" {
				// relation not found
				continue
			}
			rawRelatedData := data[relationName]
			relatedModel, err := loadedModel.App.FindModel(relationConfig.Model)
			if err != nil {
				log.Printf("ERROR: Model.Build() --> %v\n", err)
				return Instance{}
			}
			if relatedModel != nil {
				switch relationConfig.Type {
				case "belongsTo", "hasOne":
					relatedInstance := relatedModel.(*Model).Build(rawRelatedData.(wst.M), targetBaseContext)
					data[relationName] = &relatedInstance
				case "hasMany", "hasAndBelongsToMany":
					result := make([]Instance, len(rawRelatedData.(primitive.A)))
					for idx, v := range rawRelatedData.(primitive.A) {
						result[idx] = relatedModel.(*Model).Build(v.(wst.M), targetBaseContext)
					}
					data[relationName] = result
				}
			}
		}
	}

	modelInstance := Instance{
		Id:    data["id"],
		bytes: _bytes,
		data:  data,
		Model: loadedModel,
	}
	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &data
	eventContext.Instance = &modelInstance

	if loadedModel.DisabledHandlers["__operation__loaded"] != true {
		err := loadedModel.GetHandler("__operation__loaded")(eventContext)
		if err != nil {
			log.Println("Warning", err)
			return Instance{}
		}
	}

	return modelInstance
}

func ParseFilter(filter string) *wst.Filter {
	var filterMap *wst.Filter
	if filter != "" {
		_ = json.Unmarshal([]byte(filter), &filterMap)
	}
	return filterMap
}

func (loadedModel *Model) FindMany(filterMap *wst.Filter, baseContext *EventContext) ([]Instance, error) {

	if baseContext == nil {
		baseContext = &EventContext{}
	}
	var targetBaseContext = baseContext
	deepLevel := 0
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
		deepLevel++
	}

	lookups := loadedModel.ExtractLookupsFromFilter(filterMap, baseContext.DisableTypeConversions)

	documents, err := loadedModel.Datasource.FindMany(loadedModel.Name, lookups)
	if err != nil {
		return nil, err
	}
	if documents == nil {
		return nil, errors.New("invalid query result")
	}

	var results = make([]Instance, len(*documents))

	for idx, document := range *documents {
		results[idx] = loadedModel.Build(document, targetBaseContext)
	}

	return results, nil
}

func (loadedModel *Model) ExtractLookupsFromFilter(filterMap *wst.Filter, disableTypeConversions bool) *wst.A {

	if filterMap == nil {
		return nil
	}

	var targetWhere *wst.Where
	if filterMap != nil && filterMap.Where != nil {
		whereCopy := *filterMap.Where
		targetWhere = &whereCopy
	} else {
		targetWhere = nil
	}

	var targetInclude *wst.Include
	if filterMap != nil && filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
	}
	var targetOrder *wst.Order
	if filterMap != nil && filterMap.Order != nil {
		orderValue := *filterMap.Order
		targetOrder = &orderValue
	} else {
		targetOrder = nil
	}
	var targetSkip = filterMap.Skip
	var targetLimit = filterMap.Limit

	var lookups *wst.A
	if targetWhere != nil {
		if !disableTypeConversions {
			datasource.ReplaceObjectIds(*targetWhere)
		}
		lookups = &wst.A{
			{"$match": *targetWhere},
		}
	} else {
		lookups = &wst.A{}
	}

	if targetOrder != nil && len(*targetOrder) > 0 {
		orderMap := wst.M{}
		for _, orderPair := range *targetOrder {
			splt := strings.Split(orderPair, " ")
			key := splt[0]
			directionSt := splt[1]
			if strings.ToLower(strings.TrimSpace(directionSt)) == "asc" {
				orderMap[key] = 1
			} else if strings.ToLower(strings.TrimSpace(directionSt)) == "desc" {
				orderMap[key] = -1
			} else {
				panic(fmt.Sprintf("Invalid direction %v while trying to sort by %v", directionSt, key))
			}
		}
		*lookups = append(*lookups, wst.M{
			"$sort": orderMap,
		})
	}

	if targetSkip > 0 {
		*lookups = append(*lookups, wst.M{
			"$skip": targetSkip,
		})
	}
	if targetLimit > 0 {
		*lookups = append(*lookups, wst.M{
			"$limit": targetLimit,
		})
	}

	if targetInclude != nil {
		for _, includeItem := range *targetInclude {

			var targetScope *wst.Filter
			if includeItem.Scope != nil {
				scopeValue := *includeItem.Scope
				targetScope = &scopeValue
			} else {
				targetScope = nil
			}

			relationName := includeItem.Relation
			relation := loadedModel.Config.Relations[relationName]
			relatedModelName := relation.Model
			relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

			if relatedLoadedModel == nil {
				log.Println()
				log.Printf("WARNING: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
				log.Println()
				continue
			}

			if relation.PrimaryKey == "" {
				relation.PrimaryKey = "_id"
			}

			if relation.ForeignKey == "" {
				switch relation.Type {
				case "belongsTo":
					relation.ForeignKey = strings.ToLower(relatedModelName[:1]) + relatedModelName[1:] + "Id"
					break
				case "hasMany":
					relation.ForeignKey = strings.ToLower(loadedModel.Name[:1]) + loadedModel.Name[1:] + "Id"
					break
				}
			}

			if relatedLoadedModel.Datasource.Name == loadedModel.Datasource.Name {
				switch relation.Type {
				case "belongsTo", "hasMany":
					var matching wst.M
					var lookupLet wst.M
					switch relation.Type {
					case "belongsTo":
						lookupLet = wst.M{
							relation.ForeignKey: fmt.Sprintf("$%v", relation.ForeignKey),
						}
						matching = wst.M{
							"$eq": []string{fmt.Sprintf("$%v", relation.PrimaryKey), fmt.Sprintf("$$%v", relation.ForeignKey)},
						}
						break
					case "hasMany":
						lookupLet = wst.M{
							relation.ForeignKey: fmt.Sprintf("$%v", relation.PrimaryKey),
						}
						matching = wst.M{
							"$eq": []string{fmt.Sprintf("$%v", relation.ForeignKey), fmt.Sprintf("$$%v", relation.ForeignKey)},
						}
						break
					}
					pipeline := []interface{}{
						wst.M{
							"$match": wst.M{
								"$expr": wst.M{
									"$and": wst.A{
										matching,
									},
								},
							},
						},
					}
					project := wst.M{}
					for _, propertyName := range relatedLoadedModel.Config.Hidden {
						project[propertyName] = false
					}
					if len(project) > 0 {
						pipeline = append(pipeline, wst.M{
							"$project": project,
						})
					}
					if targetScope != nil {
						nestedLoopkups := relatedLoadedModel.ExtractLookupsFromFilter(targetScope, disableTypeConversions)
						if nestedLoopkups != nil {
							for _, v := range *nestedLoopkups {
								pipeline = append(pipeline, v)
							}
						}
					}

					*lookups = append(*lookups, wst.M{
						"$lookup": wst.M{
							"from":     relatedLoadedModel.Name,
							"let":      lookupLet,
							"pipeline": pipeline,
							"as":       relationName,
						},
					})
					break
				}
				switch relation.Type {
				case "belongsTo":
					*lookups = append(*lookups, wst.M{
						"$unwind": wst.M{
							"path":                       fmt.Sprintf("$%v", relationName),
							"preserveNullAndEmptyArrays": true,
						},
					})
					break
				}

			}
		}

	} else {

	}
	return lookups
}

func (loadedModel *Model) FindOne(filterMap *wst.Filter, baseContext *EventContext) (*Instance, error) {

	if filterMap == nil {
		filterMap = &wst.Filter{}
	}
	filterMap.Limit = 1

	instances, err := loadedModel.FindMany(filterMap, baseContext)
	if err != nil {
		return nil, err
	}

	if len(instances) > 0 {
		return &instances[0], nil
	}

	return nil, nil
}

func (loadedModel *Model) FindById(id interface{}, filterMap *wst.Filter, baseContext *EventContext) (*Instance, error) {

	if baseContext == nil {
		baseContext = &EventContext{}
	}
	var targetBaseContext = baseContext
	deepLevel := 0
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
		deepLevel++
	}

	lookups := loadedModel.ExtractLookupsFromFilter(filterMap, baseContext.DisableTypeConversions)

	document, err := loadedModel.Datasource.FindById(loadedModel.Name, id, lookups)
	if err != nil {
		return nil, err
	} else if document == nil {
		return nil, datasource.NewError(404, fmt.Sprintf("%v %v not found", loadedModel.Name, id))
	} else {
		result := loadedModel.Build(*document, targetBaseContext)
		return &result, nil
	}
}

func (loadedModel *Model) Create(data interface{}, baseContext *EventContext) (*Instance, error) {

	var finalData wst.M
	switch data.(type) {
	case map[string]interface{}:
		finalData = wst.M{}
		for key, value := range data.(map[string]interface{}) {
			finalData[key] = value
		}
		break
	case *map[string]interface{}:
		finalData = wst.M{}
		for key, value := range *data.(*map[string]interface{}) {
			finalData[key] = value
		}
		break
	case wst.M:
		finalData = data.(wst.M)
		break
	case *wst.M:
		finalData = *data.(*wst.M)
		break
	case Instance:
		value := data.(Instance)
		finalData = (&value).ToJSON()
		break
	case *Instance:
		finalData = data.(*Instance).ToJSON()
		break
	default:
		log.Fatal(fmt.Sprintf("Invalid input for Model.Create() <- %s", data))
	}

	if baseContext == nil {
		baseContext = &EventContext{}
	}
	var targetBaseContext = baseContext
	deepLevel := 0
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
		deepLevel++
	}
	if !baseContext.DisableTypeConversions {
		datasource.ReplaceObjectIds(finalData)
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &finalData
	eventContext.IsNewInstance = true
	if loadedModel.DisabledHandlers["__operation__before_save"] != true {
		err := loadedModel.GetHandler("__operation__before_save")(eventContext)
		if err != nil {
			return nil, err
		}
	}
	for key := range loadedModel.Config.Relations {
		delete(finalData, key)
	}
	document, err := loadedModel.Datasource.Create(loadedModel.Name, &finalData)

	if err != nil {
		return nil, err
	} else if document == nil {
		return nil, datasource.NewError(400, "Could not create document")
	} else {
		result := loadedModel.Build(*document, eventContext)
		result.HideProperties()
		eventContext.Instance = &result
		if loadedModel.DisabledHandlers["__operation__after_save"] != true {
			err := loadedModel.GetHandler("__operation__after_save")(eventContext)
			if err != nil {
				return nil, err
			}
		}
		return &result, nil
	}

}

func (modelInstance *Instance) UpdateAttributes(data interface{}, baseContext *EventContext) (*Instance, error) {

	var finalData wst.M
	switch data.(type) {
	case map[string]interface{}:
		finalData = wst.M{}
		for key, value := range data.(map[string]interface{}) {
			finalData[key] = value
		}
		break
	case *map[string]interface{}:
		finalData = wst.M{}
		for key, value := range *data.(*map[string]interface{}) {
			finalData[key] = value
		}
		break
	case wst.M:
		finalData = data.(wst.M)
		break
	case *wst.M:
		finalData = *data.(*wst.M)
		break
	case Instance:
		value := data.(Instance)
		finalData = (&value).ToJSON()
		break
	case *Instance:
		finalData = data.(*Instance).ToJSON()
		break
	default:
		log.Fatal(fmt.Sprintf("Invalid input for Model.UpdateAttributes() <- %s", data))
	}

	if baseContext == nil {
		baseContext = &EventContext{}
	}
	var targetBaseContext = baseContext
	deepLevel := 0
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
		deepLevel++
	}
	if !baseContext.DisableTypeConversions {
		datasource.ReplaceObjectIds(finalData)
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &finalData
	eventContext.Instance = modelInstance
	eventContext.ModelID = modelInstance.Id
	eventContext.IsNewInstance = false
	if modelInstance.Model.DisabledHandlers["__operation__before_save"] != true {
		err := modelInstance.Model.GetHandler("__operation__before_save")(eventContext)
		if err != nil {
			return nil, err
		}
	}

	for key := range modelInstance.Model.Config.Relations {
		delete(finalData, key)
	}
	document, err := modelInstance.Model.Datasource.UpdateById(modelInstance.Model.Name, modelInstance.Id, &finalData)

	if err != nil {
		return nil, err
	} else if document == nil {
		return nil, datasource.NewError(400, "Could not update document")
	} else {
		err := modelInstance.Reload(eventContext)
		modelInstance.HideProperties()
		if err != nil {
			return nil, err
		}
		eventContext.Instance = modelInstance
		eventContext.ModelID = modelInstance.Id
		eventContext.IsNewInstance = false
		if modelInstance.Model.DisabledHandlers["__operation__after_save"] != true {
			err = modelInstance.Model.GetHandler("__operation__after_save")(eventContext)
			if err != nil {
				return nil, err
			}
		}
		return modelInstance, nil
	}
}

func (loadedModel *Model) DeleteById(id interface{}) (int64, error) {

	var finalId interface{}
	switch id.(type) {
	case string:
		if aux, err := primitive.ObjectIDFromHex(id.(string)); err != nil {
			finalId = aux
		} else {
			finalId = aux
		}
		break
	case primitive.ObjectID:
		finalId = id.(primitive.ObjectID)
		break
	case *primitive.ObjectID:
		finalId = *id.(*primitive.ObjectID)
		break
	default:
		if loadedModel.App.Debug {
			log.Println(fmt.Sprintf("WARNING: Invalid input for Model.DeleteById() <- %s", id))
		}
	}
	//TODO: Invoke hook for __operation__before_delete and __operation__after_delete
	deletedCount := loadedModel.Datasource.DeleteById(loadedModel.Name, finalId)
	if deletedCount > 0 {
		return deletedCount, nil
	} else {
		return 0, datasource.NewError(fiber.StatusNotFound, "Document not found")
	}
}

func (modelInstance Instance) HideProperties() {
	for _, propertyName := range modelInstance.Model.Config.Hidden {
		delete(modelInstance.data, propertyName)
		// TODO: Hide in nested
	}
}

func (modelInstance Instance) Transform(out interface{}) error {
	err := bson.Unmarshal(modelInstance.bytes, out)
	if err != nil {
		return err
	}
	return nil
}

func (modelInstance Instance) UncheckedTransform(out interface{}) interface{} {
	err := modelInstance.Transform(out)
	if err != nil {
		panic(err)
	}
	return out
}

func (modelInstance *Instance) Reload(eventContext *EventContext) error {
	newInstance, err := modelInstance.Model.FindById(modelInstance.Id, nil, eventContext)
	if err != nil {
		return err
	}
	for k := range modelInstance.data {
		if modelInstance.Model.Config.Relations[k].Model == "" {
			delete(modelInstance.data, k)
		}
	}
	for k, v := range newInstance.data {
		if modelInstance.Model.Config.Relations[k].Model == "" {
			modelInstance.data[k] = v
		}
	}
	modelInstance.data = newInstance.data
	_bytes, err := bson.Marshal(modelInstance.data)
	if err != nil {
		return err
	}
	modelInstance.bytes = _bytes
	return nil
}

type RemoteMethodOptionsHttp struct {
	Path string
	Verb string
}

type RemoteMethodOptions struct {
	Name        string
	Description string
	Http        RemoteMethodOptionsHttp
}

type OperationItem struct {
	Handler func(context *EventContext) error
	Options RemoteMethodOptions
}

func (loadedModel *Model) RemoteMethod(handler func(context *EventContext) error, options RemoteMethodOptions) fiber.Router {
	if !loadedModel.Config.Public {
		panic(fmt.Sprintf("Trying to register a remote method in the private model: %v, you may set \"public\": true in the %v.json file", loadedModel.Name, loadedModel.Name))
	}
	options.Name = strings.TrimSpace(options.Name)
	if options.Name == "" {
		panic("Method name cannot be empty")
	}
	if loadedModel.remoteMethodsMap[options.Name] != nil {
		panic(fmt.Sprintf("Already registered a remote method with name '%v'", options.Name))
	}

	_, err := loadedModel.Enforcer.AddRoleForUser(options.Name, "*")
	if err != nil {
		panic(err)
	}

	var http = options.Http
	path := strings.ToLower(http.Path)
	verb := strings.ToLower(http.Verb)
	description := options.Description

	var toInvoke func(string, ...fiber.Handler) fiber.Router
	operation := ""

	switch verb {
	case "get":
		toInvoke = (*loadedModel.Router).Get
		operation = "Finds"
	case "options":
		toInvoke = (*loadedModel.Router).Options
		operation = "Gets options for"
	case "head":
		toInvoke = (*loadedModel.Router).Head
		operation = "Checks"
	case "post":
		toInvoke = (*loadedModel.Router).Post
		operation = "Creates"
	case "put":
		toInvoke = (*loadedModel.Router).Put
		operation = "Replaces"
	case "patch":
		toInvoke = (*loadedModel.Router).Patch
		operation = "Updates attributes in"
	case "delete":
		toInvoke = (*loadedModel.Router).Delete
		operation = "Deletes"
	}

	fullPath := loadedModel.BaseUrl + "/" + path
	fullPath = regexp.MustCompile("//+").ReplaceAllString(fullPath, "/")

	if (*loadedModel.App.SwaggerPaths())[fullPath] == nil {
		(*loadedModel.App.SwaggerPaths())[fullPath] = wst.M{}
	}

	if description == "" {
		description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
	}

	var parameters wst.A
	if verb == "post" || verb == "put" || verb == "patch" {
		parameters = wst.A{
			{
				"name":        "data",
				"in":          "body",
				"description": "data",
				"required":    "true",
				"schema": wst.M{
					"type": "object",
				},
			},
		}
	} else {
		parameters = wst.A{}
	}
	(*loadedModel.App.SwaggerPaths())[fullPath][verb] = wst.M{
		"description": description,
		"consumes": []string{
			"*/*",
		},
		"produces": []string{
			"application/json",
		},
		"tags": []string{
			loadedModel.Config.Name,
		},
		"parameters": parameters,
		"summary":    description,
		"responses": wst.M{
			"200": wst.M{
				"description": "OK",
				"schema": wst.M{
					"type":                 "object",
					"additionalProperties": true,
				},
			},
		},
	}

	loadedModel.remoteMethodsMap[options.Name] = &OperationItem{
		Handler: handler,
		Options: options,
	}

	return toInvoke(path, func(ctx *fiber.Ctx) error {
		return loadedModel.HandleRemoteMethod(options.Name, &EventContext{
			Ctx:    ctx,
			Remote: &options,
		})
	})
}

type BearerUser struct {
	Id   interface{}
	Data interface{}
}

type BearerRole struct {
	Name string
}

type BearerToken struct {
	User  *BearerUser
	Roles []BearerRole
}

type EphemeralData wst.M

type EventContext struct {
	Bearer                 *BearerToken
	BaseContext            *EventContext
	Remote                 *RemoteMethodOptions
	Filter                 *wst.Filter
	Data                   *wst.M
	Instance               *Instance
	Ctx                    *fiber.Ctx
	Ephemeral              *EphemeralData
	IsNewInstance          bool
	Result                 interface{}
	ModelID                interface{}
	StatusCode             int
	DisableTypeConversions bool
}

type WeStackError struct {
	FiberError *fiber.Error
	Code       string
	Details    fiber.Map
	Ctx        *EventContext
}

func (eventContext *EventContext) UpdateEphemeral(newData *wst.M) {
	if eventContext != nil && newData != nil {
		if eventContext.Ephemeral == nil {
			eventContext.Ephemeral = &EphemeralData{}
		}
		for k, v := range *newData {
			(*eventContext.Ephemeral)[k] = v
		}
	}
}

func (err *WeStackError) Error() string {
	return fmt.Sprintf("%v %v: %v", err.FiberError.Code, err.FiberError.Error(), err.Details)
}

func (eventContext *EventContext) RestError(fiberError *fiber.Error, code string, details fiber.Map) error {
	eventContext.Result = details
	return &WeStackError{
		FiberError: fiberError,
		Code:       code,
		Details:    details,
		Ctx:        eventContext,
	}
}

func (eventContext *EventContext) GetBearer(loadedModel *Model) (error, *BearerToken) {

	if eventContext.Bearer != nil {
		return nil, eventContext.Bearer
	}
	c := eventContext.Ctx
	authBytes := c.Request().Header.Peek("Authorization")

	authBearerPair := strings.Split(strings.TrimSpace(string(authBytes)), "Bearer ")

	var user *BearerUser
	roles := make([]BearerRole, 0)
	if len(authBearerPair) == 2 {

		token, err := jwt.Parse(authBearerPair[1], func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return loadedModel.App.JwtSecretKey, nil
		})

		if token != nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				claimRoles := claims["roles"]
				userId := claims["userId"]
				user = &BearerUser{
					Id:   userId,
					Data: claims,
				}
				if claimRoles != nil {
					for _, role := range claimRoles.([]interface{}) {
						roles = append(roles, BearerRole{
							Name: role.(string),
						})
					}
				}
			} else {
				log.Println(err)
			}
		}

	}
	return nil, &BearerToken{
		User:  user,
		Roles: roles,
	}

}

func wrapEventHandler(model *Model, eventKey string, handler func(eventContext *EventContext) error) func(eventContext *EventContext) error {
	currentHandler := model.eventHandlers[eventKey]
	if currentHandler != nil {
		newHandler := handler
		handler = func(eventContext *EventContext) error {
			currentHandlerError := currentHandler(eventContext)
			if currentHandlerError != nil {
				if model.App.Debug {
					log.Println("WARNING: Stop handling on error", currentHandlerError)
				}
				return currentHandlerError
			} else {
				return newHandler(eventContext)
			}
		}
	}
	return handler
}

func (loadedModel *Model) On(event string, handler func(eventContext *EventContext) error) {
	loadedModel.eventHandlers[event] = wrapEventHandler(loadedModel, event, handler)
}

func (loadedModel *Model) Observe(operation string, handler func(eventContext *EventContext) error) {
	eventKey := "__operation__" + strings.ReplaceAll(strings.TrimSpace(operation), " ", "_")
	loadedModel.On(eventKey, handler)
}

func (loadedModel *Model) GetHandler(event string) func(eventContext *EventContext) error {
	res := loadedModel.eventHandlers[event]
	if res == nil {
		loadedModel.DisabledHandlers[event] = true
		res = func(eventContext *EventContext) error {
			if loadedModel.App.Debug {
				log.Println("no handler found for ", loadedModel.Name, ".", event)
			}
			return nil
		}
	}
	return res
}

func (loadedModel *Model) HandleRemoteMethod(name string, eventContext *EventContext) error {

	operationItem := loadedModel.remoteMethodsMap[name]

	if operationItem == nil {
		return errors.New(fmt.Sprintf("Method '%v' not found", name))
	}

	c := eventContext.Ctx
	options := operationItem.Options
	handler := operationItem.Handler

	err, token := eventContext.GetBearer(loadedModel)
	if err != nil {
		return err
	}

	action := options.Name

	if loadedModel.App.Debug {
		log.Println(fmt.Sprintf("DEBUG: Check auth for %v.%v (%v %v)", loadedModel.Name, options.Name, c.Method(), c.Path()))
	}

	if token.User == nil {
		allow, exp, err := loadedModel.Enforcer.EnforceEx("_EVERYONE_", "*", action)
		if loadedModel.App.Debug {
			log.Println("Explain", exp)
		}
		if err != nil {
			return err
		}
		if !allow {
			return fiber.ErrUnauthorized
		}
	} else {

		bearerUserIdSt := fmt.Sprintf("%v", token.User.Id)

		_, err := loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, "_EVERYONE_")
		if err != nil {
			return err
		}
		_, err = loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, "_AUTHENTICATED_")
		if err != nil {
			return err
		}
		for _, r := range token.Roles {
			_, err := loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, r.Name)
			if err != nil {
				return err
			}
		}
		err = loadedModel.Enforcer.SavePolicy()
		if err != nil {
			return err
		}

		if eventContext.ModelID != nil {
		}

		objId := "*"
		if eventContext.ModelID != nil {
			objId = GetIDAsString(eventContext.ModelID)
		} else {
			objId = c.Params("id")
			if objId == "" {
				objId = "*"
			}
		}

		allow, exp, err := loadedModel.Enforcer.EnforceEx(bearerUserIdSt, objId, action)

		if loadedModel.App.Debug {
			log.Println("Explain", exp)
		}
		if err != nil {
			return err
		}
		if !allow {
			return fiber.ErrUnauthorized
		}

	}

	eventContext.Bearer = token

	return handler(eventContext)
}

func GetIDAsString(idToConvert interface{}) string {
	foundObjUserId := idToConvert
	switch idToConvert.(type) {
	case primitive.ObjectID:
		foundObjUserId = idToConvert.(primitive.ObjectID).Hex()
		break
	case string:
		foundObjUserId = idToConvert
		break
	default:
		foundObjUserId = fmt.Sprintf("%v", idToConvert)
		break
	}
	return foundObjUserId.(string)
}
