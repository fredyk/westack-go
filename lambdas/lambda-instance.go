package lambdas

import (
	"fmt"

	"github.com/fredyk/westack-go/lambdas/entities"
	modelentities "github.com/fredyk/westack-go/lambdas/entities/model-entities"
)

var (
	_ modelentities.Instance = &lambdaRemoteInstance{}
)

type lambdaRemoteInstance struct {
	data *entities.M
}

func (rtInstance *lambdaRemoteInstance) GetID() interface{} {
	return fmt.Errorf("not implemented")
}

func (rtInstance *lambdaRemoteInstance) UpdateAttributes(data interface{}, baseContext *modelentities.EventContext) (modelentities.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rtInstance *lambdaRemoteInstance) ToJSON() entities.M {
	return *rtInstance.data
}

func (rtInstance *lambdaRemoteInstance) Get(relationName string) interface{} {
	return fmt.Errorf("not implemented")
}

func (rtInstance *lambdaRemoteInstance) GetM(path string) *entities.M {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetA(path string) []entities.M {
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

func (rtInstance *lambdaRemoteInstance) GetObjectId(path string) any {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetOne(relation string) modelentities.Instance {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetMany(relation string) []modelentities.Instance {
	return nil
}

func (rtInstance *lambdaRemoteInstance) GetModel() modelentities.Model {
	return nil
}
