package model

import (
	"fmt"
	"log"
	"reflect"

	"github.com/oliveagle/jsonpath"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
)

type Instance interface {
	GetID() interface{}
	UpdateAttributes(data interface{}, baseContext *EventContext) (Instance, error)
	ToJSON() wst.M
	Get(relationName string) interface{}
	GetA(path string) *wst.A
	GetM(path string) *wst.M
	GetString(path string) string
	GetInt(path string) int64
	GetFloat64(path string) float64
	GetBoolean(path string, defaultValue bool) bool
	GetObjectId(path string) primitive.ObjectID
	GetOne(relation string) Instance
	GetMany(relation string) InstanceA
	GetModel() Model
}

type StatefulInstance struct {
	Model *StatefulModel
	Id    interface{}

	data  wst.M
	bytes []byte
}

type InstanceA []Instance

func (modelInstance *StatefulInstance) GetID() interface{} {
	if modelInstance == nil {
		return nil
	}
	return modelInstance.Id
}

func (modelInstance *StatefulInstance) GetModel() Model {
	if modelInstance == nil {
		return nil
	}
	return modelInstance.Model
}

func (modelInstance *StatefulInstance) ToJSON() wst.M {

	if modelInstance == nil {
		return nil
	}

	if modelInstance == modelInstance.Model.NilInstance {
		return wst.NilMap
	}

	result := wst.CopyMap(modelInstance.data)
	for relationName, relationConfig := range *modelInstance.Model.Config.Relations {
		if modelInstance.data[relationName] != nil {
			rawRelatedData := modelInstance.data[relationName]
			relatedModel, _ := modelInstance.Model.App.FindModel(relationConfig.Model)
			if relatedModel != nil {
				switch {
				case isSingleRelation(relationConfig.Type):
					relatedInstance := rawRelatedData.(*StatefulInstance).ToJSON()
					result[relationName] = relatedInstance
				case isManyRelation(relationConfig.Type):
					aux := make(wst.A, len(rawRelatedData.(InstanceA)))
					for idx, v := range rawRelatedData.(InstanceA) {
						aux[idx] = v.ToJSON()
					}
					result[relationName] = aux
				}
			}
		}
	}

	return result
}

func (modelInstance *StatefulInstance) Get(relationName string) interface{} {
	result := modelInstance.data[relationName]
	switch (*modelInstance.Model.Config.Relations)[relationName].Type {
	case "hasMany", "hasAndBelongsToMany":
		if result == nil {
			result = make(InstanceA, 0)
		}
	}
	return result
}

func (modelInstance *StatefulInstance) GetOne(relationName string) Instance {
	result := modelInstance.Get(relationName)
	if result == nil {
		return nil
	}
	return result.(*StatefulInstance)
}

func (modelInstance *StatefulInstance) GetMany(relationName string) InstanceA {
	return modelInstance.Get(relationName).(InstanceA)
}

func (modelInstance *StatefulInstance) HideProperties() {
	for _, propertyName := range modelInstance.Model.Config.Hidden {
		delete(modelInstance.data, propertyName)
	}
	// Hide in nested
	for relationKey, relationConfig := range *modelInstance.Model.Config.Relations {
		if relationConfig.Type == "hasMany" || relationConfig.Type == "hasAndBelongsToMany" {
			for _, instance := range modelInstance.GetMany(relationKey) {
				instance.(*StatefulInstance).HideProperties()
			}
		} else if relationConfig.Type == "hasOne" || relationConfig.Type == "belongsTo" {
			if instance := modelInstance.GetOne(relationKey); instance != nil {
				instance.(*StatefulInstance).HideProperties()
			}
		}
	}
}

func (modelInstance *StatefulInstance) Transform(out interface{}) (err error) {
	err = modelInstance.requireBytes()
	if err == nil {
		err = bson.UnmarshalWithRegistry(modelInstance.Model.App.Bson.Registry, modelInstance.bytes, out)
		if err != nil && modelInstance.Model.App.Debug {
			fmt.Printf("Error while unmarshalling instance: %s", err)
		}
	}
	return
}

func (modelInstance *StatefulInstance) UncheckedTransform(out interface{}) interface{} {
	err := modelInstance.Transform(out)
	if err != nil {
		panic(err)
	}
	return out
}

