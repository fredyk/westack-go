package lambdas

import (
	"fmt"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	_ model.Instance = &lambdaRemoteInstance{}
)

type lambdaRemoteInstance struct {
	data *wst.M
}

func (rtInstance *lambdaRemoteInstance) GetID() interface{} {
	return fmt.Errorf("not implemented")
}

func (rtInstance *lambdaRemoteInstance) UpdateAttributes(data interface{}, baseContext *model.EventContext) (model.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rtInstance *lambdaRemoteInstance) ToJSON() wst.M {
	return *rtInstance.data
}

func (rtInstance *lambdaRemoteInstance) Get(relationName string) interface{} {
	return fmt.Errorf("not implemented")
}

func (rtInstance *lambdaRemoteInstance) GetM(path string) *wst.M {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetA(path string) *wst.A {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetString(path string) string {
	return "not implemented"
}

func (rtInstance *lambdaRemoteInstance) GetInt(path string) int64 {
	return -1
}

func (rtInstance *lambdaRemoteInstance) GetFloat64(path string) float64 {
	return -1.0
}

func (rtInstance *lambdaRemoteInstance) GetBoolean(path string, defaultValue bool) bool {
	return false
}

func (rtInstance *lambdaRemoteInstance) GetObjectId(path string) primitive.ObjectID {
	return primitive.NilObjectID
}

func (rtInstance *lambdaRemoteInstance) GetOne(relation string) model.Instance {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetMany(relation string) model.InstanceA {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetModel() model.Model {
	return nil
}
