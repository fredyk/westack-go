package model

import (
	"context"
	"encoding/json"
	"fmt"
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
	Name       string                 `json:"name"`
	Config     Config                 `json:"-"`
	Datasource *datasource.Datasource `json:"-"`
	Router     *fiber.Router          `json:"-"`
	App        *wst.IApp              `json:"-"`
	BaseUrl    string                 `json:"-"`

	eventHandlers map[string]func(eventContext *EventContext) error
	modelRegistry *map[string]*Model
}

func (loadedModel *Model) SendError(ctx *fiber.Ctx, err error) error {
	switch err.(type) {
	case *WeStackError:
		return ctx.Status((err).(*WeStackError).FiberError.Code).JSON(fiber.Map{"error": err.(*WeStackError).FiberError.Error(), "details": err.(*WeStackError).Details})
	}
	return err
}

func New(config Config, modelRegistry *map[string]*Model) *Model {
	loadedModel := &Model{
		Name:          config.Name,
		Config:        config,
		modelRegistry: modelRegistry,
		eventHandlers: map[string]func(eventContext *EventContext) error{},
	}

	(*modelRegistry)[config.Name] = loadedModel

	return loadedModel
}

type ModelInstance struct {
	Model *Model
	Id    interface{}

	data  wst.M
	bytes []byte
}

type RegistryEntry struct {
	Name  string
	Model *Model
}

func (modelInstance ModelInstance) ToJSON() wst.M {
	// Cannot hide here, it may be necessary
	//modelInstance.HideProperties()
	var result wst.M
	result = wst.CopyMap(modelInstance.data)
	for relationName, relationConfig := range modelInstance.Model.Config.Relations {
		if modelInstance.data[relationName] != nil {
			if relationConfig.Type == "" {
				// relation not found
				continue
			}
			rawRelatedData := modelInstance.data[relationName]
			relatedModel := modelInstance.Model.App.FindModel(relationConfig.Model).(*Model)
			if relatedModel != nil {
				switch relationConfig.Type {
				case "belongsTo", "hasOne":
					relatedInstance := rawRelatedData.(*ModelInstance).ToJSON()
					result[relationName] = relatedInstance
				case "hasMany", "hasAndBelongsToMany":
					aux := make(wst.A, len(rawRelatedData.([]ModelInstance)))
					for idx, v := range rawRelatedData.([]ModelInstance) {
						aux[idx] = v.ToJSON()
					}
					result[relationName] = aux
				}
			}
		}
	}

	return result
}

func (modelInstance ModelInstance) Get(relationName string) interface{} {
	result := modelInstance.data[relationName]
	switch modelInstance.Model.Config.Relations[relationName].Type {
	case "hasMany", "hasAndBelongsToMany":
		if result == nil {
			result = make([]ModelInstance, 0)
		}
		break
	}
	return result
}

func (modelInstance ModelInstance) GetOne(relationName string) *ModelInstance {
	return modelInstance.Get(relationName).(*ModelInstance)
}

func (modelInstance ModelInstance) GetMany(relationName string) []ModelInstance {
	return modelInstance.Get(relationName).([]ModelInstance)
}

func (loadedModel *Model) Build(data wst.M, fromDb bool) ModelInstance {

	if data["id"] == nil {
		data["id"] = data["_id"]
		if data["id"] != nil {
			delete(data, "_id")
		}
	}

	_bytes, _ := bson.Marshal(data)
	//data = common.CopyMap(data)

	for relationName, relationConfig := range loadedModel.Config.Relations {
		if data[relationName] != nil {
			if relationConfig.Type == "" {
				// relation not found
				continue
			}
			rawRelatedData := data[relationName]
			relatedModel := loadedModel.App.FindModel(relationConfig.Model).(*Model)
			if relatedModel != nil {
				switch relationConfig.Type {
				case "belongsTo", "hasOne":
					relatedInstance := relatedModel.Build(rawRelatedData.(wst.M), false)
					data[relationName] = &relatedInstance
				case "hasMany", "hasAndBelongsToMany":
					result := make([]ModelInstance, len(rawRelatedData.(primitive.A)))
					for idx, v := range rawRelatedData.(primitive.A) {
						result[idx] = relatedModel.Build(v.(wst.M), false)
					}
					data[relationName] = result
				}
			}
		}
	}

	modelInstance := ModelInstance{
		Id:    data["id"],
		bytes: _bytes,
		data:  data,
		Model: loadedModel,
	}
	eventContext := EventContext{
		Data:     &data,
		Instance: &modelInstance,
		Ctx:      nil,
	}
	loadedModel.GetHandler("__operation__loaded")(&eventContext)

	return modelInstance
}