func (modelInstance *StatefulInstance) UpdateAttributes(data interface{}, baseContext *EventContext) (Instance, error) {

	var finalData wst.M
	switch data.(type) {
	case map[string]interface{}:
		finalData = wst.M{}
		for key, value := range data.(map[string]interface{}) {
			finalData[key] = value
		}
	case *map[string]interface{}:
		finalData = wst.M{}
		for key, value := range *data.(*map[string]interface{}) {
			finalData[key] = value
		}
	case wst.M:
		finalData = data.(wst.M)
	case *wst.M:
		finalData = *data.(*wst.M)
	case StatefulInstance:
		value := data.(StatefulInstance)
		finalData = (&value).ToJSON()
	case *StatefulInstance:
		finalData = data.(*StatefulInstance).ToJSON()
	default:
		// check if data is a struct
		if reflect.TypeOf(data).Kind() == reflect.Struct {
			bytes, err := bson.MarshalWithRegistry(modelInstance.Model.App.Bson.Registry, data)
			if err != nil {
				return nil, err
			}
			err = bson.UnmarshalWithRegistry(modelInstance.Model.App.Bson.Registry, bytes, &finalData)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid input for Model.UpdateAttributes() <- %s", data)
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
		_, err := datasource.ReplaceObjectIds(finalData)
		if err != nil {
			return nil, err
		}
	}

	eventContext := &EventContext{
		BaseContext: targetBaseContext,
	}
	eventContext.Data = &finalData
	eventContext.Instance = modelInstance
	eventContext.Model = modelInstance.Model
	eventContext.ModelID = modelInstance.Id
	eventContext.IsNewInstance = false
	eventContext.OperationName = wst.OperationNameUpdateAttributes
	if modelInstance.Model.DisabledHandlers["__operation__before_save"] != true {
		err := modelInstance.Model.GetHandler("__operation__before_save")(eventContext)
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
				v, err := modelInstance.Model.Build(eventContext.Result.(wst.M), targetBaseContext)
				if err != nil {
					return nil, err
				}
				return v, nil
			default:
				return nil, fmt.Errorf("invalid eventContext.Result type, expected Instance, Instance or wst.M; found %T", eventContext.Result)
			}
		}
	}

	for key := range *modelInstance.Model.Config.Relations {
		delete(finalData, key)
	}
	_, err := modelInstance.Model.Datasource.UpdateById(modelInstance.Model.CollectionName, modelInstance.Id, &finalData)

	if err != nil {
		return nil, err
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

func (modelInstance *StatefulInstance) Reload(eventContext *EventContext) error {
	newInstance, err := modelInstance.Model.FindById(modelInstance.Id, nil, eventContext)
	if err != nil {
		return err
	}
	for k := range modelInstance.data {
		if (*modelInstance.Model.Config.Relations)[k] == nil {
			delete(modelInstance.data, k)
		}
	}
	for k, v := range newInstance.(*StatefulInstance).data {
		if (*modelInstance.Model.Config.Relations)[k] == nil {
			modelInstance.data[k] = v
		}
	}
	modelInstance.data = newInstance.(*StatefulInstance).data
	modelInstance.bytes = nil
	return nil
}

func (modelInstance *StatefulInstance) GetString(path string) string {
	if res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path)); err == nil {
		switch res.(type) {
		case string:
			return res.(string)
		case primitive.ObjectID:
			return res.(primitive.ObjectID).Hex()
		}
	}
	return ""
}

func (modelInstance *StatefulInstance) GetFloat64(path string) float64 {
	if res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path)); err == nil {
		if v, ok := res.(float64); ok {
			return v
		} else if v, ok := res.(float32); ok {
			return float64(v)
		} else if v, ok := res.(int64); ok {
			return float64(v)
		} else if v, ok := res.(int32); ok {
			return float64(v)
		} else if v, ok := res.(int); ok {
			return float64(v)
		}
	}
	return 0
}

func (modelInstance *StatefulInstance) GetInt(path string) int64 {
	if res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path)); err == nil {
		if v, ok := res.(int64); ok {
			return v
		} else if v, ok := res.(int32); ok {
			return int64(v)
		} else if v, ok := res.(int); ok {
			return int64(v)
		} else if v, ok := res.(float64); ok {
			return int64(v)
		} else if v, ok := res.(float32); ok {
			return int64(v)
		}
	}
	return 0
}

func (modelInstance *StatefulInstance) GetBoolean(path string, defaultValue bool) bool {
	if res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path)); err == nil {
		switch res.(type) {
		case bool:
			return res.(bool)
		}
	}
	return defaultValue
}

