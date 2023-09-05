package datasource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/spf13/viper"
)

type Options struct {
	RetryOnError bool
	MongoDB      *MongoDBDatasourceOptions
}

type Datasource struct {
	Name    string
	Db      interface{}
	Viper   *viper.Viper
	Key     string
	Context context.Context
	Options *Options

	ctxCancelFn       context.CancelFunc
	SubViper          *viper.Viper
	connectorInstance PersistedConnector
	app               *wst.IApp
}

func getConnectorByName(name string, dsKey string, dsViper *viper.Viper, options *Options) (PersistedConnector, error) {
	switch name {
	case "mongodb":
		var mongoOptions *MongoDBDatasourceOptions
		if options != nil {
			mongoOptions = options.MongoDB
		}
		return NewMongoDBConnector(mongoOptions), nil
	case "memorykv":
		return NewMemoryKVConnector(wst.CreateDefaultMongoRegistry(), dsKey), nil
	default:
		return nil, errors.New("invalid connector " + name)
	}
}

func (ds *Datasource) Initialize() error {
	dsViper := ds.SubViper
	var connectorName = dsViper.GetString("connector")
	var connector PersistedConnector
	var err error
	connector, err = getConnectorByName(connectorName, ds.Key, dsViper, ds.Options)
	if err != nil {
		return err
	}
	connector.SetConfig(dsViper)
	fmt.Printf("Connecting to datasource %v...\n", ds.Key)
	err = connector.Connect(ds.Context)
	if err != nil {
		fmt.Printf("Could not connect to datasource %v: %v\n", ds.Key, err)
		return err
	} else {
		fmt.Printf("DEBUG: Connected to datasource %v\n", ds.Key)
	}

	fmt.Printf("Pinging datasource %v...\n", ds.Key)
	err = connector.Ping(ds.Context)
	if err != nil {
		fmt.Printf("Could not ping datasource %v: %v\n", ds.Key, err)
		return err
	} else {
		fmt.Printf("DEBUG: Ping result OK for datasource %v\n", ds.Key)
		ds.Db = connector.GetClient()
	}

	// Start a goroutine to reconnect to the datasource if it gets disconnected
	init := time.Now().UnixMilli()
	go func() {
		initialCtx := ds.Context
		for {
			time.Sleep(time.Second * 5)

			err := connector.Ping(initialCtx)
			if err != nil {
				log.Printf("Reconnecting datasource %v...\n", ds.Key)
				err := connector.Connect(initialCtx)
				if err != nil {
					if ds.Options == nil || !ds.Options.RetryOnError {
						ds.app.Logger().Fatalf("Could not reconnect %v: %v\n", ds.Key, err)
					}
				} else {
					err = connector.Ping(initialCtx)
					if err != nil {
						if ds.Options == nil || !ds.Options.RetryOnError {
							ds.app.Logger().Fatalf("Mongo client disconnected after %vms: %v", time.Now().UnixMilli()-init, err)
						}
					} else {
						log.Printf("successfully reconnected to %v\n", ds.Key)
						ds.Db = connector.GetClient()
					}
				}
			}
		}
	}()

	ds.connectorInstance = connector
	return nil
}

// FindMany retrieves data from the specified collection based on the provided lookup conditions using the appropriate
// data source connector specified in the configuration file.
// @param collectionName string: the name of the collection from which to retrieve data.
// @param lookups *wst.A: a pointer to an array of conditions to be used as lookup criteria. If nil, all data in the
// collection will be returned.
// @return MongoCursorI: a cursor to the result set that matches the lookup criteria, or an error if an error occurs
// while attempting to retrieve the data.
// The cursor needs to be closed outside of the function.
// Implementations for Redis and memorykv connectors are not yet implemented and will result in an error.
func (ds *Datasource) FindMany(collectionName string, lookups *wst.A) (MongoCursorI, error) {
	return ds.connectorInstance.FindMany(collectionName, lookups)
}

func (ds *Datasource) Count(collectionName string, lookups *wst.A) (int64, error) {
	return ds.connectorInstance.Count(collectionName, lookups)
}

func (ds *Datasource) Create(collectionName string, data *wst.M) (*wst.M, error) {
	return ds.connectorInstance.Create(collectionName, data)
}

func (ds *Datasource) UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error) {
	return ds.connectorInstance.UpdateById(collectionName, id, data)
}

func (ds *Datasource) DeleteById(collectionName string, id interface{}) (DeleteResult, error) {
	return ds.connectorInstance.DeleteById(collectionName, id)
}

// whereLookups is in the form of
// [
//
//	{
//	  "$match": {
//	    "name": "John"
//	  }
//	}
//
// ]
// and is used to filter the documents to delete.
// It cannot be nil or empty.
func (ds *Datasource) DeleteMany(collectionName string, whereLookups *wst.A) (result DeleteResult, err error) {
	if whereLookups == nil {
		return result, errors.New("whereLookups cannot be nil")
	}
	if len(*whereLookups) != 1 {
		return result, errors.New("whereLookups must have exactly one element as a $match stage")
	}
	if (*whereLookups)[0] == nil {
		return result, errors.New("whereLookups cannot have nil elements")
	}
	if (*whereLookups)[0]["$match"] == nil {
		return result, errors.New("first element of whereLookups must be a $match stage")
	}
	if len((*whereLookups)[0]) != 1 {
		return result, errors.New("first element of whereLookups must be a single $match stage")
	}
	if len((*whereLookups)[0]["$match"].(wst.M)) == 0 {
		return result, errors.New("first element of whereLookups must be a single and non-empty $match stage")
	}

	return ds.connectorInstance.DeleteMany(collectionName, whereLookups)

}

func (ds *Datasource) Close() error {
	err := ds.connectorInstance.Disconnect()
	if err != nil {
		fmt.Printf("ERROR: Could not close datasource %v: %v\n", ds.Key, err)
		return err
	}
	ds.ctxCancelFn()
	fmt.Printf("INFO: Closed datasource %v\n", ds.Key)
	return nil
}

func (ds *Datasource) SetTimeout(seconds float32) {
	ds.connectorInstance.SetTimeout(seconds)
}

func New(app *wst.IApp, dsKey string, dsViper *viper.Viper, parentContext context.Context) *Datasource {
	subViper := dsViper.Sub(dsKey)
	if subViper == nil {
		subViper = viper.New()
	}
	subViper.SetEnvPrefix("wst_" + dsKey)
	replacer := strings.NewReplacer(".", "_")
	subViper.SetEnvKeyReplacer(replacer)
	subViper.AutomaticEnv()
	name := subViper.GetString("name")
	if name == "" {
		name = dsKey
	}
	ctx, ctxCancelFn := context.WithCancel(parentContext)
	ds := &Datasource{
		Name:     name,
		Viper:    dsViper,
		SubViper: subViper,

		Key: dsKey,

		Context:     ctx,
		ctxCancelFn: ctxCancelFn,
		app:         app,
	}
	return ds
}
