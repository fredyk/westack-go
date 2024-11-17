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
	"github.com/fredyk/westack-go/v2/memorykv"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
)

type Model interface {
	FindMany(filterMap *wst.Filter, currentContext *EventContext) Cursor
	FindById(id interface{}, filterMap *wst.Filter, baseContext *EventContext) (Instance, error)
	Create(data interface{}, currentContext *EventContext) (Instance, error)
	Count(filterMap *wst.Filter, currentContext *EventContext) (int64, error)
	DeleteById(id interface{}, currentContext *EventContext) (datasource.DeleteResult, error)
	UpdateById(id interface{}, data interface{}, currentContext *EventContext) (Instance, error)
	GetConfig() *Config
	GetName() string
}

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
	Name        string                `json:"name"`
	Plural      string                `json:"plural"`
	Base        string                `json:"base"`
	Public      bool                  `json:"public"`
	Properties  map[string]Property   `json:"properties"`
	Relations   *map[string]*Relation `json:"relations"`
	Hidden      []string              `json:"hidden"`
	Protected   []string              `json:"protected"`
	Validations []Validation          `json:"validations"`
	Casbin      CasbinConfig          `json:"casbin"`
	Cache       CacheConfig           `json:"cache"`
	Mongo       MongoConfig           `json:"mongo"`
}

type Validation struct {
	If         map[string]Condition  `json:"if"`
	Then       *Validation           `json:"then"`
	AllOf      []Validation          `json:"allOf"`
	OneOf      []Validation          `json:"oneOf"`
	Properties map[string]Validation `json:"properties"`
	NotEmpty   bool                  `json:"notEmpty"`
}

