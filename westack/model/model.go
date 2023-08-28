package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/fredyk/westack-go/westack/memorykv"
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
	Datasource    string     `json:"datasource"`
	Ttl           int        `json:"ttl"`
	Keys          [][]string `json:"keys"`
	ExcludeFields []string   `json:"excludeFields"`
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
	NilInstance      *Instance

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
	loadedModel.NilInstance = &Instance{
		Model: loadedModel,
		Id:    primitive.NilObjectID,
		data:  wst.NilMap,
		bytes: nil,
	}

	(*modelRegistry)[name] = loadedModel

	return loadedModel
}

type RegistryEntry struct {
	Name  string
	Model *Model
}

type buildCache struct {
	singleRelatedDocumentsById map[string]Instance
}

func NewBuildCache() *buildCache {
	return &buildCache{
		singleRelatedDocumentsById: make(map[string]Instance),
	}
}

func (loadedModel *Model) Build(data wst.M, sameLevelCache *buildCache, baseContext *EventContext) (Instance, error) {

	//if loadedModel.App.Stats.BuildsByModel[loadedModel.Name] == nil {
	//	loadedModel.App.Stats.BuildsByModel[loadedModel.Name] = map[string]float64{
	//		"count": 0,
	//		"time":  0,
	//	}
	//}
	//init := time.Now().UnixMilli()

	if data["id"] == nil {
		data["id"] = data["_id"]
		if data["id"] != nil {
			delete(data, "_id")
		}
	}

	var targetBaseContext = baseContext
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
	}

	modelInstance := Instance{
		Id:    data["id"],
		bytes: nil,
		data:  data,
		Model: loadedModel,
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
				fmt.Printf("ERROR: Model.Build() --> %v\n", err)
				return Instance{}, nil
			}
			if relatedModel != nil {
				switch relationConfig.Type {
				case "belongsTo", "hasOne":
					// Check if this parent instance is already in the same level cache
					// If so, check app.Viper.GetBool("strictSingleRelatedDocumentCheck") and if true, return an error
					// If not, print a warning
					strict := loadedModel.App.Viper.GetBool("strictSingleRelatedDocumentCheck")
					if v, ok := sameLevelCache.singleRelatedDocumentsById[modelInstance.Id.(primitive.ObjectID).Hex()]; ok {
						if strict {
							fmt.Printf("ERROR: Model.Build() --> Found multiple single related documents at %v.%v with the same parent %v.Id=%v\n", loadedModel.Name, relationName, loadedModel.Name, v.Id.(primitive.ObjectID).Hex())
							return Instance{}, fmt.Errorf("found multiple single related documents at %v.%v with the same parent %v.Id=%v", loadedModel.Name, relationName, loadedModel.Name, v.Id.(primitive.ObjectID).Hex())
						} else {
							fmt.Printf("WARNING: Model.Build() --> Found multiple single related documents at %v.%v with the same parent %v.Id=%v\n", loadedModel.Name, relationName, loadedModel.Name, v.Id.(primitive.ObjectID).Hex())
						}
					} else {
						sameLevelCache.singleRelatedDocumentsById[modelInstance.Id.(primitive.ObjectID).Hex()] = modelInstance
					}
					var relatedInstance Instance
					if asInstance, asInstanceOk := rawRelatedData.(Instance); asInstanceOk {
						relatedInstance = asInstance
					} else {
						relatedInstance, err = relatedModel.(*Model).Build(rawRelatedData.(wst.M), sameLevelCache, targetBaseContext)
						if err != nil {
							fmt.Printf("ERROR: Model.Build() --> %v\n", err)
							return Instance{}, err
						}
					}
					data[relationName] = &relatedInstance
				case "hasMany", "hasAndBelongsToMany":

					var result InstanceA
					if asInstanceList, asInstanceListOk := rawRelatedData.(InstanceA); asInstanceListOk {
						result = asInstanceList
					} else {
						result = make(InstanceA, len(rawRelatedData.(primitive.A)))
						for idx, v := range rawRelatedData.(primitive.A) {
							result[idx], err = relatedModel.(*Model).Build(v.(wst.M), sameLevelCache, targetBaseContext)
							if err != nil {
								fmt.Printf("ERROR: Model.Build() --> %v\n", err)
								return Instance{}, err
							}
						}
					}

					data[relationName] = result
				}
			}
		}
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &data
	eventContext.Instance = &modelInstance

	if loadedModel.DisabledHandlers["__operation__after_load"] != true {
		err := loadedModel.GetHandler("__operation__after_load")(eventContext)
		if err != nil {
			fmt.Println("Warning", err)
			return Instance{}, nil
		}
	}

	//loadedModel.App.Stats.BuildsByModel[loadedModel.Name]["count"]++
	//loadedModel.App.Stats.BuildsByModel[loadedModel.Name]["time"] += float64(time.Now().UnixMilli() - init)

	return modelInstance, nil
}

