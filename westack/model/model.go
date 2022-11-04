package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
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

type RegistryEntry struct {
	Name  string
	Model *Model
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
		// check if data is a struct
		if reflect.TypeOf(data).Kind() == reflect.Struct {
			bytes, err := bson.Marshal(data)
			if err != nil {
				return nil, err
			}
			err = bson.Unmarshal(bytes, &finalData)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New(fmt.Sprintf("Invalid input for Model.Create() <- %s", data))
		}
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
	Claims jwt.MapClaims
}

type EphemeralData wst.M

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
