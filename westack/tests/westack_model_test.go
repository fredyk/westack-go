package tests

import (
	"testing"
	"time"

	"github.com/fredyk/westack-go/westack/model"

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

	timeNow := time.Now()
	created, err := noteModel.Create(&map[string]interface{}{
		"date": timeNow,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, timeNow.Minute(), created.ToJSON()["date"].(primitive.DateTime).Time().Minute())
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

	result, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"__forceError": true,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, result, "error")
}

func Test_CreateWithOverrideInvalid(t *testing.T) {

	t.Parallel()

	result, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"__overwriteWith": 1,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, result, "error")
}

func Test_CreateWithInvalidBsonInput(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{
		"invalid": make(chan int),
	}, systemContext)
	assert.Errorf(t, err, "Should not be able to create with invalid bson <-- %v", created)
}

func Test_CreateWithForcingError(t *testing.T) {

	t.Parallel()

	result, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"__forceAfterError": true,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, result, "error")
}

func Test_EnforceExError(t *testing.T) {

	t.Parallel()

	err, _ := noteModel.EnforceEx(nil, "", "create", &model.EventContext{})
	assert.Error(t, err)
}

func Test_CreateWithDefaultStringValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Equal(t, "default", created.GetString("defaultString"))
}

func Test_CreateWithDefaultIntValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.EqualValues(t, 1, created["defaultInt"])
}

func Test_CreateWithDefaultFloatValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.EqualValues(t, 87436874647.8761781676, created["defaultFloat"])
}

func Test_CreateWithDefaultBooleanValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.EqualValues(t, true, created["defaultBoolean"])
}

func Test_CreateWithDefaultListValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultList")
	assert.IsType(t, []interface{}{}, created["defaultList"])
	assert.EqualValues(t, []interface{}{"default"}, created["defaultList"])
}

func Test_CreateWithDefaultMapValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultMap")
	assert.IsType(t, map[string]interface{}{}, created["defaultMap"])
	assert.Contains(t, created["defaultMap"].(map[string]interface{}), "defaultKey")
	assert.Equal(t, map[string]interface{}{"defaultKey": "defaultValue"}, created["defaultMap"].(map[string]interface{}))
}

func Test_CreateWithDefaultTimeValue(t *testing.T) {

	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix()
	// Should be 15 milliseconds after at most
	upperSeconds := probablyTime.Unix() + 3
	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultTimeNow")
	var parsedTime time.Time
	parsedTime, err = time.Parse(time.RFC3339, created["defaultTimeNow"].(string))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, parsedTime.Unix(), lowerSeconds)
	assert.LessOrEqual(t, parsedTime.Unix(), upperSeconds)

}

func Test_CreateWithDefaultTimeHourAgo(t *testing.T) {
	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix() - 3600
	// Should be 15 milliseconds after at most
	upperSeconds := probablyTime.Unix() - 3597
	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultTimeHourAgo")
	var parsedTime time.Time
	parsedTime, err = time.Parse(time.RFC3339, created["defaultTimeHourAgo"].(string))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, parsedTime.Unix(), lowerSeconds)
	assert.LessOrEqual(t, parsedTime.Unix(), upperSeconds)

}

func Test_CreateWithDefaultTimeHourFromNow(t *testing.T) {

	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix() + 3600
	// Should be 15 milliseconds after at most
	upperSeconds := probablyTime.Unix() + 3603
	created, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultTimeHourFromNow")
	var parsedTime time.Time
	parsedTime, err = time.Parse(time.RFC3339, created["defaultTimeHourFromNow"].(string))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, parsedTime.Unix(), lowerSeconds)
	assert.LessOrEqual(t, parsedTime.Unix(), upperSeconds)

}
