package datasource

import (
	"context"
	"errors"
	"fmt"
	wst "github.com/fredyk/westack-go/westack/common"
	//"go.mongodb.org/mongo-driver/bson"
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
	Config wst.M
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
		ds.Db = db

		go func() {
			for {
				time.Sleep(time.Second * 5)
				err := ds.Db.(*mongo.Client).Ping(mongoCtx, nil)
				if err != nil {
					log.Printf("Reconnecting %v\n", ds.Config["url"])
					db, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(ds.Config["url"].(string)))
					if err != nil {
						log.Printf("Could not reconnect %v: %v\n", ds.Config["url"], err)
						continue
					}
					ds.Db = db
				} else {
					log.Println("Ping OK")
				}
			}
		}()
	default:
		return errors.New("Invalid connector " + connector)
	}
	return nil
}

func (ds *Datasource) FindMany(collectionName string, filter *wst.Filter, lookups *wst.A) *mongo.Cursor {
	if err := validateFilter(filter); err != nil {
		panic(err)
	}
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db *mongo.Client = ds.Db.(*mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		//var targetWhere wst.M
		//if filter != nil && (*filter)["where"] != nil {
		//	targetWhere = (*filter)["where"].(wst.M)
		//} else {
		//	targetWhere = wst.M{}
		//}
		//ReplaceObjectIds(targetWhere)
		pipeline := wst.A{
			//{"$match": targetWhere},
		}

		if lookups != nil {
			pipeline = append(pipeline, *lookups...)
		}
		allowDiskUse := true
		cursor, err := collection.Aggregate(context.Background(), pipeline, &options.AggregateOptions{
			AllowDiskUse: &allowDiskUse,
		})
		if err != nil {
			panic(err)
		}
		return cursor
	}
	return nil
}

func validateFilter(filter *wst.Filter) error {
	if filter == nil {
		return nil
	}
	//for key := range *filter {
	//	if key == "where" || key == "include" || key == "skip" || key == "limit" || key == "order" {
	//
	//	} else {
	//		return NewError(400, fmt.Sprintf("Invalid key %v in filter", key))
	//	}
	//}
	return nil
}

//goland:noinspection GoUnusedParameter
func (ds *Datasource) FindById(collectionName string, id interface{}, filter *wst.Filter, lookups *wst.A) *mongo.Cursor {
	var _id interface{}
	switch id.(type) {
	case string:
		var err error
		_id, err = primitive.ObjectIDFromHex(id.(string))
		if err != nil {
			//log.Println("WARNING: _id", _id, " is not a valid ObjectID:", err.Error())
			//return nil
			_id = id
		}
	default:
		_id = id
	}
	return findByObjectId(collectionName, _id, ds, lookups)
}

func findByObjectId(collectionName string, _id interface{}, ds *Datasource, lookups *wst.A) *mongo.Cursor {
	filter := &wst.Filter{Where: &wst.Where{"_id": _id}}
	wrappedLookups := &wst.A{
		{
			"$match": wst.M{
				"_id": _id,
			},
		},
	}
	if lookups != nil {
		*wrappedLookups = append(*wrappedLookups, *lookups...)
	}
	cursor := ds.FindMany(collectionName, filter, wrappedLookups)
	if cursor.Next(context.Background()) {
		return cursor
	} else {
		return nil
	}
}

func (ds *Datasource) Create(collectionName string, data *wst.M) (*mongo.Cursor, error) {
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db *mongo.Client = ds.Db.(*mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		cursor, err := collection.InsertOne(context.Background(), data)
		if err != nil {
			return nil, err
		}
		return findByObjectId(collectionName, cursor.InsertedID, ds, nil), nil
	}
	return nil, nil
}

func (ds *Datasource) UpdateById(collectionName string, id interface{}, data *wst.M) *mongo.Cursor {
	var connector = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		delete(*data, "id")
		delete(*data, "_id")
		if _, err := collection.UpdateOne(context.Background(), wst.M{"_id": id}, wst.M{"$set": *data}); err != nil {
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
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		if result, err := collection.DeleteOne(context.Background(), wst.M{"_id": id}); err != nil {
			panic(err)
		} else {
			return result.DeletedCount
		}
	}
	return 0
}