func ParseFilter(filter string) *wst.Filter {
	var filterMap *wst.Filter
	if filter != "" {
		_ = json.Unmarshal([]byte(filter), &filterMap)
	}
	return filterMap
}

func (loadedModel *Model) FindMany(filterMap *wst.Filter, baseContext *EventContext) Cursor {

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

	lookups, err := loadedModel.ExtractLookupsFromFilter(filterMap, baseContext.DisableTypeConversions)
	if err != nil {
		return newErrorCursor(err)
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Model = loadedModel
	if baseContext.OperationName != "" {
		eventContext.OperationName = baseContext.OperationName
	} else {
		eventContext.OperationName = wst.OperationNameFindMany
	}
	if loadedModel.DisabledHandlers["__operation__before_load"] != true {
		err := loadedModel.GetHandler("__operation__before_load")(eventContext)
		if err != nil {
			return newErrorCursor(err)
		}
		if eventContext.Result != nil {
			switch eventContext.Result.(type) {
			case *InstanceA:
				return newFixedLengthCursor(*eventContext.Result.(*InstanceA))
			case InstanceA:
				return newFixedLengthCursor(eventContext.Result.(InstanceA))
			case []*Instance:
				var result = make(InstanceA, len(eventContext.Result.([]*Instance)))
				for idx, v := range eventContext.Result.([]*Instance) {
					if v != nil {
						result[idx] = *v
					} else {
						result[idx] = Instance{}
					}
				}
				return newFixedLengthCursor(result)
			case wst.A:
				var result InstanceA
				sameLevelCache := NewBuildCache()
				if v, castOk := eventContext.Result.(wst.A); castOk {
					result = make(InstanceA, len(v))
					for idx, v := range v {
						result[idx], err = loadedModel.Build(v, sameLevelCache, targetBaseContext)
						if err != nil {
							return newErrorCursor(err)
						}
					}
				} else {
					result = make(InstanceA, len(v))
					for idx, v := range v {
						result[idx], err = loadedModel.Build(v, sameLevelCache, targetBaseContext)
						if err != nil {
							return newErrorCursor(err)
						}
					}
				}
				return newFixedLengthCursor(result)
			default:
				return newErrorCursor(fmt.Errorf("invalid eventContext.Result type, expected InstanceA or []wst.M; found %T", eventContext.Result))
			}
		}
	}
	//for key := range *loadedModel.Config.Relations {
	//	delete(finalData, key)
	//}

	dsCursor, err := loadedModel.Datasource.FindMany(loadedModel.CollectionName, lookups)
	if err != nil {
		return newErrorCursor(err)
	}
	if dsCursor == nil {
		return newErrorCursor(fmt.Errorf("invalid query result"))
	}

	var targetInclude *wst.Include
	if filterMap != nil && filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
	}

	var results = make(chan *Instance)
	var cursor = NewChannelCursor(results).(*ChannelCursor)
	cursor.UsedPipeline = lookups
	//var cursor = newMongoCursor(context.Background(), dsCursor).(*MongoCursor)

	go func() {
		err := func() error {
			defer func(cursor Cursor) {
				err := cursor.Close()
				if err != nil {
					fmt.Printf("ERROR: Could not close cursor: %v\n", err)
				}
			}(cursor)
			defer func(dsCursor datasource.MongoCursorI, ctx context.Context) {
				err := dsCursor.Close(ctx)
				if err != nil {
					fmt.Printf("ERROR: Could not close cursor: %v\n", err)
				}
			}(dsCursor, context.Background())
			disabledCache := loadedModel.App.Viper.GetBool("disableCache")
			sameLevelCache := NewBuildCache()
			var safeCacheDs *datasource.Datasource
			documentsToCacheByKey := make(map[string]wst.A)
			for dsCursor.Next(context.Background()) {
				var document wst.M
				err := dsCursor.Decode(&document)
				if err != nil {
					return err
				}

				if targetInclude != nil {
					for _, includeItem := range *targetInclude {
						relationName := includeItem.Relation
						relation := (*loadedModel.Config.Relations)[relationName]
						relatedModelName := relation.Model
						relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]
						if relatedLoadedModel == nil {
							return fmt.Errorf("related model not found")
						}

						err := loadedModel.mergeRelated(1, &wst.A{document}, includeItem, targetBaseContext)
						if err != nil {
							return err
						}

					}
				}

				inst, err := loadedModel.Build(document, sameLevelCache, targetBaseContext)
				if err != nil {
					return err
				}
				results <- &inst
				var includePrefix = ""
				if targetInclude != nil {
					marshalledTargetInclude, err := json.Marshal(targetInclude)
					if err != nil {
						return err
					}
					includePrefix = fmt.Sprintf("_inc_%s_", marshalledTargetInclude)
				}
				if filterMap != nil && filterMap.Where != nil {
					marshalledWhere, err := json.Marshal(filterMap.Where)
					if err != nil {
						return err
					}
					includePrefix += fmt.Sprintf("_whr_%s_", marshalledWhere)
				}
				if loadedModel.Config.Cache.Datasource != "" && !disabledCache {

					// Dont cache if include is set
					cacheDs, err := loadedModel.App.FindDatasource(loadedModel.Config.Cache.Datasource)
					if err != nil {
						return err
					}

					safeCacheDs = cacheDs.(*datasource.Datasource)
					for _, keyGroup := range loadedModel.Config.Cache.Keys {
						toCache := wst.CopyMap(document)

						// Remove fields that are not cacheable
						if loadedModel.Config.Cache.ExcludeFields != nil {
							for _, field := range loadedModel.Config.Cache.ExcludeFields {
								if _, ok := toCache[field]; ok {
									delete(toCache, field)
								}
							}
						}

						isUniqueId := false
						if len(keyGroup) == 1 && keyGroup[0] == "_id" {
							isUniqueId = true
						}
						canonicalId := includePrefix
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

						if isUniqueId {
							err3 := insertCacheEntries(safeCacheDs, loadedModel, wst.M{"_entries": wst.A{toCache}, "_redId": canonicalId})
							if err3 != nil {
								return err3
							}
							err2 := doExpireCacheKey(safeCacheDs, loadedModel, canonicalId)
							if err2 != nil {
								return err2
							}
						} else {
							documentsToCacheByKey[canonicalId] = append(documentsToCacheByKey[canonicalId], toCache)
						}
					}

				}

			}

			for key, documents := range documentsToCacheByKey {
				err := insertCacheEntries(safeCacheDs, loadedModel, wst.M{"_entries": documents, "_redId": key})
				if err != nil {
					return err
				}
				err = doExpireCacheKey(safeCacheDs, loadedModel, key)
				if err != nil {
					return err
				}
			}

			return nil
		}()
		if err != nil {
			if loadedModel.App.Debug {
				log.Println("CACHE ERROR:", err)
			}
			cursor.Error(err)
		}
	}()

	return cursor
}

