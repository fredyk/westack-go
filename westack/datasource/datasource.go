package datasource

import (
	"context"
	"errors"
	"fmt"
	"github.com/fredyk/westack-go/westack/memorykv"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	wst "github.com/fredyk/westack-go/westack/common"
)

type OperationError struct {
	Code    int
	Message string
}

type MongoDBDatasourceOptions struct {
	Registry     *bsoncodec.Registry
	Monitor      *event.CommandMonitor
	Timeout      int
	RetryOnError bool
}

type Options struct {
	MongoDB *MongoDBDatasourceOptions
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

	ctxCancelFn context.CancelFunc
}

func (ds *Datasource) Initialize() error {
	dsViper := ds.Viper
	var connector = dsViper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var mongoCtx context.Context
		var cancelFn context.CancelFunc
		if ds.Options != nil && ds.Options.MongoDB != nil && ds.Options.MongoDB.Timeout > 0 {
			mongoCtx, cancelFn = context.WithTimeout(ds.Context, time.Duration(ds.Options.MongoDB.Timeout)*time.Second)
		} else {
			mongoCtx, cancelFn = context.WithCancel(ds.Context)
		}

		var clientOpts *options.ClientOptions

		url := getDbUrl(dsViper, ds)

		if dsViper.GetString(ds.Key+".username") != "" && dsViper.GetString(ds.Key+".password") != "" {
			credential := options.Credential{
				Username: dsViper.GetString(ds.Key + ".username"),
				Password: dsViper.GetString(ds.Key + ".password"),
			}
			clientOpts = options.Client().ApplyURI(url).SetAuth(credential)
		} else {
			clientOpts = options.Client().ApplyURI(url)
		}

		timeoutForOptions := time.Second * 30
		if ds.Options != nil && ds.Options.MongoDB != nil && ds.Options.MongoDB.Timeout > 0 {
			timeoutForOptions = time.Duration(ds.Options.MongoDB.Timeout) * time.Second
		}
		clientOpts = clientOpts.SetSocketTimeout(timeoutForOptions).SetConnectTimeout(timeoutForOptions).SetServerSelectionTimeout(timeoutForOptions).SetMinPoolSize(1).SetMaxPoolSize(5)

		if ds.Options != nil && ds.Options.MongoDB != nil && ds.Options.MongoDB.Registry != nil {
			clientOpts = clientOpts.SetRegistry(ds.Options.MongoDB.Registry)
		}

		if ds.Options != nil && ds.Options.MongoDB != nil && ds.Options.MongoDB.Monitor != nil {
			clientOpts = clientOpts.SetMonitor(ds.Options.MongoDB.Monitor)
		}

		fmt.Printf("Connecting to datasource %v...\n", ds.Key)
		db, err := mongo.Connect(mongoCtx, clientOpts)
		if err != nil {
			cancelFn()
			return err
		}
		ds.Db = db

		if ds.Options != nil && ds.Options.MongoDB != nil {
			if ds.Options.MongoDB.Timeout > 0 {
				fmt.Printf("DEBUG: Setting timeout to %v seconds\n", ds.Options.MongoDB.Timeout)
				mongoCtx, cancelFn = context.WithTimeout(context.Background(), time.Duration(ds.Options.MongoDB.Timeout)*time.Second)
			}
		}
		fmt.Printf("Pinging datasource %v...\n", ds.Key)
		err = ds.Db.(*mongo.Client).Ping(mongoCtx, readpref.SecondaryPreferred())
		if err != nil {
			fmt.Printf("Could not connect to datasource %v: %v\n", ds.Key, err)
			cancelFn()
			return err
		} else {
			fmt.Printf("DEBUG: Connected to datasource %v\n", ds.Key)
		}

		init := time.Now().UnixMilli()
		go func() {
			initialCtx := mongoCtx
			for {
				time.Sleep(time.Second * 5)

				if ds.Options != nil && ds.Options.MongoDB != nil && ds.Options.MongoDB.Timeout > 0 {
					mongoCtx, cancelFn = context.WithTimeout(initialCtx, time.Duration(ds.Options.MongoDB.Timeout)*time.Second)
				} else {
					mongoCtx, cancelFn = context.WithCancel(initialCtx)
				}

				err := ds.Db.(*mongo.Client).Ping(mongoCtx, readpref.SecondaryPreferred())
				if err != nil {
					url = getDbUrl(dsViper, ds)
					log.Printf("Reconnecting datasource %v...\n", ds.Key)
					db, err := mongo.Connect(mongoCtx, clientOpts)
					if err != nil {
						cancelFn()
						if ds.Options == nil || ds.Options.MongoDB == nil || !ds.Options.MongoDB.RetryOnError {
							log.Fatalf("Could not reconnect %v: %v\n", url, err)
						}
					} else {
						err = ds.Db.(*mongo.Client).Ping(mongoCtx, readpref.SecondaryPreferred())
						if err != nil {
							cancelFn()
							if ds.Options == nil || ds.Options.MongoDB == nil || !ds.Options.MongoDB.RetryOnError {
								log.Fatalf("Mongo client disconnected after %vms: %v", time.Now().UnixMilli()-init, err)
							}
						} else {
							log.Printf("successfully reconnected to %v\n", url)
						}

					}
					ds.Db = db
				}
			}
		}()
		break
	case "redis":
		return fmt.Errorf("redis connector not implemented yet")
	case "memorykv":
		ds.Db = memorykv.NewMemoryKvDb(memorykv.Options{
			Name: ds.Key,
		})
		// TODO: other setup operations
		break
	default:
		return errors.New("invalid connector " + connector)
	}
	return nil
}

