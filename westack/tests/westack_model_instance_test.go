package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/model"
)

func Test_ToJSON_Nil(t *testing.T) {
	var instance *model.Instance
	json := instance.ToJSON()
	assert.Nil(t, json)
}

func Test_ToJSON_NilInstance(t *testing.T) {
	m := model.New(&model.Config{}, &map[string]*model.Model{})
	instance := m.NilInstance
	json := instance.ToJSON()
	assert.Equal(t, wst.NilMap, json)
}

func Test_ToJSON_Relations(t *testing.T) {
	m, err := server.FindModel("Note")
	if err != nil {
		t.Error(err)
		return
	}
	systemContext := &model.EventContext{
		Bearer: &model.BearerToken{User: &model.BearerUser{System: true}},
	}
	instance := m.Build(wst.M{
		"_id":    noteId,
		"userId": userId,
		"user": wst.M{
			"id": userId,
		},
	}, systemContext)
	json := instance.ToJSON()
	user := json.GetM("user")
	assert.NotNil(t, user)
	assert.Equal(t, userId.Hex(), user["id"].(primitive.ObjectID).Hex())

}