func New(config wst.M) *Datasource {
	ds := &Datasource{
		Config: config,
	}
	return ds
}

func ReplaceObjectIds(data interface{}) interface{} {

	if data == nil {
		return nil
	}

	var finalData wst.M
	switch data.(type) {
	case int, int32, int64, float32, float64, bool, primitive.ObjectID, *primitive.ObjectID, time.Time, primitive.DateTime:
		return data
	case string:
		var newValue interface{}
		var err error
		if regexp.MustCompile("^([0-9a-f]{24})$").MatchString(data.(string)) {
			newValue, err = primitive.ObjectIDFromHex(data.(string))
			//} else if regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d+)?)([+:\\-/0-9a-zA-Z]+)?$").MatchString(data.(string)) {
		} else if regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d+)?)([Z]+)?$").MatchString(data.(string)) {
			//	TODO: parse all type of dates
			//layout := "2006-01-02T15:04:05.000-03:00"
			layout := "2006-01-02T15:04:05.000Z"
			newValue, err = time.Parse(layout, data.(string))
			//if err == nil {
			//	newValue = newValue.(time.Time).Unix()
			//	newValue = primitive.Timestamp{T: uint32(newValue.(int64))}
			//}
		}
		if err != nil {
			log.Println("WARNING: ", err)
		}
		if newValue != nil {
			return newValue
		} else {
			return data
		}
		break
	case wst.Where:
		finalData = wst.M{}
		for key, value := range data.(wst.Where) {
			finalData[key] = value
		}
		break
	case *wst.Where:
		finalData = wst.M{}
		for key, value := range *data.(*wst.Where) {
			finalData[key] = value
		}
		break
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
	default:
		log.Println(fmt.Sprintf("WARNING: Invalid input for ReplaceObjectIds() <- %s", data))
		return data
	}
	for key, value := range finalData {
		if value == nil {
			continue
		}
		var err error
		var newValue interface{}
		switch value.(type) {
		case string, wst.Where, *wst.Where, wst.M, *wst.M, int, int32, int64, float32, float64, bool, primitive.ObjectID, *primitive.ObjectID, time.Time, primitive.DateTime:
			newValue = ReplaceObjectIds(value)
			break
		default:
			asMap, asMapOk := value.(wst.M)
			if asMapOk {
				newValue = ReplaceObjectIds(asMap)
			} else {
				asList, asListOk := value.([]interface{})
				if asListOk {
					for i, asListItem := range asList {
						asList[i] = ReplaceObjectIds(asListItem)
					}
				} else {
					_, asStringListOk := value.([]string)
					if !asStringListOk {
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
								_, asStringListOk := value.([]string)
								if !asStringListOk {
									log.Println(fmt.Sprintf("WARNING: What to do with %v (%s)?", value, value))
								}
							}
						}
					}
				}
			}
		}
		if err == nil && newValue != nil {
			switch data.(type) {
			case wst.Where:
				data.(wst.Where)[key] = newValue
				break
			case *wst.Where:
				(*data.(*wst.Where))[key] = newValue
				break
			case wst.M:
				data.(wst.M)[key] = newValue
				break
			case *wst.M:
				(*data.(*wst.M))[key] = newValue
				break
			case map[string]interface{}:
				data.(map[string]interface{})[key] = newValue
				break
			case *map[string]interface{}:
				(*data.(*map[string]interface{}))[key] = newValue
				break
			default:
				log.Fatal(fmt.Sprintf("Invalid input ReplaceObjectIds() <- %s", data))
			}
			//log.Println(fmt.Sprintf("DEBUG: Converted %v to %v", value, newValue))
		} else if err != nil {
			log.Println("WARNING: ", err)
		}
	}
	return data
}
