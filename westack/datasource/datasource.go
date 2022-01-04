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

func (ds *Datasource) FindMany(collectionName string, filter *map[string]interface{}) *mongo.Cursor {
	if err := validateFilter(filter); err != nil {
		panic(err)
	}
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db mongo.Client = ds.Db.(mongo.Client)

		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		var targetFilter map[string]interface{}
		if filter != nil && (*filter)["where"] != nil {
			targetFilter = (*filter)["where"].(map[string]interface{})
		} else {
			targetFilter = map[string]interface{}{}
		}
		cursor, err := collection.Find(context.Background(), targetFilter)
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
	for key, _ := range *filter {
		if key == "where" || key == "include" || key == "skip" || key == "limit" || key == "order" {

		} else {
			return NewError(400, fmt.Sprintf("Invalid key %v in filter", key))
		}
	}
	return nil
}

func (ds *Datasource) FindById(collectionName string, id interface{}, filter *map[string]interface{}) *mongo.Cursor {
	var _id interface{}
	switch id.(type) {
	case string:
		var err error
		_id, err = primitive.ObjectIDFromHex(id.(string))
		if err != nil {
			log.Println("WARNING: _id", _id, " is not a valid ObjectID:", err.Error())
			//return nil
		}
	default:
		_id = id
	}
	return findByObjectId(collectionName, _id, ds)
}

func findByObjectId(collectionName string, _id interface{}, ds *Datasource) *mongo.Cursor {
	filter := &map[string]interface{}{"where": map[string]interface{}{"_id": _id}}
	cursor := ds.FindMany(collectionName, filter)
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
		return findByObjectId(collectionName, cursor.InsertedID, ds)
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
		return findByObjectId(collectionName, id, ds)
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
