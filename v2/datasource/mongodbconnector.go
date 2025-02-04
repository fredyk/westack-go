package datasource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDBDatasourceOptions struct {
	Registry *bsoncodec.Registry
	Monitor  *event.CommandMonitor
	Timeout  float32
}

type MongoDBConnector struct {
	db      *mongo.Client
	options *MongoDBDatasourceOptions
	dsViper *viper.Viper
	context context.Context
}

// MongoDBConnector implements the PersistedConnector interface

func (connector *MongoDBConnector) GetName() string {
	return "mongodb"
}

func (connector *MongoDBConnector) SetConfig(dsViper *viper.Viper) {
	connector.dsViper = dsViper
}

func (connector *MongoDBConnector) Connect(parentContext context.Context) error {
	var mongoCtx context.Context
	var cancelFn context.CancelFunc

	minPoolSize := connector.dsViper.GetInt64("minPoolSize")
	maxPoolSize := connector.dsViper.GetInt64("maxPoolSize")

	if maxPoolSize == 0 {
		maxPoolSize = 5
	}

	if connector.options != nil && connector.options.Timeout > 0 {
		fmt.Printf("[DEBUG] Setting timeout to %v seconds\n", connector.options.Timeout)
		mongoCtx, cancelFn = context.WithTimeout(parentContext, time.Duration(connector.options.Timeout)*time.Second)
		defer cancelFn()
	} else {
		//mongoCtx, cancelFn = context.WithCancel(parentContext)
		mongoCtx = parentContext
	}

	url := getDbUrl(connector.dsViper)

	var clientOpts *options.ClientOptions
	if connector.dsViper.GetString("username") != "" && connector.dsViper.GetString("password") != "" {
		credential := options.Credential{
			Username: connector.dsViper.GetString("username"),
			Password: connector.dsViper.GetString("password"),
		}
		clientOpts = options.Client().ApplyURI(url).SetAuth(credential)
	} else {
		clientOpts = options.Client().ApplyURI(url)
	}

	timeoutForOptions := time.Second * 30
	if connector.options != nil && connector.options.Timeout > 0 {
		timeoutForOptions = time.Duration(connector.options.Timeout) * time.Second
	}
	clientOpts = clientOpts.SetSocketTimeout(timeoutForOptions).SetConnectTimeout(timeoutForOptions).SetServerSelectionTimeout(timeoutForOptions).SetMinPoolSize(uint64(minPoolSize)).SetMaxPoolSize(uint64(maxPoolSize))

	if connector.options != nil && connector.options.Registry != nil {
		clientOpts = clientOpts.SetRegistry(connector.options.Registry)
	}

	if connector.options != nil && connector.options.Monitor != nil {
		clientOpts = clientOpts.SetMonitor(connector.options.Monitor)
	}

	db, err := mongo.Connect(mongoCtx, clientOpts)
	if err != nil {
		cancelFn()
		return err
	}
	connector.db = db
	connector.context = mongoCtx

	return nil
}

func (connector *MongoDBConnector) FindMany(collectionName string, lookups *wst.A) (MongoCursorI, error) {
	var mongoClient = connector.db

	database := mongoClient.Database(connector.dsViper.GetString("database"))
	collection := database.Collection(collectionName)

	pipeline := wst.A{}

	if lookups != nil {
		pipeline = append(pipeline, *lookups...)
	}
	ctx := connector.context
	cursor, err := collection.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true).SetBatchSize(16))
	if err != nil {
		return nil, err
	}
	return cursor, nil
}

func (connector *MongoDBConnector) findByObjectId(collectionName string, _id interface{}, lookups *wst.A) (*wst.M, error) {
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
	cursor, err := connector.FindMany(collectionName, wrappedLookups)
	if err != nil {
		return nil, err
	}
	defer func(cursor MongoCursorI, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			panic(err)
		}
	}(cursor, connector.context)
	var results []wst.M
	err = cursor.All(connector.context, &results)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		return &(results)[0], nil
	} else {
		return nil, errors.New("document not found")
	}
}

func (connector *MongoDBConnector) Count(collectionName string, lookups *wst.A) (wst.CountResult, error) {
	var db = connector.db

	database := db.Database(connector.dsViper.GetString("database"))
	collection := database.Collection(collectionName)

	pipeline := wst.A{}

	if lookups != nil {
		pipeline = append(pipeline, *lookups...)
	}
	pipeline = append(pipeline, wst.M{
		"$group": wst.M{
			"_id": 1,
			"_n":  wst.M{"$sum": 1},
		},
	})
	allowDiskUse := true
	ctx := connector.context
	cursor, err := collection.Aggregate(ctx, pipeline, &options.AggregateOptions{
		AllowDiskUse: &allowDiskUse,
	})
	if err != nil {
		fmt.Printf("error %v\n", err)
		return wst.CountResult{}, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			panic(err)
		}
	}(cursor, ctx)
	var documents []struct {
		Count int64 `bson:"_n"`
	}
	err = cursor.All(ctx, &documents)
	if err != nil {
		return wst.CountResult{}, err
	}
	if len(documents) == 0 {
		return wst.CountResult{}, nil
	}
	return wst.CountResult{Count: documents[0].Count}, nil
}

