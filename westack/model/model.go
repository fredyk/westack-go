package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/oliveagle/jsonpath"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
)

type Property struct {
	Type     interface{} `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default"`
}

type Relation struct {
	Type       string  `json:"type"`
	Model      string  `json:"model"`
	PrimaryKey *string `json:"primaryKey"`
	ForeignKey *string `json:"foreignKey"`
	Options    struct {
		//Inverse bool `json:"inverse"`
		SkipAuth bool `json:"skipAuth"`
	} `json:"options"`
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

type CacheConfig struct {
	Datasource string     `json:"datasource"`
	Ttl        int        `json:"ttl"`
	Keys       [][]string `json:"keys"`
}

type MongoConfig struct {
	//Database string `json:"database"`
	Collection string `json:"collection"`
}

type Config struct {
	Name       string                `json:"name"`
	Plural     string                `json:"plural"`
	Base       string                `json:"base"`
	Public     bool                  `json:"public"`
	Properties map[string]Property   `json:"properties"`
	Relations  *map[string]*Relation `json:"relations"`
	Hidden     []string              `json:"hidden"`
	Casbin     CasbinConfig          `json:"casbin"`
	Cache      CacheConfig           `json:"cache"`
	Mongo      MongoConfig           `json:"mongo"`
}

type SimplifiedConfig struct {
	Datasource string `json:"dataSource"`
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
	CollectionName   string                 `json:"-"`
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

	authCache           map[string]map[string]map[string]bool
	hasHiddenProperties bool
}

func (loadedModel *Model) GetModelRegistry() *map[string]*Model {
	return loadedModel.modelRegistry
}

func (loadedModel *Model) SendError(ctx *fiber.Ctx, err error) error {
	switch err.(type) {
	case *wst.WeStackError:
		errorName := err.(*wst.WeStackError).Name
		if errorName == "" {
			errorName = "Error"
		}
		return ctx.Status((err).(*wst.WeStackError).FiberError.Code).JSON(fiber.Map{
			"error": fiber.Map{
				"statusCode": (err).(*wst.WeStackError).FiberError.Code,
				"name":       errorName,
				"code":       err.(*wst.WeStackError).Code,
				"error":      err.(*wst.WeStackError).FiberError.Error(),
				"message":    (err.(*wst.WeStackError).Details)["message"],
				"details":    err.(*wst.WeStackError).Details,
			},
		})
	default:
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"statusCode": 500,
				"name":       "InternalServerError",
				"code":       "INTERNAL_SERVER_ERROR",
				"error":      err.Error(),
				"message":    err.Error(),
			},
		})
	}
}

func New(config *Config, modelRegistry *map[string]*Model) *Model {
	name := config.Name
	collectionName := config.Mongo.Collection
	if collectionName == "" {
		collectionName = name
	}
	loadedModel := &Model{
		Name:             name,
		CollectionName:   collectionName,
		Config:           config,
		DisabledHandlers: map[string]bool{},

		modelRegistry:    modelRegistry,
		eventHandlers:    map[string]func(eventContext *EventContext) error{},
		remoteMethodsMap: map[string]*OperationItem{},
		authCache:        map[string]map[string]map[string]bool{},
	}

	(*modelRegistry)[name] = loadedModel

	return loadedModel
}

type Instance struct {
	Model *Model
	Id    interface{}

	data  wst.M
	bytes []byte
}

type InstanceA []Instance

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
	for relationName, relationConfig := range *modelInstance.Model.Config.Relations {
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
				switch {
				case isSingleRelation(relationConfig.Type):
					relatedInstance := rawRelatedData.(*Instance).ToJSON()
					result[relationName] = relatedInstance
					break
				case isManyRelation(relationConfig.Type):
					aux := make(wst.A, len(rawRelatedData.([]Instance)))
					for idx, v := range rawRelatedData.([]Instance) {
						aux[idx] = v.ToJSON()
					}
					result[relationName] = aux
					break
				}
			}
		}
	}

	return result
}

func isManyRelation(relationType string) bool {
	return relationType == "hasMany" || relationType == "hasManyThrough" || relationType == "hasAndBelongsToMany"
}

func isSingleRelation(relationType string) bool {
	return relationType == "hasOne" || relationType == "belongsTo"
}

func (instances InstanceA) ToJSON() []wst.M {
	result := make([]wst.M, len(instances))
	for idx, instance := range instances {
		result[idx] = instance.ToJSON()
	}
	return result
}