func parseFilter(filter string) *wst.Filter {
	var filterMap *wst.Filter
	if filter != "" {
		_ = json.Unmarshal([]byte(filter), &filterMap)
	}
	return filterMap
}

func (loadedModel *Model) FindMany(filterMap *wst.Filter) ([]ModelInstance, error) {

	lookups := loadedModel.ExtractLookupsFromFilter(filterMap)

	var documents wst.A
	err := loadedModel.Datasource.FindMany(loadedModel.Name, filterMap, lookups).All(context.Background(), &documents)
	if err != nil {
		return nil, err
	}

	var results = make([]ModelInstance, len(documents))

	for idx, document := range documents {
		results[idx] = loadedModel.Build(document, true)
	}

	return results, nil
}

func (loadedModel *Model) ExtractLookupsFromFilter(filterMap *wst.Filter) *wst.A {

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
		//includeAsMaps, asMapsOk := filterMap.Include
		//if asMapsOk {
		//	includeAsInterfaces := make([]interface{}, len(includeAsMaps))
		//	for idx, v := range includeAsMaps {
		//		includeAsInterfaces[idx] = v
		//	}
		//	targetInclude = &includeAsInterfaces
		//} else {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces

		//}
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
		datasource.ReplaceObjectIds(*targetWhere)
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
					relation.ForeignKey = relatedModelName + "Id"
					break
				case "hasMany":
					relation.ForeignKey = loadedModel.Name + "Id"
					break
				}
			}

			if relatedLoadedModel.Datasource.Config["name"] == loadedModel.Datasource.Config["name"] {
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
						//	TODO: Recursive
						if loadedModel.App.Debug {
							log.Println("Process recursive scope ", targetScope)
						}
						nestedLoopkups := relatedLoadedModel.ExtractLookupsFromFilter(targetScope)
						if nestedLoopkups != nil {
							if loadedModel.App.Debug {
								log.Println("nested lookups: ", nestedLoopkups)
							}
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
				if targetScope != nil {
					if loadedModel.App.Debug {
						lookupBytes, _ := json.Marshal(*lookups)
						log.Println("final lookup: ", string(lookupBytes))
					}
				}

			}
		}

	} else {

	}
	return lookups
}

func (loadedModel *Model) FindOne(filterMap *wst.Filter) (*ModelInstance, error) {
	var documents wst.A

	lookups := loadedModel.ExtractLookupsFromFilter(filterMap)

	err := loadedModel.Datasource.FindMany(loadedModel.Name, filterMap, lookups).All(context.Background(), &documents)
	if err != nil {
		return nil, err
	}

	if len(documents) == 0 {
		return nil, nil
	} else {
		modelInstance := loadedModel.Build(documents[0], true)
		return &modelInstance, nil
	}
}

func (loadedModel *Model) FindById(id interface{}, filterMap *wst.Filter) (*ModelInstance, error) {
	var document wst.M

	lookups := loadedModel.ExtractLookupsFromFilter(filterMap)

	cursor := loadedModel.Datasource.FindById(loadedModel.Name, id, filterMap, lookups)
	if cursor != nil {
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		} else {
			result := loadedModel.Build(document, true)
			return &result, nil
		}
	} else {
		return nil, datasource.NewError(404, fmt.Sprintf("%v %v not found", loadedModel.Name, id))
	}
}

func (loadedModel *Model) Create(data interface{}) (*ModelInstance, error) {

	var finalData wst.M
	switch data.(type) {
	//case wst.M:
	//	finalData = wst.M{}
	//	for key, value := range data.(wst.M) {
	//		finalData[key] = value
	//	}
	//	break
	case wst.M:
		finalData = data.(wst.M)
		break
	case *wst.M:
		finalData = *data.(*wst.M)
		break
	case ModelInstance:
		finalData = data.(ModelInstance).ToJSON()
		break
	case *ModelInstance:
		finalData = data.(*ModelInstance).ToJSON()
		break
	default:
		log.Fatal(fmt.Sprintf("Invalid input for Model.Create() <- %s", data))
	}
	datasource.ReplaceObjectIds(finalData)
	eventContext := EventContext{
		Data:          &finalData,
		Ctx:           nil,
		IsNewInstance: true,
	}
	loadedModel.GetHandler("__operation__before_save")(&eventContext)
	var document wst.M
	for key := range loadedModel.Config.Relations {
		delete(finalData, key)
	}
	cursor := loadedModel.Datasource.Create(loadedModel.Name, &finalData)
	if cursor != nil {
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		} else {
			result := loadedModel.Build(document, true)
			result.HideProperties()
			loadedModel.GetHandler("__operation__after_save")(&EventContext{
				Data:          &result.data,
				Instance:      &result,
				Ctx:           nil,
				IsNewInstance: true,
			})
			return &result, nil
		}
	} else {
		return nil, datasource.NewError(400, "Could not create document")
	}

}

