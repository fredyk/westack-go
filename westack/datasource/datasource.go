package datasource

import (
	"context"
	"errors"
	"fmt"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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
	Name  string
	Db    interface{}
	Viper *viper.Viper

	Key         string
	Context     context.Context
	ctxCancelFn context.CancelFunc
}

func (ds *Datasource) Initialize() error {
	dsViper := ds.Viper
	var connector = dsViper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		mongoCtx, cancelFn := context.WithCancel(ds.Context)

		var clientOpts *options.ClientOptions

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

		if dsViper.GetString(ds.Key+".username") != "" && dsViper.GetString(ds.Key+".password") != "" {
			credential := options.Credential{
				Username: dsViper.GetString(ds.Key + ".username"),
				Password: dsViper.GetString(ds.Key + ".password"),
			}
			clientOpts = options.Client().ApplyURI(url).SetAuth(credential)
		} else {
			clientOpts = options.Client().ApplyURI(url)
		}

		clientOpts = clientOpts.SetSocketTimeout(time.Second * 30).SetConnectTimeout(time.Second * 30).SetServerSelectionTimeout(time.Second * 30).SetMinPoolSize(1).SetMaxPoolSize(5)

		db, err := mongo.Connect(mongoCtx, clientOpts)
		if err != nil {
			cancelFn()
			return err
		}
		ds.Db = db

		err = ds.Db.(*mongo.Client).Ping(mongoCtx, readpref.SecondaryPreferred())
		if err != nil {
			cancelFn()
			return err
		}

		init := time.Now().UnixMilli()
		go func() {
			for {
				time.Sleep(time.Second * 5)

				mongoCtx, cancelFn = context.WithCancel(mongoCtx)
				err := ds.Db.(*mongo.Client).Ping(mongoCtx, readpref.SecondaryPreferred())
				if err != nil {
					log.Printf("Reconnecting %v...\n", url)
					db, err := mongo.Connect(mongoCtx, clientOpts)
					if err != nil {
						cancelFn()
						log.Fatalf("Could not reconnect %v: %v\n", url, err)
					} else {
						err = ds.Db.(*mongo.Client).Ping(mongoCtx, readpref.SecondaryPreferred())
						if err != nil {
							cancelFn()
							log.Fatalf("Mongo client disconnected after %vms: %v", time.Now().UnixMilli()-init, err)
						}

						log.Printf("successfully reconnected to %v\n", url)
					}
					ds.Db = db
				}
			}
		}()
		break
	case "redis":

		// Create redis client
		var rClient *redis.Client = redis.NewClient(&redis.Options{
			Addr:     dsViper.GetString(ds.Key + ".url"),
			Password: dsViper.GetString(ds.Key + ".password"), // no password set
			DB:       dsViper.GetInt(ds.Key + ".database"),    // use default DB
		})
		ds.Db = rClient

		break
	//case "memory":
	//	ds.Db = make(map[interface{}]interface{})
	//	break
	default:
		return errors.New("Invalid connector " + connector)
	}
	return nil
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
		var rClient = ds.Db.(*redis.Client)

		if lookups == nil || len(*lookups) == 0 {
			return nil, errors.New("empty query")
		}

		potentialMatchStage := (*lookups)[0]

		var _id interface{}
		if match, isPresent := potentialMatchStage["$match"]; !isPresent {
			return nil, errors.New("invalid first stage for redis. First stage must contain $match")
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

		cmd := rClient.Get(ds.Context, fmt.Sprintf("%v:%v:%v", ds.Viper.GetString(ds.Key+".database"), collectionName, _id))
		err := cmd.Err()

		if err != nil {
			if err.Error() == "redis: nil" {
				return &wst.A{}, nil
			} else {
				return nil, err
			}
		}

		bytes, err := cmd.Bytes()
		if err != nil {
			return nil, err
		}

		out := make(wst.A, 1)
		err = bson.Unmarshal(bytes, &out[0])
		if err != nil {
			return nil, err
		}

		return &out, nil
	}
	return nil, errors.New(fmt.Sprintf("invalid connector %v", connector))
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
	//case "memory":
	//	if result, isPresent := ds.Db.(map[interface{}]wst.M)[_id]; isPresent {
	//		return &result, nil
	//	} else {
	//		return ???
	//	}
	case "redis":
		var rClient = ds.Db.(*redis.Client)

		cmd := rClient.Get(ds.Context, fmt.Sprintf("%v:%v:%v", ds.Viper.GetString(ds.Key+".database"), collectionName, _id))
		err := cmd.Err()

		if err != nil {
			return nil, err
		}

		bytes, err := cmd.Bytes()
		if err != nil {
			return nil, err
		}

		var out wst.M
		err = bson.Unmarshal(bytes, &out)
		if err != nil {
			return nil, err
		}

		return &out, nil
	}
	return nil, errors.New(fmt.Sprintf("invalid connector %v", connector))
}

func (ds *Datasource) Create(collectionName string, data *wst.M) (*wst.M, error) {
	var connector = ds.Viper.GetString(ds.Key + ".connector")
	switch connector {
	case "mongodb":
		var db = ds.Db.(*mongo.Client)

		database := db.Database(ds.Viper.GetString(ds.Key + ".database"))
		collection := database.Collection(collectionName)
		insertOneResult, err := collection.InsertOne(ds.Context, data)
		if err != nil {
			return nil, err
		}
		return findByObjectId(collectionName, insertOneResult.InsertedID, ds, nil)
	case "redis":
		var rClient = ds.Db.(*redis.Client)

		var id interface{}

		if (*data)["_redId"] == nil {
			id = uuid.New().String()
			(*data)["_redId"] = id
		} else {
			id = (*data)["_redId"]
		}

		bytes, err := bson.Marshal(data)
		if err != nil {
			return nil, err
		}

		//key := fmt.Sprintf("%v:%v:%v", ds.Viper.GetString(ds.Key+".database"), collectionName, id)
		key := fmt.Sprintf("%v:%v:%v", ds.Viper.GetString(ds.Key+".database"), collectionName, id)

		statusCmd := rClient.Set(ds.Context, key, bytes, 0)
		err = statusCmd.Err()
		if err != nil {
			return nil, err
		}
		return findByObjectId(collectionName, id, ds, nil)
		//case "memory":
		//	dict := ds.Db.(map[interface{}]interface{})
		//
		//	var id interface{}
		//	if (*data)["_id"] == nil {
		//		id = uuid.New()
		//		(*data)["_id"] = id
		//	} else {
		//		id = (*data)["_id"]
		//	}
		//
		//	dict[id] = data
		//
		//	return findByObjectId(collectionName, id, ds, nil)
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
		} else if wst.IsAnyDate(data) {
			newValue, err = wst.ParseDate(data)
		}
		if err != nil {
			log.Println("WARNING: ", err)
		}
		if newValue != nil {
			return newValue
		} else {
			return data
		}
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
							asList, asMListOk := value.([]wst.M)
							if asMListOk {
								for i, asListItem := range asList {
									asList[i] = ReplaceObjectIds(asListItem).(wst.M)
								}
							} else {
								log.Println(fmt.Sprintf("WARNING: What to do with %v (%s)?", value, value))
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
				log.Println(fmt.Sprintf("WARNING: invalid input ReplaceObjectIds() <- %s", data))
				break
			}
		} else if err != nil {
			log.Println("WARNING: ", err)
		}
	}
	return data
}