func getDbUrl(dsViper *viper.Viper, ds *Datasource) string {
	url := ""
	if dsViper.GetString(ds.Key+".url") != "" {
		url = dsViper.GetString(ds.Key + ".url")
	} else {
		port := 0
		if dsViper.GetInt(ds.Key+".port") > 0 {
			port = dsViper.GetInt(ds.Key + ".port")
		}
		url = fmt.Sprintf("mongodb://%v:%v/%v", dsViper.GetString(ds.Key+".host"), port, dsViper.GetString(ds.Key+".database"))
		log.Printf("Using composed url %v\n", url)
	}
	return url
}

func (ds *Datasource) FindMany(collectionName string, lookups *wst.A) (*wst.A, error) {
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Viper.GetString(ds.Key + ".database"))
		collection := database.Collection(collectionName)

		pipeline := wst.A{}

		if lookups != nil {
			pipeline = append(pipeline, *lookups...)
		}
		allowDiskUse := true
		ctx := ds.Context
		cursor, err := collection.Aggregate(ctx, pipeline, &options.AggregateOptions{
			AllowDiskUse: &allowDiskUse,
		})
		if err != nil {
			return nil, err
		}
		defer func(cursor *mongo.Cursor, ctx context.Context) {
			err := cursor.Close(ctx)
			if err != nil {
				panic(err)
			}
		}(cursor, ctx)
		var documents wst.A
		err = cursor.All(ds.Context, &documents)
		if err != nil {
			return nil, err
		}
		return &documents, nil
	case "redis":
		return nil, fmt.Errorf("redis connector not implemented yet")
	case "memorykv":
		db := ds.Db.(memorykv.MemoryKvDb)
		if lookups == nil || len(*lookups) == 0 {
			return nil, errors.New("empty query")
		}

		potentialMatchStage := (*lookups)[0]

		var _id interface{}
		if match, isPresent := potentialMatchStage["$match"]; !isPresent {
			return nil, errors.New("invalid first stage for memorykv. First stage must contain $match")
		} else {
			if asM, ok := match.(wst.M); !ok {
				return nil, errors.New(fmt.Sprintf("invalid $match value type %s", asM))
			} else {
				if len(asM) == 0 {
					return nil, errors.New("empty $match")
				} else {
					for _, v := range asM {
						//key := fmt.Sprintf("%v:%v:%v", ds.Viper.GetString(ds.Keys+".database"), collectionName, k)
						_id = v
						break
					}
				}
			}
		}

		var idAsString string
		switch _id.(type) {
		case string:
			idAsString = _id.(string)
		case primitive.ObjectID:
			idAsString = _id.(primitive.ObjectID).Hex()
		case uuid.UUID:
			idAsString = _id.(uuid.UUID).String()
		}
		bucket := db.GetBucket(collectionName)
		bytes, err := bucket.Get(fmt.Sprintf("_id:%v", idAsString))
		if err != nil {
			return nil, err
		} else if bytes == nil {
			// TODO: Check if we should return an error or not
			return &wst.A{}, nil
		}
		var documents wst.A = make(wst.A, 1)
		err = bson.Unmarshal(bytes, &documents[0])
		if err != nil {
			return nil, err
		}
		return &documents, nil

	default:
		return nil, errors.New("invalid connector " + connector)
	}
}