func (modelInstance *StatefulInstance) GetObjectId(path string) (result primitive.ObjectID) {
	result = primitive.NilObjectID
	if res, err := jsonpath.JsonPathLookup(modelInstance.data, fmt.Sprintf("$.%v", path)); err == nil {
		switch res.(type) {
		case string:
			_id, err := primitive.ObjectIDFromHex(res.(string))
			if err == nil {
				result = _id
			}
		case primitive.ObjectID:
			result = res.(primitive.ObjectID)
		}
	}
	return result
}

func (modelInstance *StatefulInstance) GetM(path string) *wst.M {
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

func (modelInstance *StatefulInstance) GetA(path string) *wst.A {
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
		log.Printf("[WARNING] GetA: %v <%s> is not an array\n", path, modelInstance.data[path])
		return nil
	}
}

func (modelInstance *StatefulInstance) requireBytes() (err error) {
	if modelInstance.bytes == nil {
		//if len(modelInstance.bytes) == 0 {
		//// register encoder for primitive.ObjectID
		//bson.DefaultRegistry.RegisterEncoder(primitive.ObjectID{}, bson.ObjectIDEncoder{})
		//bson.NewRegistryBuilder().Build()
		//bson.MarshalWithRegistry(bson.NewRegistryBuilder().RegisterTypeEncoder(reflect.TypeOf(primitive.ObjectID{}), bson.ObjectIDEncoder{}).Build(), modelInstance.data)
		//// register decoder for primitive.ObjectID
		//bson.DefaultRegistry.RegisterDecoder(primitive.ObjectID{}, bson.ObjectIDDecoder{})

		if modelInstance.Model.App.Debug {
			log.Printf("[DEBUG] marshalling at requireBytes(): %v\n", modelInstance.data)
		}
		//modelInstance.bytes, Err = bson.MarshalWithRegistry(modelInstance.Model.App.Bson.Registry, modelInstance.data)
		modelInstance.bytes, err = modelInstance.MarshalBSON()
		//modelInstance.bytes, Err = easyjson.Marshal(modelInstance.data)
	}
	if err != nil && modelInstance.Model.App.Debug {
		log.Printf("[ERROR] while marshalling Instance: %v\n", err)
	}
	return err
}

// Inherit easyjson

func (modelInstance *StatefulInstance) MarshalBSON() (out []byte, err error) {
	// marshal modelInstance.data
	toMarshal := modelInstance.data
	if modelInstance.Model.App.Debug {
		log.Printf("[DEBUG] marshalling Instance: %v\n", toMarshal)
	}
	insertedId := false
	if v, ok0 := toMarshal["id"]; ok0 {
		if _, ok1 := toMarshal["_id"]; !ok1 {
			toMarshal["_id"] = v
			insertedId = true
		}
	}
	//bytes, Err := easyjson.Marshal(toMarshal)
	//w.Raw(bytes, Err)
	//if modelInstance.Model.App.Debug {
	//	log.Printf("[DEBUG] marshalled Instance: %v\n", len(bytes))
	//}
	bytes, err := bson.MarshalWithRegistry(modelInstance.Model.App.Bson.Registry, toMarshal)
	if insertedId {
		delete(toMarshal, "_id")
	}
	return bytes, err
}

func (instances InstanceA) ToJSON() []wst.M {
	result := make([]wst.M, len(instances))
	for idx, instance := range instances {
		result[idx] = instance.ToJSON()
	}
	return result
}

//func (instances InstanceA) MarshalBSON() (out []byte, Err error) {
//	// marshal bson as array of modelInstance.data
//	//toMarshal := make([]wst.M, len(instances))
//	//for idx, instance := range instances {
//	//	toMarshal[idx] = instance.data
//	//}
//	if instances[0].Model.App.Debug {
//		log.Printf("[DEBUG] marshalling InstanceA: %v\n", instances)
//	}
//	//
//	for idx, instance := range instances {
//		if idx == 0 {
//			out, Err = bson.MarshalWithRegistry(instance.Model.App.Bson.Registry, instance.data)
//			if Err != nil {
//				return
//			}
//		} else {
//			aux, Err := bson.MarshalWithRegistry(instance.Model.App.Bson.Registry, instance.data)
//			if Err != nil {
//				return nil, Err
//			}
//			out = append(out, aux...)
//		}
//	}
//	return
//}