func (modelInstance Instance) Get(relationName string) interface{} {
	result := modelInstance.data[relationName]
	switch (*modelInstance.Model.Config.Relations)[relationName].Type {
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

	for relationName, relationConfig := range *loadedModel.Config.Relations {
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
					var relatedInstance Instance
					if asInstance, asInstanceOk := rawRelatedData.(Instance); asInstanceOk {
						relatedInstance = asInstance
					} else {
						relatedInstance = relatedModel.(*Model).Build(rawRelatedData.(wst.M), targetBaseContext)
					}
					data[relationName] = &relatedInstance
				case "hasMany", "hasAndBelongsToMany":

					var result []Instance
					if asInstanceList, asInstanceListOk := rawRelatedData.([]Instance); asInstanceListOk {
						result = asInstanceList
					} else {
						result = make([]Instance, len(rawRelatedData.(primitive.A)))
						for idx, v := range rawRelatedData.(primitive.A) {
							result[idx] = relatedModel.(*Model).Build(v.(wst.M), targetBaseContext)
						}
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

func (loadedModel *Model) FindMany(filterMap *wst.Filter, baseContext *EventContext) (InstanceA, error) {

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

	documents, err := loadedModel.Datasource.FindMany(loadedModel.CollectionName, lookups)
	if err != nil {
		return nil, err
	}
	if documents == nil {
		return nil, errors.New("invalid query result")
	}

	var targetInclude *wst.Include
	if filterMap != nil && filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
	}
	if targetInclude != nil {
		for _, includeItem := range *targetInclude {
			relationName := includeItem.Relation
			relation := (*loadedModel.Config.Relations)[relationName]
			relatedModelName := relation.Model
			relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]
			if relatedLoadedModel == nil {
				return nil, errors.New("related model not found")
			}

			err := loadedModel.mergeRelated(1, documents, includeItem, targetBaseContext)
			if err != nil {
				return nil, err
			}

		}
	}

	var results = make([]Instance, len(*documents))

	for idx, document := range *documents {
		results[idx] = loadedModel.Build(document, targetBaseContext)

		if targetInclude == nil && loadedModel.Config.Cache.Datasource != "" {
			// Dont cache if include is set
			cacheDs, err := loadedModel.App.FindDatasource(loadedModel.Config.Cache.Datasource)
			if err != nil {
				return nil, err
			}

			safeCacheDs := cacheDs.(*datasource.Datasource)

			toCache := wst.CopyMap(document)

			for _, keyGroup := range loadedModel.Config.Cache.Keys {
				canonicalId := ""
				for idx, key := range keyGroup {
					if idx > 0 {
						canonicalId = fmt.Sprintf("%v:", canonicalId)
					}
					v := (document)[key]
					if key == "_id" && v == nil && document["id"] != nil {
						v = document["id"]
					}
					switch v.(type) {
					case primitive.ObjectID:
						v = v.(primitive.ObjectID).Hex()
						break
					case *primitive.ObjectID:
						v = v.(*primitive.ObjectID).Hex()
						break
					default:
						break
					}
					canonicalId = fmt.Sprintf("%v%v:%v", canonicalId, key, v)
				}
				toCache["_redId"] = canonicalId
				_, err := safeCacheDs.Create(loadedModel.CollectionName, &toCache)
				if err != nil {
					return nil, err
				}
			}

			connectorName := safeCacheDs.Key + ".connector"
			switch safeCacheDs.Viper.GetString(connectorName) {
			case "redis":
				baseKey := fmt.Sprintf("%v:%v", safeCacheDs.Viper.GetString(safeCacheDs.Key+".database"), loadedModel.CollectionName)
				for _, keyGroup := range loadedModel.Config.Cache.Keys {
					keyToExpire := baseKey
					for _, key := range keyGroup {
						v := (document)[key]
						switch v.(type) {
						case primitive.ObjectID:
							v = v.(primitive.ObjectID).Hex()
							break
						case *primitive.ObjectID:
							v = v.(*primitive.ObjectID).Hex()
							break
						default:
							break
						}
						keyToExpire = fmt.Sprintf("%v:%v:%v", keyToExpire, key, v)
					}
					cmd := safeCacheDs.Db.(*redis.Client).Expire(safeCacheDs.Context, keyToExpire, time.Duration(loadedModel.Config.Cache.Ttl)*time.Second)
					_, err := cmd.Result()
					if err != nil {
						return nil, err
					}
				}
				break
			default:
				return nil, errors.New(fmt.Sprintf("Unsupported cache connector %v", connectorName))
			}

		}

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

	var targetInclude *wst.Include
	if filterMap != nil && filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
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
			relation := (*loadedModel.Config.Relations)[relationName]
			relatedModelName := relation.Model
			relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

			if relatedLoadedModel == nil {
				log.Println()
				log.Printf("WARNING: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
				log.Println()
				continue
			}

			if relatedLoadedModel.Datasource.Name == loadedModel.Datasource.Name {
				switch relation.Type {
				case "belongsTo", "hasOne", "hasMany":
					var matching wst.M
					var lookupLet wst.M
					switch relation.Type {
					case "belongsTo":
						lookupLet = wst.M{
							*relation.ForeignKey: fmt.Sprintf("$%v", *relation.ForeignKey),
						}
						matching = wst.M{
							"$eq": []string{fmt.Sprintf("$%v", *relation.PrimaryKey), fmt.Sprintf("$$%v", *relation.ForeignKey)},
						}
						break
					case "hasOne", "hasMany":
						lookupLet = wst.M{
							*relation.ForeignKey: fmt.Sprintf("$%v", *relation.PrimaryKey),
						}
						matching = wst.M{
							"$eq": []string{fmt.Sprintf("$%v", *relation.ForeignKey), fmt.Sprintf("$$%v", *relation.ForeignKey)},
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
							"from":     relatedLoadedModel.CollectionName,
							"let":      lookupLet,
							"pipeline": pipeline,
							"as":       relationName,
						},
					})
					break
				}
				switch relation.Type {
				case "hasOne", "belongsTo":
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
	var _id interface{}
	switch id.(type) {
	case string:
		var err error
		_id, err = primitive.ObjectIDFromHex(id.(string))
		if err != nil {
			_id = id
		}
	default:
		_id = id
	}

	if filterMap == nil {
		filterMap = &wst.Filter{}
	}
	if filterMap.Where == nil {
		filterMap.Where = &wst.Where{}
	}

	(*filterMap.Where)["_id"] = _id
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
	for key := range *loadedModel.Config.Relations {
		delete(finalData, key)
	}
	document, err := loadedModel.Datasource.Create(loadedModel.CollectionName, &finalData)

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

	for key := range *modelInstance.Model.Config.Relations {
		delete(finalData, key)
	}
	document, err := modelInstance.Model.Datasource.UpdateById(modelInstance.Model.CollectionName, modelInstance.Id, &finalData)

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
	deletedCount := loadedModel.Datasource.DeleteById(loadedModel.CollectionName, finalId)
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
		if (*modelInstance.Model.Config.Relations)[k] == nil {
			delete(modelInstance.data, k)
		}
	}
	for k, v := range newInstance.data {
		if (*modelInstance.Model.Config.Relations)[k] == nil {
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

func (modelInstance *Instance) GetString(path string) string {
	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	if err != nil {
		return ""
	}
	switch res.(type) {
	case string:
		return res.(string)
	case float64:
		return strconv.FormatFloat(res.(float64), 'f', -1, 64)
	case int:
		return strconv.Itoa(res.(int))
	case int64:
		return strconv.FormatInt(res.(int64), 10)
	case bool:
		return strconv.FormatBool(res.(bool))
	default:
		return ""
	}
}

func (modelInstance *Instance) GetFloat64(path string) float64 {
	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	if err != nil {
		return 0
	}
	switch res.(type) {
	case string:
		aux, err := strconv.ParseFloat(res.(string), 64)
		if err != nil {
			return 0
		}
		return aux
	case float64:
		return res.(float64)
	case int:
		return float64(res.(int))
	case int64:
		return float64(res.(int64))
	case bool:
		if res.(bool) {
			return 1
		} else {
			return 0
		}
	default:
		return 0
	}
}

func (modelInstance *Instance) GetInt(path string) int64 {

	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	if err != nil {
		return 0
	}
	switch res.(type) {
	case float64:
		return int64(res.(float64))
	case int64:
		return res.(int64)
	case int32:
		return int64(res.(int32))
	case int:
		return int64(res.(int))
	case float32:
		return int64(res.(float32))
	default:
		return 0
	}
}

func (modelInstance *Instance) GetBoolean(path string, defaultValue bool) bool {
	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	if err != nil {
		return defaultValue
	}
	switch res.(type) {
	case bool:
		return res.(bool)
	default:
		return defaultValue
	}
}

func (modelInstance *Instance) GetObjectId(path string) (result primitive.ObjectID) {
	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	result = primitive.NilObjectID
	if err == nil {
		switch res.(type) {
		case string:
			_id, err := primitive.ObjectIDFromHex(res.(string))
			if err == nil {
				result = _id
			}
			break
		case primitive.ObjectID:
			result = res.(primitive.ObjectID)
			break
		}
	}
	return result
}

func (modelInstance *Instance) GetM(path string) *wst.M {
	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	if err != nil {
		return nil
	}
	switch res.(type) {
	case wst.M:
		v := res.(wst.M)
		return &v
	case primitive.M:
		out := &wst.M{}
		for k, v := range res.(primitive.M) {
			(*out)[k] = v
		}
		return out
	case map[string]interface{}:
		out := &wst.M{}
		for k, v := range res.(map[string]interface{}) {
			(*out)[k] = v
		}
		return out
	default:
		return nil
	}
}

func (modelInstance *Instance) GetA(path string) *wst.A {
	res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path))
	if err != nil {
		return nil
	}
	switch res.(type) {
	case wst.A:
		v := res.(wst.A)
		return &v
	case primitive.A:
		out := &wst.A{}
		for idx, v := range res.(primitive.A) {
			*out = append(*out, wst.M{})
			for k, v := range v.(primitive.M) {
				(*out)[idx][k] = v
			}
		}
		return out
	case []interface{}:
		out := &wst.A{}
		for idx, v := range res.([]interface{}) {
			*out = append(*out, wst.M{})
			switch v.(type) {
			case wst.M:
				for k, v := range v.(wst.M) {
					(*out)[idx][k] = v
				}
			case primitive.M:
				for k, v := range v.(primitive.M) {
					(*out)[idx][k] = v
				}
			}
		}
		return out
	case []map[string]interface{}:
		out := &wst.A{}
		for idx, v := range res.([]map[string]interface{}) {
			*out = append(*out, wst.M{})
			for k, v := range v {
				(*out)[idx][k] = v
			}
		}
		return out
	default:
		log.Printf("WARNING: GetA: %v <%s> is not an array\n", path, modelInstance.data[path])
		return nil
	}
}

type RemoteMethodOptionsHttp struct {
	Path string
	Verb string
}

type ArgHttp struct {
	Source string
}

type RemoteMethodOptionsHttpArg struct {
	Arg         string
	Type        string
	Description string
	Http        ArgHttp
	Required    bool
}

type RemoteMethodOptionsHttpArgs []RemoteMethodOptionsHttpArg

type RemoteMethodOptions struct {
	Name        string
	Description string
	Accepts     RemoteMethodOptionsHttpArgs
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

	var http = options.Http
	path := http.Path
	verb := strings.ToLower(http.Verb)
	description := options.Description

	for _, arg := range options.Accepts {
		arg.Arg = strings.TrimSpace(arg.Arg)
		if arg.Arg == "" {
			panic(fmt.Sprintf("Argument name cannot be empty in the remote method '%v'", options.Name))
		}
		if arg.Http.Source != "query" && arg.Http.Source != "body" {
			panic(fmt.Sprintf("Argument '%v' in the remote method '%v' has an invalid 'in' value: '%v'", arg.Arg, options.Name, arg.Http.Source))
		}
	}

	_, err := loadedModel.Enforcer.AddRoleForUser(options.Name, "*")
	if err != nil {
		panic(err)
	}

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
	fullPath = regexp.MustCompile(`:(\w+)`).ReplaceAllString(fullPath, "{$1}")

	if (*loadedModel.App.SwaggerPaths())[fullPath] == nil {
		(*loadedModel.App.SwaggerPaths())[fullPath] = wst.M{}
	}

	if description == "" {
		description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
	}

	pathDef := wst.M{
		//"description": description,
		//"consumes": []string{
		//	"*/*",
		//},
		//"produces": []string{
		//	"application/json",
		//},
		"tags": []string{
			loadedModel.Name,
		},
		//"requestBody": requestBody,
		"summary": description,
		"security": []fiber.Map{
			{"bearerAuth": []string{}},
		},
		"responses": wst.M{
			"200": wst.M{
				"description": "OK",
				"content": wst.M{
					"application/json": wst.M{
						"schema": wst.M{
							"type": "object",
						},
					},
				},
				//"$ref": "#/components/schemas/" + loadedModel.Config.Name,
				//"schema": wst.M{
				//	"type":                 "object",
				//	"additionalProperties": true,
				//},
			},
		},
	}

	pathParams := regexp.MustCompile(`:(\w+)`).FindAllString(path, -1)

	params := make([]wst.M, len(pathParams))

	for idx, param := range pathParams {
		params[idx] = wst.M{
			"name":     strings.TrimPrefix(param, ":"),
			"in":       "path",
			"required": true,
			"schema": wst.M{
				"type": "string",
			},
		}
	}

	(*loadedModel.App.SwaggerPaths())[fullPath][verb] = pathDef

	if verb == "post" || verb == "put" || verb == "patch" {
		pathDef["requestBody"] = wst.M{
			"description": "data",
			"required":    true,
			//"name":        "data",
			//"in":          "body",
			//"schema": wst.M{
			//	"type": "object",
			//},
			"content": wst.M{
				"application/json": wst.M{
					"schema": wst.M{
						"type": "object",
					},
				},
			},
		}
	} else {

		for _, param := range options.Accepts {
			paramType := param.Type
			if paramType == "" {
				panic(fmt.Sprintf("Argument '%v' in the remote method '%v' has an invalid 'type' value: '%v'", param.Arg, options.Name, paramType))
			}
			paramDescription := param.Description
			if paramType == "date" {
				paramType = "string"
				paramDescription += " (format: ISO8601)"
			}
			params = append(params, wst.M{
				"name":        param.Arg,
				"in":          param.Http.Source,
				"description": paramDescription,
				"required":    param.Required,
				"schema": wst.M{
					"type": paramType,
				},
			})
		}

	}

	if len(params) > 0 {
		pathDef["parameters"] = params
	}

	(*loadedModel.App.SwaggerPaths())[fullPath][verb] = pathDef

	loadedModel.remoteMethodsMap[options.Name] = &OperationItem{
		Handler: handler,
		Options: options,
	}

	return toInvoke(path, func(ctx *fiber.Ctx) error {
		eventContext := &EventContext{
			Ctx:    ctx,
			Remote: &options,
		}
		err2 := loadedModel.HandleRemoteMethod(options.Name, eventContext)
		if err2 != nil {

			if err2 == fiber.ErrUnauthorized {
				err2 = wst.CreateError(fiber.ErrUnauthorized, "UNAUTHORIZED", fiber.Map{"message": "Unauthorized"}, "Error")
			}

			log.Printf("Error in remote method %v.%v (%v %v%v): %v\n", loadedModel.Name, options.Name, strings.ToUpper(verb), loadedModel.BaseUrl, path, err2.Error())
			return loadedModel.SendError(eventContext.Ctx, err2)
		}
		return nil
	})
}

type BearerUser struct {
	Id     interface{}
	Data   interface{}
	System bool
}

type BearerRole struct {
	Name string
}

type BearerToken struct {
	User   *BearerUser
	Roles  []BearerRole
	Raw    string
	Claims wst.M
}

type EphemeralData wst.M

type EventContext struct {
	Bearer                 *BearerToken
	BaseContext            *EventContext
	Remote                 *RemoteMethodOptions
	Filter                 *wst.Filter
	Data                   *wst.M
	Query                  *wst.M
	Instance               *Instance
	Ctx                    *fiber.Ctx
	Ephemeral              *EphemeralData
	IsNewInstance          bool
	Result                 interface{}
	ModelID                interface{}
	StatusCode             int
	DisableTypeConversions bool
	SkipFieldProtection    bool
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

func (eventContext *EventContext) GetBearer(loadedModel *Model) (error, *BearerToken) {

	if eventContext.Bearer != nil {
		return nil, eventContext.Bearer
	}
	c := eventContext.Ctx
	authBytes := c.Request().Header.Peek("Authorization")
	authSt := string(authBytes)
	if authSt == "" {
		authSt = eventContext.Ctx.Query("access_token")
		if authSt != "" {
			authSt = "Bearer " + authSt
		}
	}
	authBearerPair := strings.Split(strings.TrimSpace(authSt), "Bearer ")

	var user *BearerUser
	roles := make([]BearerRole, 0)
	bearerClaims := wst.M{}
	rawToken := ""
	if len(authBearerPair) == 2 {

		rawToken = authBearerPair[1]

		token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return loadedModel.App.JwtSecretKey, nil
		})

		if token != nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				for k, v := range claims {
					bearerClaims[k] = v
				}
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
		User:   user,
		Roles:  roles,
		Claims: bearerClaims,
		Raw:    rawToken,
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
					debug.PrintStack()
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

	objId := "*"
	if eventContext.ModelID != nil {
		objId = GetIDAsString(eventContext.ModelID)
	} else {
		objId = c.Params("id")
		if objId == "" {
			objId = "*"
		}
	}

	err, allowed := loadedModel.EnforceEx(token, objId, action, eventContext)
	if err != nil {
		return err
	}
	if !allowed {
		return fiber.ErrUnauthorized
	}

	eventContext.Bearer = token

	eventContext.Data = &wst.M{}
	eventContext.Query = &wst.M{}

	if strings.ToLower(options.Http.Verb) == "post" || strings.ToLower(options.Http.Verb) == "put" || strings.ToLower(options.Http.Verb) == "patch" {
		var data *wst.M
		bytes := eventContext.Ctx.Body()
		if len(bytes) > 0 {
			err := json.Unmarshal(bytes, &data)
			if err != nil {
				return wst.CreateError(fiber.ErrBadRequest, "INVALID_BODY", fiber.Map{"message": err.Error()}, "ValidationError")
			}
			eventContext.Data = data
		} else {
			// Empty body is allowed
		}
	}

	foundSomeQuery := false
	for _, paramDef := range options.Accepts {
		key := paramDef.Arg
		if paramDef.Http.Source == "body" {

			// Already parsed. Only used for OpenAPI Description

		} else if paramDef.Http.Source == "query" {

			var param interface{}
			paramSt := c.Query(key, "")
			switch paramDef.Type {
			case "string":
				param = paramSt
				break
			case "date":
				param, err = wst.ParseDate(paramSt)
				if err != nil {
					return wst.CreateError(fiber.ErrBadRequest, "INVALID_DATE", fiber.Map{"message": err.Error()}, "ValidationError")
				}
				break
			case "number":
				param, err = strconv.ParseFloat(paramSt, 64)
				if err != nil {
					return wst.CreateError(fiber.ErrBadRequest, "INVALID_NUMBER", fiber.Map{"message": err.Error()}, "ValidationError")
				}
				break
			}
			(*eventContext.Query)[key] = param

			if paramDef.Arg == "filter" {
				filterSt := (*eventContext.Query)[key].(string)
				filterMap := ParseFilter(filterSt)

				eventContext.Filter = filterMap
				continue
			}

			foundSomeQuery = true

		}
	}
	eventContext.Data = datasource.ReplaceObjectIds(eventContext.Data).(*wst.M)
	if foundSomeQuery {
		eventContext.Query = datasource.ReplaceObjectIds(eventContext.Query).(*wst.M)
	}

	return handler(eventContext)
}

func (loadedModel *Model) EnforceEx(token *BearerToken, objId string, action string, eventContext *EventContext) (error, bool) {

	if token != nil && token.User != nil && token.User.System == true {
		return nil, true
	}

	if token == nil {
		log.Printf("WARNING: Trying to enforce without token at %v.%v\n", loadedModel.Name, action)
	}

	var bearerUserIdSt string
	var targetObjId string

	if token == nil || token.User == nil {
		bearerUserIdSt = "_EVERYONE_"
		targetObjId = "*"
		if result, isPresent := loadedModel.authCache[bearerUserIdSt][targetObjId][action]; isPresent {
			if loadedModel.App.Debug || !result {
				log.Printf("DEBUG: Cache hit for %v.%v ---> %v\n", loadedModel.Name, action, result)
			}
			return nil, result
		}

	} else {

		bearerUserIdSt = fmt.Sprintf("%v", token.User.Id)
		targetObjId = objId

		if result, isPresent := loadedModel.authCache[bearerUserIdSt][targetObjId][action]; isPresent {
			if loadedModel.App.Debug || !result {
				log.Printf("DEBUG: Cache hit for %v.%v ---> %v\n", loadedModel.Name, action, result)
			}
			return nil, result
		}

		_, err := loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, "_EVERYONE_")
		if err != nil {
			return err, false
		}
		_, err = loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, "_AUTHENTICATED_")
		if err != nil {
			return err, false
		}
		for _, r := range token.Roles {
			_, err := loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, r.Name)
			if err != nil {
				return err, false
			}
		}
		err = loadedModel.Enforcer.SavePolicy()
		if err != nil {
			return err, false
		}

	}

	allow, exp, err := loadedModel.Enforcer.EnforceEx(bearerUserIdSt, targetObjId, action)

	if loadedModel.App.Debug {
		log.Println("Explain", exp)
	}
	if err != nil {
		loadedModel.authCache[bearerUserIdSt][targetObjId][action] = false
		return err, false
	}
	if allow {
		if loadedModel.authCache[bearerUserIdSt] == nil {
			loadedModel.authCache[bearerUserIdSt] = map[string]map[string]bool{}
		}
		if loadedModel.authCache[bearerUserIdSt][targetObjId] == nil {
			loadedModel.authCache[bearerUserIdSt][targetObjId] = map[string]bool{}
		}
		loadedModel.authCache[bearerUserIdSt][targetObjId][action] = true
		return nil, true

	}
	return fiber.ErrUnauthorized, false
}

/*
params:
	- relationDeepLevel: Starts at 1 (Root is 0)
*/
func (loadedModel *Model) mergeRelated(relationDeepLevel byte, documents *wst.A, includeItem wst.IncludeItem, baseContext *EventContext) error {

	if documents == nil {
		return nil
	}

	parentDocs := documents

	relationName := includeItem.Relation
	relation := (*loadedModel.Config.Relations)[relationName]
	relatedModelName := relation.Model
	relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

	parentModel := loadedModel
	parentRelationName := relationName

	if relatedLoadedModel == nil {
		log.Println()
		log.Printf("WARNING: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
		log.Println()
		return nil
	}

	//if relation.Options.SkipAuth && relationDeepLevel > 1 {
	// Only skip auth checking for relations above the level 1
	if relation.Options.SkipAuth {
		if loadedModel.App.Debug {
			log.Printf("DEBUG: SkipAuth %v.%v\n", loadedModel.Name, relationName)
		}
	} else {
		objId := "*"
		if len(*documents) == 1 {
			objId = (*documents)[0]["_id"].(primitive.ObjectID).Hex()
		}

		action := fmt.Sprintf("__get__%v", relationName)
		if loadedModel.App.Debug {
			log.Printf("DEBUG: Check %v.%v\n", loadedModel.Name, action)
		}
		err, allowed := loadedModel.EnforceEx(baseContext.Bearer, objId, action, baseContext)
		if err != nil && err != fiber.ErrUnauthorized {
			return err
		}
		if !allowed {
			for _, doc := range *documents {
				delete(doc, relationName)
			}
		}
	}

	if relatedLoadedModel.Datasource.Name != loadedModel.Datasource.Name {
		switch relation.Type {
		case "belongsTo", "hasOne", "hasMany":
			keyFrom := ""
			keyTo := ""
			switch relation.Type {
			case "belongsTo":
				keyFrom = *relation.PrimaryKey
				keyTo = *relation.ForeignKey
				break
			case "hasOne", "hasMany":
				keyFrom = *relation.ForeignKey
				keyTo = *relation.PrimaryKey
				break
			}

			var targetScope *wst.Filter
			if includeItem.Scope != nil {
				scopeValue := *includeItem.Scope
				targetScope = &scopeValue
			} else {
				targetScope = &wst.Filter{}
			}

			wasEmptyWhere := false
			if targetScope.Where == nil {
				targetScope.Where = &wst.Where{}
				wasEmptyWhere = true
			}

			cachedRelatedDocs := make([]InstanceA, len(*documents))
			localCache := map[string]InstanceA{}

			for documentIdx, document := range *documents {

				if wasEmptyWhere && relatedLoadedModel.Config.Cache.Datasource != "" /* && keyFrom == relatedLoadedModel.Config.Cache.Keys*/ {

					cacheDs, err := loadedModel.App.FindDatasource(relatedLoadedModel.Config.Cache.Datasource)
					if err != nil {
						return err
					}
					safeCacheDs := cacheDs.(*datasource.Datasource)

					//baseKey := fmt.Sprintf("%v:%v", safeCacheDs.Viper.GetString(safeCacheDs.Key+".database"), relatedLoadedModel.Config.Name)
					for _, keyGroup := range relatedLoadedModel.Config.Cache.Keys {

						if len(keyGroup) == 1 && keyGroup[0] == keyFrom {

							cacheKeyTo := fmt.Sprintf("%v:%v", keyFrom, document[keyTo])

							if localCache[cacheKeyTo] != nil {
								cachedRelatedDocs[documentIdx] = localCache[cacheKeyTo]
							} else {
								var cachedDocs *wst.A

								cacheLookups := &wst.A{wst.M{"$match": wst.M{keyFrom: cacheKeyTo}}}
								cachedDocs, err = safeCacheDs.FindMany(relatedLoadedModel.CollectionName, cacheLookups)
								if err != nil {
									return err
								}

								for _, cachedDoc := range *cachedDocs {
									cachedInstance := relatedLoadedModel.Build(cachedDoc, baseContext)
									if cachedRelatedDocs[documentIdx] == nil {
										cachedRelatedDocs[documentIdx] = InstanceA{}
									}
									cachedRelatedDocs[documentIdx] = append(cachedRelatedDocs[documentIdx], cachedInstance)
								}
								localCache[cacheKeyTo] = cachedRelatedDocs[documentIdx]
							}

						}
					}

				}

				relatedInstances := cachedRelatedDocs[documentIdx]
				if relatedInstances == nil {

					(*targetScope.Where)[keyFrom] = document[keyTo]
					if isSingleRelation(relation.Type) {
						targetScope.Limit = 1
					}

					var err error
					relatedInstances, err = relatedLoadedModel.FindMany(targetScope, baseContext)
					if err != nil {
						return err
					}
				} else {
					if loadedModel.App.Debug {
						log.Printf("Found cache for %v.%v[%v]\n", loadedModel.Name, relationName, documentIdx)
					}
				}

				if loadedModel.hasHiddenProperties {
					for _, relatedInstance := range relatedInstances {
						relatedInstance.HideProperties()
					}
				}

				switch {
				case isSingleRelation(relation.Type):
					if len(relatedInstances) > 0 {
						document[relationName] = relatedInstances[0]
					} else {
						document[relationName] = nil
					}
					break
				case isManyRelation(relation.Type):
					document[relationName] = relatedInstances
					break
				}

			}

			break
		}

	} else {

		if includeItem.Scope != nil && documents != nil && len(*documents) > 0 {
			if includeItem.Scope.Include != nil {

				for _, includeItem := range *includeItem.Scope.Include {
					relationName := includeItem.Relation
					//relation := (*loadedModel.Config.Relations)[relationName]
					_isSingleRelation := isSingleRelation(relation.Type)
					_isManyRelation := !_isSingleRelation
					//relatedModelName := relation.Model
					//relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

					nestedDocuments := make(wst.A, 0)

					for _, doc := range *documents {

						switch {
						case _isSingleRelation:
							if doc[parentRelationName] != nil {
								documentsValue := make(wst.A, 1)

								if relatedInstance, ok := doc[parentRelationName].(map[string]interface{}); ok {
									documentsValue[0] = wst.M{}
									for k, v := range relatedInstance {
										documentsValue[0][k] = v
									}
								} else if relatedInstance, ok := doc[parentRelationName].(wst.M); ok {
									documentsValue[0] = relatedInstance
								} else {
									log.Printf("WARNING: Invalid type for %v.%v %s\n", loadedModel.Name, relationName, doc[parentRelationName])
								}

								//documents = &documentsValue
								nestedDocuments = append(nestedDocuments, documentsValue...)
							}
							break
						case _isManyRelation:
							if doc[parentRelationName] != nil {

								if asGeneric, ok := doc[parentRelationName].([]interface{}); ok {
									relatedInstances := asGeneric
									nestedDocuments = append(nestedDocuments, *wst.AFromGenericSlice(&relatedInstances)...)
								} else if asPrimitiveA, ok := doc[parentRelationName].(primitive.A); ok {
									relatedInstances := asPrimitiveA
									nestedDocuments = append(nestedDocuments, *wst.AFromPrimitiveSlice(&relatedInstances)...)
								} else if asA, ok := doc[parentRelationName].(wst.A); ok {
									nestedDocuments = append(nestedDocuments, asA...)
								} else {
									log.Println("WARNING: unknown type for relation", relationName, "in", loadedModel.Name)
									continue
								}

							}
							break
						}

					}
					loadedModel := relatedLoadedModel
					if loadedModel.App.Debug {
						log.Printf("Dispatch nested relation %v.%v.%v (n=%v, m=%v)\n", parentModel.Name, parentRelationName, relationName, len(*parentDocs), len(nestedDocuments))
					}
					err := loadedModel.mergeRelated(relationDeepLevel+1, &nestedDocuments, includeItem, baseContext)
					if err != nil {
						return err
					}

				}
			}
		}
	}

	return nil
}

func (loadedModel *Model) Initialize() {
	if len(loadedModel.Config.Hidden) > 0 {
		loadedModel.hasHiddenProperties = true
	}
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