func (connector *MongoDBConnector) Create(collectionName string, data *wst.M) (*wst.M, error) {
	var db = connector.db

	database := db.Database(connector.dsViper.GetString("database"))
	collection := database.Collection(collectionName)
	if (*data)["_id"] == nil && (*data)["id"] != nil {
		(*data)["_id"] = (*data)["id"]
	}
	insertOneResult, err := collection.InsertOne(connector.context, data)
	if err != nil {
		return nil, err
	}
	return connector.findByObjectId(collectionName, insertOneResult.InsertedID, nil)
}

func (connector *MongoDBConnector) UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error) {
	var db = connector.db

	database := db.Database(connector.dsViper.GetString("database"))
	collection := database.Collection(collectionName)
	delete(*data, "id")
	delete(*data, "_id")
	m, err := updateOneWithRetries(collection, connector, id, data, 2)
	if err != nil {
		return m, err
	}
	return connector.findByObjectId(collectionName, id, nil)
}

func updateOneWithRetries(collection *mongo.Collection, connector *MongoDBConnector, id interface{}, data *wst.M, remainingRetries int) (*wst.M, error) {
	if _, err := collection.UpdateOne(connector.context, wst.M{"_id": id}, wst.M{"$set": *data}); err != nil {
		// broken pipe
		if remainingRetries > 0 && strings.Contains(err.Error(), "broken pipe") {
			fmt.Printf("[WARNING] Retrying updateOne for %v\n", id)
			return updateOneWithRetries(collection, connector, id, data, remainingRetries-1)
		}
		return nil, err
	}
	return nil, nil
}

func (connector *MongoDBConnector) DeleteById(collectionName string, id interface{}) (result wst.DeleteResult, err error) {
	var db = connector.db

	database := db.Database(connector.dsViper.GetString("database"))
	collection := database.Collection(collectionName)
	mongoResult, err := collection.DeleteOne(connector.context, wst.M{"_id": id})
	if err != nil {
		return result, err
	}
	return wst.DeleteResult{DeletedCount: mongoResult.DeletedCount}, nil
}

func (connector *MongoDBConnector) DeleteMany(collectionName string, whereLookups *wst.A) (result wst.DeleteResult, err error) {
	db := connector.db
	database := db.Database(connector.dsViper.GetString("database"))
	collection := database.Collection(collectionName)

	ctx := connector.context
	var mongoFilter bson.D
	for key, value := range (*whereLookups)[0]["$match"].(wst.M) {
		mongoFilter = append(mongoFilter, bson.E{Key: key, Value: value})
	}
	mongoResult, err := collection.DeleteMany(ctx, mongoFilter)
	if err != nil {
		return result, err
	}
	return wst.DeleteResult{DeletedCount: mongoResult.DeletedCount}, nil
}

func (connector *MongoDBConnector) Disconnect() error {
	return connector.db.Disconnect(connector.context)
}

func (connector *MongoDBConnector) Ping(parentCtx context.Context) error {
	var mongoCtx context.Context
	var cancelFn context.CancelFunc
	if connector.options != nil && connector.options.Timeout > 0 {
		mongoCtx, cancelFn = context.WithTimeout(parentCtx, time.Duration(connector.options.Timeout)*time.Second)
		defer cancelFn()
	} else {
		mongoCtx = parentCtx
	}

	return connector.db.Ping(mongoCtx, readpref.SecondaryPreferred())
}

func (connector *MongoDBConnector) SetTimeout(seconds float32) {
	if connector.options != nil {
		connector.options.Timeout = seconds
	} else {
		connector.options = &MongoDBDatasourceOptions{
			Timeout: seconds,
		}
	}
	connector.dsViper.Set("timeout", seconds)
}

func (connector *MongoDBConnector) GetClient() interface{} {
	return connector.db
}

func getDbUrl(dsViper *viper.Viper) string {
	url := ""
	if dsViper.GetString("url") != "" {
		url = dsViper.GetString("url")
	} else {
		port := 0
		if dsViper.GetInt("port") > 0 {
			port = dsViper.GetInt("port")
		}
		url = fmt.Sprintf("mongodb://%v:%v/%v", dsViper.GetString("host"), port, dsViper.GetString("database"))
		log.Printf("Using composed url %v\n", url)
	}
	return url
}

// NewMongoDBConnector Factory method for MongoDBConnector
func NewMongoDBConnector(mongoOptions *MongoDBDatasourceOptions) PersistedConnector {

	return &MongoDBConnector{
		options: mongoOptions,
	}
}