func insertCacheEntries(safeCacheDs *datasource.Datasource, loadedModel *Model, toCache wst.M) error {
	cached, err := safeCacheDs.Create(loadedModel.CollectionName, &toCache)
	if err != nil {
		return err
	}
	if loadedModel.App.Debug {
		fmt.Printf("[DEBUG] cached %v(len=%v) in memorykv\n", toCache["_redId"], len(toCache["_entries"].(wst.A)))
		fmt.Printf("[DEBUG] cached doc %v in memorykv\n", cached)
	}
	return nil
}

func doExpireCacheKey(safeCacheDs *datasource.Datasource, loadedModel *Model, canonicalId string) error {
	connectorName := safeCacheDs.SubViper.GetString("connector")
	switch connectorName {
	case "redis":
		return errors.New("redis cache connector not implemented")
	case "memorykv":
		db := safeCacheDs.Db.(memorykv.MemoryKvDb)
		bucket := db.GetBucket(loadedModel.CollectionName)
		if loadedModel.App.Debug {
			log.Println("CACHING", loadedModel.Name)
		}
		if loadedModel.App.Debug {
			log.Println("CACHING CANONICAL ID", canonicalId)
		}
		ttl := time.Duration(loadedModel.Config.Cache.Ttl) * time.Second
		if loadedModel.App.Debug {
			fmt.Printf("[DEBUG] trying to expire %v in %v seconds\n", canonicalId, ttl)
		}
		err := bucket.Expire(canonicalId, ttl)
		if err != nil {
			return err
		}
		if loadedModel.App.Debug {
			fmt.Printf("[DEBUG] expiring %v in %v seconds\n", canonicalId, ttl)
		}
	default:
		return errors.New(fmt.Sprintf("Unsupported cache connector %v", connectorName))
	}
	return nil
}

