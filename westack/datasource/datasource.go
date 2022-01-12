package datasource

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"regexp"
	"time"
)

type OperationError struct {
	Code    int
	Message string
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%v %v", e.Code, e.Message)
}

func NewError(code int, message string) *OperationError {
	res := &OperationError{
		code, message,
	}
	return res
}

type Datasource struct {
	Config map[string]interface{}
	Db     interface{}
}

func (ds *Datasource) Initialize() error {
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		mongoCtx := context.Background()
		db, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(ds.Config["url"].(string)))
		if err != nil {
			return err
		}
		ds.Db = *db
	default:
		return errors.New("Invalid connector " + connector)
	}
	return nil
}

func (ds *Datasource) FindMany(collectionName string, filter *map[string]interface{}, lookups *[]map[string]interface{}) *mongo.Cursor {
	if err := validateFilter(filter); err != nil {
		panic(err)
	}
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db mongo.Client = ds.Db.(mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		var targetWhere map[string]interface{}
		if filter != nil && (*filter)["where"] != nil {
			targetWhere = (*filter)["where"].(map[string]interface{})
		} else {
			targetWhere = map[string]interface{}{}
		}
		ReplaceObjectIds(targetWhere)
		pipeline := []map[string]interface{}{
			{"$match": targetWhere},
		}

		if lookups != nil {
			pipeline = append(pipeline, *lookups...)
		}
		cursor, err := collection.Aggregate(context.Background(), pipeline)
		if err != nil {
			panic(err)
		}
		return cursor
	}
	return nil
}

func validateFilter(filter *map[string]interface{}) error {
	if filter == nil {
		return nil
	}
	for key := range *filter {
		if key == "where" || key == "include" || key == "skip" || key == "limit" || key == "order" {

		} else {
			return NewError(400, fmt.Sprintf("Invalid key %v in filter", key))
		}
	}
	return nil
}

//goland:noinspection GoUnusedParameter
func (ds *Datasource) FindById(collectionName string, id interface{}, filter *map[string]interface{}, lookups *[]map[string]interface{}) *mongo.Cursor {
	var _id interface{}
	switch id.(type) {
	case string:
		var err error
		_id, err = primitive.ObjectIDFromHex(id.(string))
		if err != nil {
			log.Println("WARNING: _id", _id, " is not a valid ObjectID:", err.Error())
			//return nil
			_id = id
		}
	default:
		_id = id
	}
	return findByObjectId(collectionName, _id, ds, lookups)
}

func findByObjectId(collectionName string, _id interface{}, ds *Datasource, lookups *[]map[string]interface{}) *mongo.Cursor {
	filter := &map[string]interface{}{"where": map[string]interface{}{"_id": _id}}
	cursor := ds.FindMany(collectionName, filter, lookups)
	if cursor.Next(context.Background()) {
		return cursor
	} else {
		return nil
	}
}

func (ds *Datasource) Create(collectionName string, data *bson.M) *mongo.Cursor {
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db mongo.Client = ds.Db.(mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		cursor, err := collection.InsertOne(context.Background(), data)
		if err != nil {
			panic(err)
		}
		return findByObjectId(collectionName, cursor.InsertedID, ds, nil)
	}
	return nil
}

func (ds *Datasource) UpdateById(collectionName string, id interface{}, data *bson.M) *mongo.Cursor {
	var connector = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db = ds.Db.(mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		if _, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": data}); err != nil {
			panic(err)
		}
		return findByObjectId(collectionName, id, ds, nil)
	}
	return nil
}

func (ds *Datasource) DeleteById(collectionName string, id interface{}) int64 {
	var connector = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db = ds.Db.(mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		if result, err := collection.DeleteOne(context.Background(), bson.M{"_id": id}); err != nil {
			panic(err)
		} else {
			return result.DeletedCount
		}
	}
	return 0
}

func New(config map[string]interface{}) *Datasource {
	ds := &Datasource{
		Config: config,
	}
	return ds
}

func ReplaceObjectIds(data interface{}) interface{} {

	var finalData bson.M
	switch data.(type) {
	case string:
	case int32:
	case int64:
	case float32:
	case float64:
	case primitive.ObjectID:
	case *primitive.ObjectID:
	case time.Time:
		return data
	case map[string]interface{}:
		finalData = bson.M{}
		for key, value := range data.(map[string]interface{}) {
			finalData[key] = value
		}
		break
	case *map[string]interface{}:
		finalData = bson.M{}
		for key, value := range *data.(*map[string]interface{}) {
			finalData[key] = value
		}
		break
	case bson.M:
		finalData = data.(bson.M)
		break
	case *bson.M:
		finalData = *data.(*bson.M)
		break
	default:
		log.Println(fmt.Sprintf("WARNING: Invalid input for ReplaceObjectIds() <- %s", data))
		return data
	}
	for key, value := range finalData {
		var err error
		var newValue interface{}
		switch value.(type) {
		case string:
			if regexp.MustCompile("^([0-9a-f]{24})$").MatchString(value.(string)) {
				newValue, err = primitive.ObjectIDFromHex(value.(string))
				//} else if regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d+)?)([+:\\-/0-9a-zA-Z]+)?$").MatchString(value.(string)) {
			} else if regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d+)?)([Z]+)?$").MatchString(value.(string)) {
				//	TODO: parse all type of dates
				//layout := "2006-01-02T15:04:05.000-03:00"
				layout := "2006-01-02T15:04:05.000Z"
				newValue, err = time.Parse(layout, value.(string))
				//if err == nil {
				//	newValue = newValue.(time.Time).Unix()
				//	newValue = primitive.Timestamp{T: uint32(newValue.(int64))}
				//}
			}
		case bson.M:
		case *bson.M:
			newValue = ReplaceObjectIds(value)
			break
		case int32:
		case int64:
		case float32:
		case float64:
		case primitive.ObjectID:
		case *primitive.ObjectID:
		case time.Time:
			break
		default:
			asMap, asMapOk := value.(map[string]interface{})
			if asMapOk {
				newValue = ReplaceObjectIds(asMap)
			} else {
				asList, asListOk := value.([]interface{})
				if asListOk {
					for i, asListItem := range asList {
						asList[i] = ReplaceObjectIds(asListItem)
					}
				} else {
					log.Println(fmt.Sprintf("WARNING: What to do with %v (%s)?", value, value))
				}
			}
		}
		if err == nil && newValue != nil {
			switch data.(type) {
			case map[string]interface{}:
				data.(map[string]interface{})[key] = newValue
				break
			case *map[string]interface{}:
				(*data.(*map[string]interface{}))[key] = newValue
				break
			case bson.M:
				data.(bson.M)[key] = newValue
				break
			case *bson.M:
				(*data.(*bson.M))[key] = newValue
				break
			default:
				log.Fatal(fmt.Sprintf("Invalid input for Model.Create() <- %s", data))
			}
			log.Println(fmt.Sprintf("DEBUG: Converted %v to %v", value, newValue))
		} else if err != nil {
			log.Println("WARNING: ", err)
		}
	}
	return data
}