func (modelInstance *ModelInstance) UpdateAttributes(data interface{}) (*ModelInstance, error) {

	var finalData wst.M
	switch data.(type) {
	//case wst.M:
	//	finalData = wst.M{}
	//	for key, value := range data.(wst.M) {
	//		finalData[key] = value
	//	}
	//	break
	case wst.M:
		finalData = data.(wst.M)
		break
	case *wst.M:
		finalData = *data.(*wst.M)
		break
	case ModelInstance:
		finalData = data.(ModelInstance).ToJSON()
		break
	case *ModelInstance:
		finalData = data.(*ModelInstance).ToJSON()
		break
	default:
		log.Fatal(fmt.Sprintf("Invalid input for Model.UpdateAttributes() <- %s", data))
	}
	datasource.ReplaceObjectIds(finalData)
	eventContext := EventContext{
		Data:          &finalData,
		Instance:      modelInstance,
		Ctx:           nil,
		IsNewInstance: false,
	}
	modelInstance.Model.GetHandler("__operation__before_save")(&eventContext)
	var document wst.M
	for key := range modelInstance.Model.Config.Relations {
		delete(finalData, key)
	}
	cursor := modelInstance.Model.Datasource.UpdateById(modelInstance.Model.Name, modelInstance.Id, &finalData)
	if cursor != nil {
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		} else {
			err := modelInstance.Reload()
			modelInstance.HideProperties()
			if err != nil {
				return nil, err
			}
			modelInstance.Model.GetHandler("__operation__after_save")(&EventContext{
				Data:          &modelInstance.data,
				Instance:      modelInstance,
				Ctx:           nil,
				IsNewInstance: false,
			})
			return modelInstance, nil
		}
	} else {
		return nil, datasource.NewError(400, "Could not create document")
	}
}

func (loadedModel *Model) DeleteById(id interface{}) (int64, error) {

	var finalId interface{}
	switch id.(type) {
	case string:
		if aux, err := primitive.ObjectIDFromHex(id.(string)); err != nil {
			// id can be a non-objectid
			//return 0, err
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
	//eventContext := EventContext{
	//	Data:          &finalId,
	//	Instance:      modelInstance,
	//	Ctx:           nil,
	//	IsNewInstance: false,
	//}
	//modelInstance.Model.GetHandler("__operation__before_save")(&eventContext)
	deletedCount := loadedModel.Datasource.DeleteById(loadedModel.Name, finalId)
	if deletedCount > 0 {
		return deletedCount, nil
	} else {
		return 0, datasource.NewError(fiber.StatusNotFound, "Document not found")
	}
}

func handleError(c *fiber.Ctx, err error) error {
	switch err.(type) {
	case *datasource.OperationError:
		return c.Status(err.(*datasource.OperationError).Code).JSON(fiber.Map{"error": fiber.Map{"status": err.(*datasource.OperationError).Code, "message": err.(*datasource.OperationError).Message}})
	default:
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"status": 500, "message": "Internal Server Error"}})
	}
}

func (modelInstance ModelInstance) HideProperties() {
	for _, propertyName := range modelInstance.Model.Config.Hidden {
		delete(modelInstance.data, propertyName)
		// TODO: Hide in nested
		//for relationName, relation := range modelInstance.Model.Config.Relations {
		//	switch relation.Type {
		//	case "belongsTo":
		//		if modelInstance.data[relationName] != nil {
		//
		//		}
		//		break
		//	case "hasMany":
		//		break
		//	}
		//}
	}
}

func (modelInstance ModelInstance) Transform(out interface{}) error {
	err := bson.Unmarshal(modelInstance.bytes, out)
	if err != nil {
		return err
	}
	return nil
}

