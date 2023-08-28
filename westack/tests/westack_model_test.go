package tests

import (
	"github.com/fredyk/westack-go/westack/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_CreateWithMap(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(map[string]interface{}{
		"title": "Test",
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Test", created.GetString("title"))
}

func Test_CreateWithMapPointer(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(&map[string]interface{}{
		"date": time.Now(),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, time.Now().Minute(), created.ToJSON()["date"].(primitive.DateTime).Time().Minute())
}

func Test_CreateWithStruct(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(struct {
		SomeInt int `bson:"someInt"`
	}{
		SomeInt: 1,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), created.GetInt("someInt"))
}

func Test_CreateWithM(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{
		"someFloat": 1.1,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 1.1, created.GetFloat64("someFloat"))
}

func Test_CreateWithMPointer(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(&wst.M{
		"someBoolean": true,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, true, created.GetBoolean("someBoolean", false))
}

func Test_CreateWithInstance(t *testing.T) {

	t.Parallel()

	build, err2 := noteModel.Build(wst.M{
		"foo": "bar",
	}, model.NewBuildCache(), systemContext)
	assert.Nil(t, err2)
	created, err := noteModel.Create(build, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "bar", created.GetString("foo"))
}

func Test_CreateWithInstancePointer(t *testing.T) {

	t.Parallel()

	v, err := noteModel.Build(wst.M{
		"foo2": "bar2",
	}, model.NewBuildCache(), systemContext)
	assert.NoError(t, err)
	created, err := noteModel.Create(&v, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "bar2", created.GetString("foo2"))
}

func Test_CreateWithBadStruct(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(struct {
		SomeInt chan int `bson:"someInt"`
	}{
		SomeInt: make(chan int),
	}, systemContext)
	assert.Error(t, err)
}

func Test_CreateWithInvalidInput(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(1, systemContext)
	assert.Error(t, err)
}

func Test_CreateWithOverrideResultAsM(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{
		"__overwriteWith": wst.M{
			"overrided1": true,
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, true, created.GetBoolean("overrided1", false))
}

func Test_CreateWithOverrideResultAsInstance(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(wst.M{
		"__overwriteWithInstance": wst.M{
			"overrided2": wst.M{
				"overrided3": true,
			},
		},
	}, systemContext)
	assert.NoError(t, err)
}

func Test_CreateWithOverrideResultAsInstancePointer(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(wst.M{
		"__overwriteWithInstancePointer": wst.M{
			"overrided4": wst.M{
				"overrided5": true,
			},
		},
	}, systemContext)
	assert.NoError(t, err)
}

func Test_CreateWithOverrideResultError(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(wst.M{
		"__forceError": true,
	}, systemContext)
	assert.Error(t, err)
}

func Test_CreateWithOverrideInvalid(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(wst.M{
		"__overwriteWith": 1,
	}, systemContext)
	assert.Error(t, err)
}

func Test_CreateWithInvalidBsonInput(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{
		"invalid": make(chan int),
	}, systemContext)
	assert.NotNilf(t, err, "Should not be able to create with invalid bson <-- %v", created)
}

func Test_CreateWithForcingError(t *testing.T) {

	t.Parallel()

	_, err := noteModel.Create(wst.M{
		"__forceAfterError": true,
	}, systemContext)
	assert.Error(t, err)
}

func Test_EnforceExError(t *testing.T) {

	t.Parallel()

	err, _ := noteModel.EnforceEx(nil, "", "create", &model.EventContext{})
	assert.Error(t, err)
}
