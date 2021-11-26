package datasource

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

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

func (ds *Datasource) Create(collectionName string, data *map[string]interface{}) *mongo.Cursor {
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

func New(config map[string]interface{}) *Datasource {
	ds := &Datasource{
		Config: config,
	}
	return ds
}