type Condition struct {
	Equals      interface{}   `json:"equals"`
	NotEquals   interface{}   `json:"notEquals"`
	Contains    []interface{} `json:"contains"`
	NotContains []interface{} `json:"notContains"`
	Exists      bool          `json:"exists"`
	NotExists   bool          `json:"notExists"`
	Empty       bool          `json:"empty"`
	NotEmpty    bool          `json:"notEmpty"`
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
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type StatefulModel struct {
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
	NilInstance      *StatefulInstance

	eventHandlers    map[string]func(eventContext *EventContext) error
	modelRegistry    *map[string]*StatefulModel
	remoteMethodsMap map[string]*OperationItem

	authCache           map[string]map[string]map[string]bool
	hasHiddenProperties bool
	pendingOperations   []pendingOperationEntry
}

type pendingOperationEntry struct {
	eventKey    string
	handler     func(eventContext *EventContext) error
	operationId int64
}

func (loadedModel *StatefulModel) GetConfig() *Config {
	return loadedModel.Config
}

func (loadedModel *StatefulModel) GetName() string {
	return loadedModel.Name
}

func (loadedModel *StatefulModel) GetModelRegistry() *map[string]*StatefulModel {
	return loadedModel.modelRegistry
}

func New(config *Config, modelRegistry *map[string]*StatefulModel) Model {
	name := config.Name
	collectionName := config.Mongo.Collection
	if collectionName == "" {
		collectionName = name
	}
	loadedModel := &StatefulModel{
		Name:             name,
		CollectionName:   collectionName,
		Config:           config,
		DisabledHandlers: map[string]bool{},

		modelRegistry:     modelRegistry,
		eventHandlers:     map[string]func(eventContext *EventContext) error{},
		remoteMethodsMap:  map[string]*OperationItem{},
		authCache:         map[string]map[string]map[string]bool{},
		pendingOperations: []pendingOperationEntry{},
	}
	loadedModel.NilInstance = &StatefulInstance{
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
	Model *StatefulModel
}

func (loadedModel *StatefulModel) Build(data wst.M, currentContext *EventContext) (Instance, error) {

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

	var targetBaseContext = FindBaseContext(currentContext)

	modelInstance := StatefulInstance{
		Id:    data["id"],
		bytes: nil,
		data:  data,
		Model: loadedModel,
	}

	beforeBuildEventContext := &EventContext{
		BaseContext:   targetBaseContext,
		Data:          &data,
		Model:         loadedModel,
		ModelID:       modelInstance.Id,
		OperationName: currentContext.OperationName,
	}

	if !loadedModel.DisabledHandlers["__operation__before_build"] {
		err := loadedModel.GetHandler("__operation__before_build")(beforeBuildEventContext)
		if err != nil {
			return &StatefulInstance{}, fmt.Errorf("error in __operation__before_build: %v", err)
		}
	}

	for relationName, relationConfig := range *loadedModel.Config.Relations {
		if data[relationName] != nil && relationConfig.Type != "" {
			rawRelatedData := data[relationName]
			var err error
			relatedModel, _ := loadedModel.App.FindModel(relationConfig.Model)
			if relatedModel != nil {
				switch relationConfig.Type {
				case "belongsTo", "hasOne":
					var relatedInstance Instance
					if asInstance, asInstanceOk := rawRelatedData.(*StatefulInstance); asInstanceOk {
						relatedInstance = asInstance
					} else {
						relatedInstance, err = relatedModel.(*StatefulModel).Build(rawRelatedData.(wst.M), targetBaseContext)
						if err != nil {
							fmt.Printf("[ERROR] Model.Build() --> %v\n", err)
							return &StatefulInstance{}, err
						}
					}
					data[relationName] = relatedInstance
				case "hasMany", "hasAndBelongsToMany":

					var result InstanceA
					if asInstanceList, asInstanceListOk := rawRelatedData.(InstanceA); asInstanceListOk {
						result = asInstanceList
					} else {
						result = make(InstanceA, len(rawRelatedData.(primitive.A)))
						for idx, v := range rawRelatedData.(primitive.A) {
							result[idx], err = relatedModel.(*StatefulModel).Build(v.(wst.M), targetBaseContext)
							if err != nil {
								fmt.Printf("[ERROR] Model.Build() --> %v\n", err)
								return &StatefulInstance{}, err
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

	/* trunk-ignore(golangci-lint/gosimple) */
	if loadedModel.DisabledHandlers["__operation__after_load"] != true {
		err := loadedModel.GetHandler("__operation__after_load")(eventContext)
		if err != nil {
			return &StatefulInstance{}, err
		}
	}

	//loadedModel.App.Stats.BuildsByModel[loadedModel.Name]["count"]++
	//loadedModel.App.Stats.BuildsByModel[loadedModel.Name]["time"] += float64(time.Now().UnixMilli() - init)

	return &modelInstance, nil
}

func ParseFilter(filter string) *wst.Filter {
	var filterMap *wst.Filter
	if filter != "" {
		_ = json.Unmarshal([]byte(filter), &filterMap)
	}
	return filterMap
}

func (loadedModel *StatefulModel) FindMany(filterMap *wst.Filter, currentContext *EventContext) Cursor {

	currentContext = existingOrEmpty(currentContext)
	targetBaseContext := FindBaseContext(currentContext)

	lookups, err := loadedModel.ExtractLookupsFromFilter(filterMap, currentContext.DisableTypeConversions)
	if err != nil {
		return NewErrorCursor(err)
	}

	currentOperationContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	currentOperationContext.Model = loadedModel
	if currentContext.OperationName != "" {
		currentOperationContext.OperationName = currentContext.OperationName
	} else {
		currentOperationContext.OperationName = wst.OperationNameFindMany
	}
	if loadedModel.DisabledHandlers["__operation__before_load"] != true {
		err := loadedModel.GetHandler("__operation__before_load")(currentOperationContext)
		if err != nil {
			return NewErrorCursor(err)
		}
		if currentOperationContext.Result != nil {
			switch currentOperationContext.Result.(type) {
			case *InstanceA:
				return newFixedLengthCursor(*currentOperationContext.Result.(*InstanceA))
			case InstanceA:
				return newFixedLengthCursor(currentOperationContext.Result.(InstanceA))
			case []*StatefulInstance:
				return newFixedLengthCursor(copyInstanceSlice(currentOperationContext.Result.([]*StatefulInstance)))
			case wst.A:
				var result InstanceA
				result, err = loadedModel.buildInstanceAFromA(currentOperationContext.Result.(wst.A), currentOperationContext)
				if err != nil {
					return NewErrorCursor(err)
				}
				return newFixedLengthCursor(result)
			default:
				return NewErrorCursor(fmt.Errorf("invalid eventContext.Result type, expected InstanceA or []wst.M; found %T", currentOperationContext.Result))
			}
		}
	}
	//for key := range *loadedModel.Config.Relations {
	//	delete(finalData, key)
	//}

	dsCursor, err := loadedModel.Datasource.FindMany(loadedModel.CollectionName, lookups)
	if err != nil {
		return NewErrorCursor(err)
	}
	if dsCursor == nil {
		return NewErrorCursor(fmt.Errorf("invalid query result"))
	}

	var targetInclude *wst.Include
	if filterMap != nil && filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
	}

	var results = make(chan Instance)
	var cursor = NewChannelCursor(results).(*ChannelCursor)
	cursor.UsedPipeline = lookups
	//var cursor = newMongoCursor(context.Background(), dsCursor).(*MongoCursor)

	go loadedModel.dispatchFindManyResults(cursor, dsCursor, targetInclude, currentOperationContext, results, filterMap)

	return cursor
}

func FindBaseContext(currentContext *EventContext) *EventContext {
	var targetBaseContext = currentContext
	for {
		if targetBaseContext.BaseContext != nil {
			targetBaseContext = targetBaseContext.BaseContext
		} else {
			break
		}
	}
	return targetBaseContext
}

func (loadedModel *StatefulModel) buildInstanceAFromA(v wst.A, targetBaseContext *EventContext) (result InstanceA, err error) {
	result = make(InstanceA, len(v))
	for idx, v := range v {
		result[idx], err = loadedModel.Build(v, targetBaseContext)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func copyInstanceSlice(src []*StatefulInstance) InstanceA {
	var result = make(InstanceA, len(src))
	for idx, v := range src {
		result[idx] = v
	}
	return result
}

func insertCacheEntries(safeCacheDs *datasource.Datasource, loadedModel *StatefulModel, toCache wst.M) error {
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

func doExpireCacheKey(safeCacheDs *datasource.Datasource, loadedModel *StatefulModel, canonicalId string) (err error) {
	connectorName := safeCacheDs.SubViper.GetString("connector")
	switch connectorName {
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
		err = bucket.Expire(canonicalId, ttl)
		if loadedModel.App.Debug {
			fmt.Printf("[DEBUG] expiring %v in %v seconds, err=%v\n", canonicalId, ttl, err)
		}
	default:
		return errors.New(fmt.Sprintf("Unsupported cache connector %v", connectorName))
	}
	return err
}

func existingOrEmpty[T any](existing *T) *T {
	if existing != nil {
		return existing
	}
	return new(T)
}

func (loadedModel *StatefulModel) Count(filterMap *wst.Filter, currentContext *EventContext) (int64, error) {
	currentContext = existingOrEmpty(currentContext)
	var targetBaseContext = FindBaseContext(currentContext)

	lookups, err := loadedModel.ExtractLookupsFromFilter(filterMap, currentContext.DisableTypeConversions)
	if err != nil {
		return 0, err
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Model = loadedModel
	if currentContext.OperationName != "" {
		eventContext.OperationName = currentContext.OperationName
	} else {
		eventContext.OperationName = wst.OperationNameCount
	}

	eventContext.DisableTypeConversions = currentContext.DisableTypeConversions

	eventContext.Filter = filterMap

	count, err := loadedModel.Datasource.Count(loadedModel.CollectionName, lookups)
	if err != nil {
		return 0, err
	}

	return count, nil

}

func (loadedModel *StatefulModel) FindOne(filterMap *wst.Filter, baseContext *EventContext) (Instance, error) {

	if filterMap == nil {
		filterMap = &wst.Filter{}
	}
	filterMap.Limit = 1

	return loadedModel.FindMany(filterMap, baseContext).Next()
}

func (loadedModel *StatefulModel) FindById(id interface{}, filterMap *wst.Filter, baseContext *EventContext) (Instance, error) {
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
		return instances[0], nil
	}

	return nil, nil
}

func (loadedModel *StatefulModel) Create(data interface{}, currentContext *EventContext) (Instance, error) {

	var finalData wst.M

	if m, ok := data.(map[string]interface{}); ok {
		finalData = wst.M{}
		for key, value := range m {
			finalData[key] = value
		}
	} else if m, ok := data.(*map[string]interface{}); ok {
		finalData = wst.M{}
		for key, value := range *m {
			finalData[key] = value
		}
	} else if m, ok := data.(wst.M); ok {
		finalData = m
	} else if m, ok := data.(*wst.M); ok {
		finalData = *m
	} else if value, ok := data.(StatefulInstance); ok {
		finalData = (&value).ToJSON()
	} else if value, ok := data.(*StatefulInstance); ok {
		finalData = value.ToJSON()
	} else if value, ok := data.(*Instance); ok {
		finalData = (*value).ToJSON()
	} else {
		// check if data is a struct
		if reflect.TypeOf(data).Kind() == reflect.Struct {
			bytes, err := bson.MarshalWithRegistry(loadedModel.App.Bson.Registry, data)
			if err != nil {
				return nil, err
			}
			err = bson.UnmarshalWithRegistry(loadedModel.App.Bson.Registry, bytes, &finalData)
			if err != nil {
				// how to test this???
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid input for Model.Create() <- %s", cast.ToString(data))
		}
	}

	currentContext = existingOrEmpty(currentContext)
	var targetBaseContext = FindBaseContext(currentContext)
	if !currentContext.DisableTypeConversions {
		_, err := datasource.ReplaceObjectIds(finalData)
		if err != nil {
			return nil, err
		}
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
			case *StatefulInstance, Instance:
				return eventContext.Result.(*StatefulInstance), nil
			case *Instance:
				return (*eventContext.Result.(*Instance)).(*StatefulInstance), nil
			case StatefulInstance:
				v := eventContext.Result.(StatefulInstance)
				return &v, nil
			case wst.M:
				v, err := loadedModel.Build(eventContext.Result.(wst.M), targetBaseContext)
				if err != nil {
					return nil, err
				}
				return v, nil
			default:
				return nil, fmt.Errorf("invalid eventContext.Result type, expected Instance, Instance or wst.M; found %T", eventContext.Result)
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
		result, err := loadedModel.Build(*document, eventContext)
		if err != nil {
			return nil, err
		}
		result.(*StatefulInstance).HideProperties()
		eventContext.Instance = result.(*StatefulInstance)
		if loadedModel.DisabledHandlers["__operation__after_save"] != true {
			err := loadedModel.GetHandler("__operation__after_save")(eventContext)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}

}

func (loadedModel *StatefulModel) DeleteById(id interface{}, currentContext *EventContext) (datasource.DeleteResult, error) {

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
			fmt.Println(fmt.Sprintf("[WARNING] Invalid input for Model.DeleteById() <- %s", id))
		}
	}

	currentContext = existingOrEmpty(currentContext)
	var targetBaseContext = FindBaseContext(currentContext)
	eventContext := &EventContext{
		BaseContext:   targetBaseContext,
		ModelID:       finalId,
		OperationName: wst.OperationNameDeleteById,
	}
	if loadedModel.DisabledHandlers["__operation__before_delete"] != true {
		err := loadedModel.GetHandler("__operation__before_delete")(eventContext)
		if err != nil {
			return datasource.DeleteResult{}, err
		}
	}

	deleteResult, err := loadedModel.Datasource.DeleteById(loadedModel.CollectionName, finalId)
	if err != nil {
		return deleteResult, err
	}
	if loadedModel.DisabledHandlers["__operation__after_delete"] != true {
		err = loadedModel.GetHandler("__operation__after_delete")(eventContext)
	}
	return deleteResult, err
}

func (loadedModel *StatefulModel) DeleteMany(where *wst.Where, currentContext *EventContext) (result datasource.DeleteResult, err error) {
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
	currentContext = existingOrEmpty(currentContext)
	var targetBaseContext = FindBaseContext(currentContext)
	if !currentContext.DisableTypeConversions {
		_, err := datasource.ReplaceObjectIds(&(*whereLookups)[0])
		if err != nil {
			return result, err
		}
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	//eventContext.Data = &finalData
	eventContext.Model = loadedModel
	eventContext.IsNewInstance = false
	eventContext.OperationName = wst.OperationNameDeleteMany

	return loadedModel.Datasource.DeleteMany(loadedModel.CollectionName, whereLookups)
}

func (loadedModel *StatefulModel) UpdateById(id interface{}, data interface{}, currentContext *EventContext) (Instance, error) {

	var finalId interface{}
	/* trunk-ignore(golangci-lint/gosimple) */
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
			fmt.Println(fmt.Sprintf("[WARNING] Invalid input for Model.UpdateById() <- %s", id))
		}
	}

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
	case StatefulInstance:
		value := data.(StatefulInstance)
		finalData = (&value).ToJSON()
		break
	case *StatefulInstance:
		finalData = data.(*StatefulInstance).ToJSON()
		break
	default:
		// check if data is a struct
		if reflect.TypeOf(data).Kind() == reflect.Struct {
			bytes, err := bson.MarshalWithRegistry(loadedModel.App.Bson.Registry, data)
			if err != nil {
				return nil, err
			}
			err = bson.UnmarshalWithRegistry(loadedModel.App.Bson.Registry, bytes, &finalData)
			if err != nil {
				// how to test this???
				return nil, err
			}
		} else {
			return nil, errors.New(fmt.Sprintf("Invalid input for Model.UpdateById() <- %s", data))
		}
	}

	currentContext = existingOrEmpty(currentContext)
	var targetBaseContext = FindBaseContext(currentContext)
	if !currentContext.DisableTypeConversions {
		_, err := datasource.ReplaceObjectIds(finalData)
		if err != nil {
			return nil, err
		}
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &finalData
	eventContext.Model = loadedModel
	eventContext.IsNewInstance = false
	eventContext.OperationName = wst.OperationNameUpdateById

	if loadedModel.DisabledHandlers["__operation__before_save"] != true {
		err := loadedModel.GetHandler("__operation__before_save")(eventContext)
		if err != nil {
			return nil, err
		}
		if eventContext.Result != nil {
			switch eventContext.Result.(type) {
			case *StatefulInstance, Instance:
				return eventContext.Result.(*StatefulInstance), nil
			case *Instance:
				return (*eventContext.Result.(*Instance)).(*StatefulInstance), nil
			case StatefulInstance:
				v := eventContext.Result.(StatefulInstance)
				return &v, nil
			case wst.M:
				v, err := loadedModel.Build(eventContext.Result.(wst.M), targetBaseContext)
				if err != nil {
					return nil, err
				}
				return v, nil
			default:
				return nil, fmt.Errorf("invalid eventContext.Result type, expected Instance, Instance or wst.M; found %T", eventContext.Result)
			}
		}
	}

	document, err := loadedModel.Datasource.UpdateById(loadedModel.CollectionName, finalId, &finalData)
	if err != nil {
		return nil, err
	} else {
		result, err := loadedModel.Build(*document, eventContext)
		if err != nil {
			return nil, err
		}
		result.(*StatefulInstance).HideProperties()
		eventContext.Instance = result.(*StatefulInstance)
		if !loadedModel.DisabledHandlers["__operation__after_save"] {
			err := loadedModel.GetHandler("__operation__after_save")(eventContext)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
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

type BearerAccount struct {
	Id     interface{}
	Data   interface{}
	System bool
}

type BearerRole struct {
	Name string
}

type BearerToken struct {
	Account *BearerAccount
	Roles   []BearerRole
	Raw     string
	Claims  jwt.MapClaims
}

type EphemeralData wst.M

var operationCounter int64 = 1

func wrapEventHandler(model *StatefulModel, eventKey string, handler func(eventContext *EventContext) error) func(eventContext *EventContext) error {
	currentHandler := model.eventHandlers[eventKey]
	if currentHandler != nil {
		newHandler := handler
		handler = func(eventContext *EventContext) error {
			currentHandlerError := currentHandler(eventContext)
			if currentHandlerError != nil {
				if model.App.Debug {
					fmt.Println("[WARNING] Stop handling on error", currentHandlerError)
					debug.PrintStack()
				}
				return currentHandlerError
			} else {
				return newHandler(eventContext)
			}
		}
	}
	wrappedHandler := func(eventContext *EventContext) error {
		baseContext := FindBaseContext(eventContext)
		if baseContext != nil {
			if baseContext.OperationId == 0 {
				baseContext.OperationId = operationCounter
				operationCounter++
			}

			// First, process new callbacks and remove them
			err := dispatchPendingOperations(eventContext, model, eventKey, baseContext)
			if err != nil {
				return err
			}
		}

		return handler(eventContext)
	}
	return wrappedHandler
}

func dispatchPendingOperations(eventContext *EventContext, model *StatefulModel, eventKey string, baseContext *EventContext) error {
	n := 0
	for i, pendingOperation := range model.pendingOperations {
		if pendingOperation.eventKey == eventKey && pendingOperation.operationId == baseContext.OperationId {
			err := pendingOperation.handler(eventContext)
			if err != nil {
				return err
			}
			// splice taking into account the number of elements processed
			model.pendingOperations = append(model.pendingOperations[:i-n], model.pendingOperations[i+1:]...)
			n++
		}
	}
	return nil
}

func (loadedModel *StatefulModel) QueueOperation(operation string, eventContext *EventContext, fn func(nextCtx *EventContext) error) {
	eventKey := mapOperationName(operation)
	loadedModel.DisabledHandlers[eventKey] = false
	loadedModel.pendingOperations = append(loadedModel.pendingOperations, pendingOperationEntry{
		eventKey:    eventKey,
		handler:     fn,
		operationId: FindBaseContext(eventContext).OperationId,
	})
}

func (loadedModel *StatefulModel) On(event string, handler func(eventContext *EventContext) error) {
	loadedModel.eventHandlers[event] = wrapEventHandler(loadedModel, event, handler)
}

func (loadedModel *StatefulModel) Observe(operation string, handler func(eventContext *EventContext) error) {
	loadedModel.On(mapOperationName(operation), handler)
}

func mapOperationName(operation string) string {
	return "__operation__" + strings.ReplaceAll(strings.TrimSpace(operation), " ", "_")
}

var handlerMutex = sync.Mutex{}

func (loadedModel *StatefulModel) GetHandler(event string) func(eventContext *EventContext) error {
	res := loadedModel.eventHandlers[event]
	if res == nil {
		handlerMutex.Lock()
		loadedModel.DisabledHandlers[event] = true
		handlerMutex.Unlock()
		res = func(eventContext *EventContext) error {
			// First, process new callbacks and remove them
			err := dispatchPendingOperations(eventContext, loadedModel, event, FindBaseContext(eventContext))
			if err != nil {
				return err
			}
			if loadedModel.App.Debug {
				fmt.Println("no handler found for ", loadedModel.Name, ".", event)
			}
			return nil
		}
	}
	return res
}

func (loadedModel *StatefulModel) Initialize() {
	if len(loadedModel.Config.Hidden) > 0 {
		loadedModel.hasHiddenProperties = true
	}
}

func (loadedModel *StatefulModel) dispatchFindManyResults(cursor *ChannelCursor, dsCursor datasource.MongoCursorI, targetInclude *wst.Include, currentContext *EventContext, results chan Instance, filterMap *wst.Filter) {
	err := func() error {
		defer func(cursor Cursor) {
			//// wait 16ms for error
			//time.Sleep(1600 * time.Millisecond)
			err := cursor.Close()
			if err != nil {
				fmt.Printf("[ERROR] Could not close cursor: %v\n", err)
			}
		}(cursor)
		defer func(dsCursor datasource.MongoCursorI, ctx context.Context) {
			err := dsCursor.Close(ctx)
			if err != nil {
				fmt.Printf("[ERROR] Could not close cursor: %v\n", err)
			}
		}(dsCursor, context.Background())
		disabledCache := loadedModel.App.Viper.GetBool("disableCache")
		var safeCacheDs *datasource.Datasource
		if loadedModel.Config.Cache.Datasource != "" && !disabledCache {

			// Dont cache if include is set
			cacheDs, err := loadedModel.App.FindDatasource(loadedModel.Config.Cache.Datasource)
			if err != nil {
				return err
			}

			safeCacheDs = cacheDs.(*datasource.Datasource)
		}

		documentsToCacheByKey := make(map[string]wst.A)
		for dsCursor.Next(loadedModel.Datasource.Context) {
			inst, err := loadedModel.dispatchFindManySingleDocument(dsCursor, targetInclude, currentContext, filterMap, disabledCache, safeCacheDs, documentsToCacheByKey)
			if err != nil {
				cursor.Error(err)
				return err
			} else if inst != nil {
				results <- inst
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
}

func (loadedModel *StatefulModel) dispatchFindManySingleDocument(dsCursor datasource.MongoCursorI, targetInclude *wst.Include, currentContext *EventContext, filterMap *wst.Filter, disabledCache bool, safeCacheDs *datasource.Datasource, documentsToCacheByKey map[string]wst.A) (*StatefulInstance, error) {
	var document wst.M
	err := dsCursor.Decode(&document)
	if err != nil {
		return nil, err
	}

	if targetInclude != nil {
		for _, includeItem := range *targetInclude {
			relationName := includeItem.Relation
			relation := (*loadedModel.Config.Relations)[relationName]
			relatedModelName := relation.Model
			relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]
			if relatedLoadedModel == nil {
				return nil, fmt.Errorf("could not find related model %v", relatedModelName)
			}

			err := loadedModel.mergeRelated(1, &wst.A{document}, includeItem, currentContext)
			if err != nil {
				return nil, err
			}

		}
	}

	inst, err := loadedModel.Build(document, currentContext)
	if err != nil {
		return nil, err
	}
	var includePrefix = ""
	if targetInclude != nil {
		marshalledTargetInclude, err := json.Marshal(targetInclude)
		if err != nil {
			return nil, err
		}
		includePrefix = fmt.Sprintf("_inc_%s_", marshalledTargetInclude)
	}
	if filterMap != nil && filterMap.Where != nil {
		marshalledWhere, err := json.Marshal(filterMap.Where)
		if err != nil {
			return nil, err
		}
		includePrefix += fmt.Sprintf("_whr_%s_", marshalledWhere)
	}
	if safeCacheDs != nil && !disabledCache {

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
					return nil, err3
				}
				err2 := doExpireCacheKey(safeCacheDs, loadedModel, canonicalId)
				if err2 != nil {
					return nil, err2
				}
			} else {
				documentsToCacheByKey[canonicalId] = append(documentsToCacheByKey[canonicalId], toCache)
			}
		}

	}
	return inst.(*StatefulInstance), err
}

func GetIDAsString(idToConvert interface{}) string {
	var foundObjAccountId string
	if v, ok := idToConvert.(primitive.ObjectID); ok {
		foundObjAccountId = v.Hex()
	} else if v, ok := idToConvert.(*primitive.ObjectID); ok {
		foundObjAccountId = v.Hex()
	} else if v, ok := idToConvert.(string); ok {
		foundObjAccountId = v
	} else {
		foundObjAccountId = fmt.Sprintf("%v", idToConvert)
	}
	return foundObjAccountId
}

func CreateBearer(subjectId interface{}, createdAtSeconds float64, ttlSeconds float64, roles []string) *BearerToken {
	return &BearerToken{
		Account: &BearerAccount{Id: subjectId},
		Claims: jwt.MapClaims{
			"created":   createdAtSeconds,
			"ttl":       ttlSeconds,
			"roles":     roles,
			"accountId": GetIDAsString(subjectId),
		},
	}
}
