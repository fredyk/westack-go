package modelentities

import "github.com/fredyk/westack-go/v2/lambdas/entities"

type Model interface {
	FindMany(filterMap *entities.Filter, currentContext *EventContext) Cursor
	FindById(id interface{}, filterMap *entities.Filter, baseContext *EventContext) (Instance, error)
	Create(data interface{}, currentContext *EventContext) (Instance, error)
	Count(filterMap *entities.Filter, currentContext *EventContext) (CountResult, error)
	DeleteById(id interface{}, currentContext *EventContext) (DeleteResult, error)
	UpdateById(id interface{}, data interface{}, currentContext *EventContext) (Instance, error)
	GetConfig() *Config
	GetName() string
}

// DeleteResult is the result of a DeleteMany operation.
type DeleteResult struct {
	// DeletedCount is the number of documents deleted.
	DeletedCount int64 `json:"deletedCount"`
}

// CountResult is the result of a Count operation.
type CountResult struct {
	// Count is the number of documents.
	Count int64 `json:"count"`
}

// LoginResult is the result of a login operation.
type LoginResult struct {
	Id        string `json:"id"`
	AccountId string `json:"accountId"`
}