func (loadedModel *Model) Count(filterMap *wst.Filter, baseContext *EventContext) (int64, error) {
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

	lookups, err := loadedModel.ExtractLookupsFromFilter(filterMap, baseContext.DisableTypeConversions)
	if err != nil {
		return 0, err
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Model = loadedModel
	if baseContext.OperationName != "" {
		eventContext.OperationName = baseContext.OperationName
	} else {
		eventContext.OperationName = wst.OperationNameCount
	}

	eventContext.DisableTypeConversions = baseContext.DisableTypeConversions

	eventContext.Filter = filterMap

	count, err := loadedModel.Datasource.Count(loadedModel.CollectionName, lookups)
	if err != nil {
		return 0, err
	}

	return count, nil

}

func (loadedModel *Model) FindOne(filterMap *wst.Filter, baseContext *EventContext) (*Instance, error) {

	if filterMap == nil {
		filterMap = &wst.Filter{}
	}
	filterMap.Limit = 1

	return loadedModel.FindMany(filterMap, baseContext).Next()
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

	baseContext.OperationName = wst.OperationNameFindById
	instances, err := loadedModel.FindMany(filterMap, baseContext).All()
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
				// how to test this???
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
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
	}
	if !baseContext.DisableTypeConversions {
		datasource.ReplaceObjectIds(finalData)
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &finalData
	eventContext.Model = loadedModel
	eventContext.IsNewInstance = true
	eventContext.OperationName = wst.OperationNameCreate
	if loadedModel.DisabledHandlers["__operation__before_save"] != true {
		err := loadedModel.GetHandler("__operation__before_save")(eventContext)
		if err != nil {
			return nil, err
		}
		if eventContext.Result != nil {
			switch eventContext.Result.(type) {
			case *Instance:
				return eventContext.Result.(*Instance), nil
			case Instance:
				v := eventContext.Result.(Instance)
				return &v, nil
			case wst.M:
				v, err := loadedModel.Build(eventContext.Result.(wst.M), NewBuildCache(), targetBaseContext)
				if err != nil {
					return nil, err
				}
				return &v, nil
			default:
				return nil, fmt.Errorf("invalid eventContext.Result type, expected *Instance, Instance or wst.M; found %T", eventContext.Result)
			}
		}
	}
	for key := range *loadedModel.Config.Relations {
		delete(finalData, key)
	}
	document, err := loadedModel.Datasource.Create(loadedModel.CollectionName, &finalData)

	if err != nil {
		return nil, err
	} else {
		result, err := loadedModel.Build(*document, NewBuildCache(), eventContext)
		if err != nil {
			return nil, err
		}
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

func (loadedModel *Model) DeleteById(id interface{}) (datasource.DeleteResult, error) {

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
			fmt.Println(fmt.Sprintf("WARNING: Invalid input for Model.DeleteById() <- %s", id))
		}
	}
	//TODO: Invoke hook for __operation__before_delete and __operation__after_delete
	return loadedModel.Datasource.DeleteById(loadedModel.CollectionName, finalId)
}

func (loadedModel *Model) DeleteMany(where *wst.Where, systemContext *EventContext) (result datasource.DeleteResult, err error) {
	if where == nil {
		return result, errors.New("where cannot be nil")
	}
	if len(*where) == 0 {
		return result, errors.New("where cannot be empty")
	}
	whereLookups := &wst.A{
		{
			"$match": wst.M(*where),
		},
	}
	return loadedModel.Datasource.DeleteMany(loadedModel.CollectionName, whereLookups)
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
					fmt.Println("WARNING: Stop handling on error", currentHandlerError)
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

var handlerMutex = sync.Mutex{}

func (loadedModel *Model) GetHandler(event string) func(eventContext *EventContext) error {
	res := loadedModel.eventHandlers[event]
	if res == nil {
		handlerMutex.Lock()
		loadedModel.DisabledHandlers[event] = true
		handlerMutex.Unlock()
		res = func(eventContext *EventContext) error {
			if loadedModel.App.Debug {
				fmt.Println("no handler found for ", loadedModel.Name, ".", event)
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
