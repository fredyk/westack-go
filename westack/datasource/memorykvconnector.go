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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MemoryKVConnector implements the PersistedConnector interface
type MemoryKVConnector struct {
	db       memorykv.MemoryKvDb
	dsKey    string
	dsConfig *viper.Viper
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
	return NewFixedMongoCursor(documents), nil
}

func (connector *MemoryKVConnector) findObjectById(collectionName string, _id interface{}, lookups *wst.A) (*wst.M, error) {
	db := connector.db
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
	allBytes, err := bucket.Get(idAsString)
	if err != nil {
		return nil, err
	}
	if len(allBytes) == 0 {
		return nil, errors.New("document not found")
	} else if len(allBytes) > 1 {
		return nil, errors.New("multiple documents found")
	} else {
		err = bson.Unmarshal(allBytes[0], &document)
		if err != nil {
			return nil, err
		}
		return &document, nil
	}
}

func (connector *MemoryKVConnector) Count(collectionName string, lookups *wst.A) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) Create(collectionName string, data *wst.M) (*wst.M, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) DeleteById(collectionName string, id interface{}) int64 {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) DeleteMany(collectionName string, whereLookups *wst.A) (DeleteManyResult, error) {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (connector *MemoryKVConnector) Ping(parentCtx context.Context) error {
	// We don't need to ping memorykv
	return nil
}

func (connector *MemoryKVConnector) GetClient() interface{} {
	return connector.db
}

// NewMemoryKVConnector Factory method for MemoryKVConnector
func NewMemoryKVConnector(dsKey string) PersistedConnector {
	return &MemoryKVConnector{
		dsKey: dsKey,
	}
}
