package datasource

import (
	"context"
	"errors"
	"fmt"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/memorykv"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// MemoryKVConnector implements the PersistedConnector interface
type MemoryKVConnector struct {
	db       memorykv.MemoryKvDb
	dsKey    string
	dsConfig *viper.Viper
	registry *bsoncodec.Registry
}

func (connector *MemoryKVConnector) GetName() string {
	return "memorykv"
}

func (connector *MemoryKVConnector) Connect(parentContext context.Context) error {
	connector.db = memorykv.NewMemoryKvDb(memorykv.Options{
		Name: connector.dsKey,
	})
	return nil
}

func (connector *MemoryKVConnector) SetConfig(dsViper *viper.Viper) {
	connector.dsConfig = dsViper
}

func (connector *MemoryKVConnector) FindMany(collectionName string, lookups *wst.A) (MongoCursorI, error) {
	db := connector.db
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

	// fmt.Println("QUERYING CACHE: collection=", collectionName, "id=", idAsString) TODO: check debug

	bytes, err := bucket.Get(idAsString)
	var documents [][]byte
	if err != nil {
		return nil, err
	} else if bytes == nil {
		// TODO: Check if we should return an error or not
		//return &wst.A{}, nil
		documents = nil
	} else {
		documents = bytes
	}
	return NewFixedMongoCursor(connector.registry, documents), nil
}

func (connector *MemoryKVConnector) findByObjectId(collectionName string, _id interface{}, lookups *wst.A) (*wst.M, error) {
	panic("Not implemented")
}

func (connector *MemoryKVConnector) Count(collectionName string, lookups *wst.A) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) Create(collectionName string, data *wst.M) (*wst.M, error) {
	db := connector.db

	var id interface{}

	var allBytes [][]byte
	var idAsStr string
	if (*data)["_redId"] == nil {
		id = uuid.New().String()
		(*data)["_redId"] = id
	} else {
		id = (*data)["_redId"]
	}
	var finalEntries wst.A
	if v, ok := (*data)["_entries"].(wst.A); ok {
		finalEntries = v
	} else if v, ok := (*data)["_entries"].([]interface{}); ok {
		finalEntries = wst.A{}
		for _, entry := range v {
			var vv wst.M
			if vvv, ok := entry.(wst.M); ok {
				vv = vvv
			} else if vvv, ok := entry.(map[string]interface{}); ok {
				vv = vvv
			} else {
				return nil, errors.New(fmt.Sprintf("invalid _entries type %v", (*data)["_entries"]))
			}
			finalEntries = append(finalEntries, vv)
		}
	} else {
		return nil, errors.New(fmt.Sprintf("invalid _entries type %v", (*data)["_entries"]))
	}
	for _, doc := range finalEntries {
		switch id.(type) {
		case string:
			idAsStr = id.(string)
		case primitive.ObjectID:
			idAsStr = id.(primitive.ObjectID).Hex()
		}

		bytes, err := bson.MarshalWithRegistry(connector.registry, doc)
		if err != nil {
			return nil, err
		}
		allBytes = append(allBytes, bytes)
	}

	err := db.GetBucket(collectionName).SetEx(idAsStr, allBytes, 365*86400*time.Second)
	return data, err
}

func (connector *MemoryKVConnector) UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) DeleteById(collectionName string, id interface{}) (DeleteResult, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) DeleteMany(collectionName string, whereLookups *wst.A) (DeleteResult, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) Disconnect() error {
	// Clear memory of buckets
	return connector.db.Purge()
}

func (connector *MemoryKVConnector) Ping(parentCtx context.Context) error {
	// We don't need to ping memorykv
	return nil
}

func (connector *MemoryKVConnector) SetTimeout(seconds float32) {
	// We don't need to set timeout for memorykv
}

func (connector *MemoryKVConnector) GetClient() interface{} {
	return connector.db
}

// NewMemoryKVConnector Factory method for MemoryKVConnector
func NewMemoryKVConnector(registry *bsoncodec.Registry, dsKey string) PersistedConnector {
	return &MemoryKVConnector{
		dsKey:    dsKey,
		registry: registry,
	}
}
