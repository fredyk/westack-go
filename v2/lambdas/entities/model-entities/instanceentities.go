package modelentities

import (
	"github.com/fredyk/westack-go/v2/lambdas/entities"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Instance interface {
	GetID() interface{}
	UpdateAttributes(data interface{}, baseContext *EventContext) (Instance, error)
	ToJSON() entities.M
	Get(relationName string) interface{}
	GetA(path string) []entities.M
	GetM(path string) *entities.M
	GetString(path string) string
	GetInt(path string) int64
	GetFloat64(path string) float64
	GetBoolean(path string, defaultValue bool) bool
	GetObjectId(path string) primitive.ObjectID
	GetOne(relation string) Instance
	GetMany(relation string) []Instance
	GetModel() Model
}
