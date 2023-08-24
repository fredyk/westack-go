package datasource

import (
	"context"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/spf13/viper"
)

// DeleteManyResult is the result of a DeleteMany operation.
type DeleteManyResult struct {
	// DeletedCount is the number of documents deleted.
	DeletedCount int64
}

type PersistedConnector interface {
	// GetName Returns the name of the connector
	GetName() string
	// SetConfig Sets the configuration for the datasource
	SetConfig(dsViper *viper.Viper)
	// Connect Connects to the datasource
	Connect(parentContext context.Context) error
	// FindMany Finds many documents in the datasource
	FindMany(collectionName string, lookups *wst.A) (MongoCursorI, error)
	// findObjectById Finds a document in the datasource
	findByObjectId(collectionName string, _id interface{}, lookups *wst.A) (*wst.M, error)
	// Count Counts documents in the datasource
	Count(collectionName string, lookups *wst.A) (int64, error)
	// Create Creates a document in the datasource
	Create(collectionName string, data *wst.M) (*wst.M, error)
	// UpdateById Updates a document in the datasource
	UpdateById(collectionName string, id interface{}, data *wst.M) (*wst.M, error)
	// DeleteById Deletes a document in the datasource
	DeleteById(collectionName string, id interface{}) int64
	// DeleteMany Deletes many documents in the datasource
	DeleteMany(collectionName string, whereLookups *wst.A) (DeleteManyResult, error)
	// Disconnect Disconnects from the datasource
	Disconnect() error
	// Ping Pings the datasource
	Ping(parentCtx context.Context) error
	// GetClient Returns the client for the datasource
	GetClient() interface{}
}
