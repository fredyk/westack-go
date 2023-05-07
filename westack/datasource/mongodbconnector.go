package datasource

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"time"
)

type MongoDBDatasourceOptions struct {
	Registry *bsoncodec.Registry
	Monitor  *event.CommandMonitor
	Timeout  int
}

type MongoDBConnector struct {
	db      *mongo.Client
	options *MongoDBDatasourceOptions
	dsViper *viper.Viper
}

// MongoDBConnector implements the PersistedConnector interface

func (connector *MongoDBConnector) GetName() string {
	return "mongodb"
}

func (connector *MongoDBConnector) Connect(parentContext context.Context) error {
	var mongoCtx context.Context
	var cancelFn context.CancelFunc
	if connector.options != nil && connector.options.Timeout > 0 {
		fmt.Printf("DEBUG: Setting timeout to %v seconds\n", connector.options.Timeout)
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
	clientOpts = clientOpts.SetSocketTimeout(timeoutForOptions).SetConnectTimeout(timeoutForOptions).SetServerSelectionTimeout(timeoutForOptions).SetMinPoolSize(1).SetMaxPoolSize(5)

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

	return nil
}

func (connector *MongoDBConnector) Disconnect() error {
	//TODO implement me
	panic("implement me")
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
func NewMongoDBConnector(dsViper *viper.Viper, mongoOptions *MongoDBDatasourceOptions) PersistedConnector {

	return &MongoDBConnector{
		options: mongoOptions,
		dsViper: dsViper,
	}
}