func (modelInstance ModelInstance) UncheckedTransform(out interface{}) interface{} {
	err := modelInstance.Transform(out)
	if err != nil {
		panic(err)
	}
	return out
}

func (modelInstance *ModelInstance) Reload() error {
	newInstance, err := modelInstance.Model.FindById(modelInstance.Id, nil)
	if err != nil {
		return err
	}
	for k, _ := range modelInstance.data {
		if modelInstance.Model.Config.Relations[k].Model == "" {
			// Delete this non-relation key
			delete(modelInstance.data, k)
		}
	}
	for k, v := range newInstance.data {
		if modelInstance.Model.Config.Relations[k].Model == "" {
			// Add this non-relation key
			modelInstance.data[k] = v
		}
	}
	// It's better not to update relations because of performance
	//for relationName, relation := range modelInstance.Model.Config.Relations {
	//	switch relation.Type {
	//	case "belongsTo", "hasOne":
	//		err := modelInstance.GetOne(relationName).Reload()
	//		if err != nil {
	//			return err
	//		}
	//		//modelInstance.data[relationName] =  modelInstance.GetOne(relationName).data
	//		break
	//	case "hasMany", "hasAndBelongsToMany":
	//		for _, related := range modelInstance.GetMany(relationName) {
	//			err := related.Reload()
	//			if err != nil {
	//				return err
	//			}
	//		}
	//		break
	//	}
	//}
	modelInstance.data = newInstance.data
	_bytes, err := bson.Marshal(modelInstance.data)
	if err != nil {
		return err
	}
	modelInstance.bytes = _bytes
	return nil
}

func (loadedModel *Model) FindManyRoute(c *fiber.Ctx) error {
	filterSt := c.Query("filter")
	filterMap := parseFilter(filterSt)

	result, err := loadedModel.FindMany(filterMap)
	out := make(wst.A, len(result))
	for idx, item := range result {
		item.HideProperties()
		out[idx] = item.ToJSON()
	}

	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(out)
}

func (loadedModel *Model) FindByIdRoute(c *fiber.Ctx) error {
	id := c.Params("id")
	filterSt := c.Query("filter")
	filterMap := parseFilter(filterSt)
	if filterMap == nil {
		filterMap = &wst.Filter{}
	}

	if filterSt == "" {

	}
	result, err := loadedModel.FindById(id, filterMap)
	if result != nil {
		result.HideProperties()
	}
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(result.ToJSON())
}

type RemoteMethodOptionsHttp struct {
	Path string
	Verb string
}

type RemoteMethodOptions struct {
	Description string
	Http        RemoteMethodOptionsHttp
}

func (loadedModel *Model) RemoteMethod(handler func(c *fiber.Ctx) error, options RemoteMethodOptions) fiber.Router {
	if !loadedModel.Config.Public {
		panic(fmt.Sprintf("Trying to register a remote method in the private model: %v, you may set \"public\": true in the %v.json file", loadedModel.Name, loadedModel.Name))
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

	return toInvoke(path, handler)
}

type EventContext struct {
	Data          *wst.M
	Instance      *ModelInstance
	Ctx           *fiber.Ctx
	IsNewInstance bool
	Result        interface{}
	ModelID       interface{}
	StatusCode    int
}

type WeStackError struct {
	FiberError *fiber.Error
	Details    fiber.Map
	Ctx        *EventContext
}

func (err *WeStackError) Error() string {
	return fmt.Sprintf("%v %v: %v", err.FiberError.Code, err.FiberError.Error(), err.Details)
}

func (ctx *EventContext) RestError(fiberError *fiber.Error, details fiber.Map) error {
	ctx.Result = details
	return &WeStackError{
		FiberError: fiberError,
		Details:    details,
		Ctx:        ctx,
	}
}

func wrapEventHandler(model *Model, eventKey string, handler func(eventContext *EventContext) error) func(eventContext *EventContext) error {
	currentHandler := model.eventHandlers[eventKey]
	if currentHandler != nil {
		newHandler := handler
		handler = func(eventContext *EventContext) error {
			currentHandlerResult := currentHandler(eventContext)
			if currentHandlerResult != nil {
				return currentHandlerResult
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
		res = func(eventContext *EventContext) error {
			if loadedModel.App.Debug {
				log.Println("no handler found for ", loadedModel.Name, ".", event)
			}
			return nil
		}
	}
	return res
}