func (ds *Datasource) Count(collectionName string, lookups *wst.A) (int64, error) {
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Viper.GetString(ds.Key + ".database"))
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
		ctx := ds.Context
		cursor, err := collection.Aggregate(ctx, pipeline, &options.AggregateOptions{
			AllowDiskUse: &allowDiskUse,
		})
		if err != nil {
			fmt.Printf("error %v\n", err)
			return 0, err
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
		err = cursor.All(ds.Context, &documents)
		if err != nil {
			return 0, err
		}
		if len(documents) == 0 {
			return 0, nil
		}
		return documents[0].Count, nil

	}
	return 0, errors.New(fmt.Sprintf("invalid connector %v", connector))
}

func findByObjectId(collectionName string, _id interface{}, ds *Datasource, lookups *wst.A) (*wst.M, error) {
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
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
		results, err := ds.FindMany(collectionName, wrappedLookups)
		if err != nil {
			return nil, err
		}
		if results != nil && len(*results) > 0 {
			return &(*results)[0], nil
		} else {
			return nil, errors.New("document not found")
		}
	case "redis":
		return nil, fmt.Errorf("redis connector not implemented yet")
	case "memorykv":
		db := ds.Db.(memorykv.MemoryKvDb)
		bucket := db.GetBucket(collectionName)
		var idAsString string
		switch _id.(type) {
		case string:
			idAsString = _id.(string)
		case primitive.ObjectID:
			idAsString = _id.(primitive.ObjectID).Hex()
		case uuid.UUID:
			idAsString = _id.(uuid.UUID).String()
		}
		var document wst.M
		bytes, err := bucket.Get(fmt.Sprintf("_id:%v", idAsString))
		if err != nil {
			return nil, err
		}

		err = bson.Unmarshal(bytes, &document)
		if err != nil {
			return nil, err
		}
		return &document, nil
	default:
		return nil, errors.New("invalid connector " + connector)
	}
}

func (ds *Datasource) Create(collectionName string, data *wst.M) (*wst.M, error) {
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Viper.GetString(ds.Key + ".database"))
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
		if (*data)["_id"] == nil {
			id = uuid.New()
			(*data)["_id"] = id
		} else {
			id = (*data)["_id"]
		}
		var idAsStr string
		switch id.(type) {
		case string:
			idAsStr = id.(string)
		case primitive.ObjectID:
			idAsStr = id.(primitive.ObjectID).Hex()
		case uuid.UUID:
			idAsStr = id.(uuid.UUID).String()
		}

		var dataAsBytes []byte
		dataAsBytes, err := bson.Marshal(data)
		if err != nil {
			return nil, err
		}

		//dict[id] = data
		err = dict.GetBucket(collectionName).Set(idAsStr, dataAsBytes)
		if err != nil {
			return nil, err
		}

		return findByObjectId(collectionName, id, ds, nil)
	}
	return nil, errors.New(fmt.Sprintf("invalid connector %v", connector))
}

func (ds *Datasource) UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error) {
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Viper.GetString(ds.Key + ".database"))
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
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Viper.GetString(ds.Key + ".database"))
		collection := database.Collection(collectionName)
		if result, err := collection.DeleteOne(ds.Context, wst.M{"_id": id}); err != nil {
			panic(err)
		} else {
			return result.DeletedCount
		}
	}
	return 0
}

func New(dsKey string, dsViper *viper.Viper, parentContext context.Context) *Datasource {
	name := dsViper.GetString(dsKey + ".name")
	if name == "" {
		name = dsKey
	}
	ctx, ctxCancelFn := context.WithCancel(parentContext)
	ds := &Datasource{
		Name:  name,
		Viper: dsViper,

		Key: dsKey,

		Context:     ctx,
		ctxCancelFn: ctxCancelFn,
	}
	return ds
}
