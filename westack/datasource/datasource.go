package datasource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fredyk/westack-go/westack/memorykv"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
)

type OperationError struct {
	Code    int
	Message string
}

type Options struct {
	RetryOnError bool
	MongoDB      *MongoDBDatasourceOptions
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
	Name    string
	Db      interface{}
	Viper   *viper.Viper
	Key     string
	Context context.Context
	Options *Options

	ctxCancelFn       context.CancelFunc
	SubViper          *viper.Viper
	connectorInstance PersistedConnector
}

func getConnectorByName(name string, dsKey string, dsViper *viper.Viper, options *Options) (PersistedConnector, error) {
	switch name {
	case "mongodb":
		var mongoOptions *MongoDBDatasourceOptions
		if options != nil {
			mongoOptions = options.MongoDB
		}
		return NewMongoDBConnector(mongoOptions), nil
	case "redis":
		return nil, fmt.Errorf("redis connector not implemented yet")
	case "memorykv":
		return NewMemoryKVConnector(dsKey), nil
	default:
		return nil, errors.New("invalid connector " + name)
	}
}

func (ds *Datasource) Initialize() error {
	dsViper := ds.SubViper
	if dsViper == nil {
		return fmt.Errorf("could not find datasource %v", ds.Key)
	}
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
		fmt.Printf("Could not connect to datasource %v: %v\n", ds.Key, err)
		return err
	} else {
		fmt.Printf("DEBUG: Connected to datasource %v\n", ds.Key)
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
						log.Fatalf("Could not reconnect %v: %v\n", ds.Key, err)
					}
				} else {
					err = connector.Ping(initialCtx)
					if err != nil {
						if ds.Options == nil || !ds.Options.RetryOnError {
							log.Fatalf("Mongo client disconnected after %vms: %v", time.Now().UnixMilli()-init, err)
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
	connector := ds.connectorInstance
	cursor, err := connector.FindMany(collectionName, lookups)
	if err != nil {
		return nil, err
	}
	return cursor, nil
}

func (ds *Datasource) Count(collectionName string, lookups *wst.A) (int64, error) {
	connector := ds.connectorInstance
	count, err := connector.Count(collectionName, lookups)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func findByObjectId(collectionName string, _id interface{}, ds *Datasource, lookups *wst.A) (*wst.M, error) {
	connector := ds.connectorInstance
	data, err := connector.findByObjectId(collectionName, _id, lookups)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ds *Datasource) Create(collectionName string, data *wst.M) (*wst.M, error) {
	var connector = ds.SubViper.GetString("connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.SubViper.GetString("database"))
		collection := database.Collection(collectionName)
		if (*data)["_id"] == nil && (*data)["id"] != nil {
			(*data)["_id"] = (*data)["id"]
		}
		insertOneResult, err := collection.InsertOne(ds.Context, data)
		if err != nil {
			return nil, err
		}
		return findByObjectId(collectionName, insertOneResult.InsertedID, ds, nil)
	case "redis":
		return nil, fmt.Errorf("redis connector not implemented yet")
	case "memorykv":
		dict := ds.Db.(memorykv.MemoryKvDb)

		var id interface{}

		var allBytes [][]byte
		var idAsStr string
		if (*data)["_redId"] == nil {
			id = uuid.New().String()
			(*data)["_redId"] = id
		} else {
			id = (*data)["_redId"]
		}
		for _, doc := range (*data)["_entries"].(wst.A) {
			switch id.(type) {
			case string:
				idAsStr = id.(string)
			case primitive.ObjectID:
				idAsStr = id.(primitive.ObjectID).Hex()
			}

			bytes, err := bson.Marshal(doc)
			if err != nil {
				return nil, err
			}
			allBytes = append(allBytes, bytes)
		}

		//dict[id] = data
		err := dict.GetBucket(collectionName).Set(idAsStr, allBytes)
		if err != nil {
			return nil, err
		}
		//return findByObjectId(collectionName, id, ds, nil)
		return data, nil
	}
	return nil, errors.New(fmt.Sprintf("invalid connector %v", connector))
}

func (ds *Datasource) UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error) {
	var connector = ds.SubViper.GetString("connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.SubViper.GetString("database"))
		collection := database.Collection(collectionName)
		delete(*data, "id")
		delete(*data, "_id")
		if _, err := collection.UpdateOne(ds.Context, wst.M{"_id": id}, wst.M{"$set": *data}); err != nil {
			panic(err)
		}
		return findByObjectId(collectionName, id, ds, nil)
	}
	return nil, errors.New(fmt.Sprintf("invalid connector %v", connector))
}

func (ds *Datasource) DeleteById(collectionName string, id interface{}) int64 {
	var connector = ds.SubViper.GetString("connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.SubViper.GetString("database"))
		collection := database.Collection(collectionName)
		if result, err := collection.DeleteOne(ds.Context, wst.M{"_id": id}); err != nil {
			panic(err)
		} else {
			return result.DeletedCount
		}
	}
	return 0
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
func (ds *Datasource) DeleteMany(collectionName string, whereLookups *wst.A) (result DeleteManyResult, err error) {
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

	connector := ds.connectorInstance
	return connector.DeleteMany(collectionName, whereLookups)

}

func New(dsKey string, dsViper *viper.Viper, parentContext context.Context) *Datasource {
	subViper := dsViper.Sub(dsKey)
	if subViper == nil {
		subViper = viper.New()
	}
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
	}
	return ds
}
