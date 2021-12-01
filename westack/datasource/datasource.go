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
		db, _ := mongo.Connect(mongoCtx, options.Client().ApplyURI(ds.Config["url"].(string)))
		ds.Db = *db
	default:
		return errors.New("Invalid connector " + connector)
	}
	return nil
}

func (ds *Datasource) FindMany(collectionName string, filter *map[string]interface{}) *mongo.Cursor {
	if err := validateFilter(filter); err != nil {
		//log.Println(err)
		panic(err)
	}
	var connector string = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db mongo.Client = ds.Db.(mongo.Client)
		// TODO: dynamic database
		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		var targetFilter map[string]interface{}
		if filter != nil && (*filter)["where"] != nil {
			targetFilter = (*filter)["where"].(map[string]interface{})
		} else {
			targetFilter = map[string]interface{}{}
		}
		cursor, _ := collection.Find(context.Background(), targetFilter)
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

func (ds *Datasource) FindById(collectionName string, id string, filter *map[string]interface{}) *mongo.Cursor {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	return findByObjectId(collectionName, _id, ds)
}

func findByObjectId(collectionName string, _id primitive.ObjectID, ds *Datasource) *mongo.Cursor {
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
		// TODO: dynamic database
		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		cursor, _ := collection.InsertOne(context.Background(), data)
		return findByObjectId(collectionName, cursor.InsertedID.(primitive.ObjectID), ds)
	}
	return nil
}

func (ds *Datasource) UpdateById(collectionName string, id primitive.ObjectID, data *bson.M) *mongo.Cursor {
	var connector = ds.Config["connector"].(string)
	switch connector {
	case "mongodb":
		var db = ds.Db.(mongo.Client)
		// TODO: dynamic database
		database := db.Database(ds.Config["database"].(string))
		collection := database.Collection(collectionName)
		if _, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": data}); err != nil {
			panic(err)
		}
		return findByObjectId(collectionName, id, ds)
	}
	return nil
}

func New(config map[string]interface{}) *Datasource {
	ds := &Datasource{
		Config: config,
	}
	return ds
}
