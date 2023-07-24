package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/model"
)

func Test_ToJSON_Nil(t *testing.T) {

	t.Parallel()

	var instance *model.Instance
	json := instance.ToJSON()
	assert.Nil(t, json)
}

func Test_ToJSON_NilInstance(t *testing.T) {

	t.Parallel()

	m := model.New(&model.Config{}, &map[string]*model.Model{})
	instance := m.NilInstance
	json := instance.ToJSON()
	assert.Equal(t, wst.NilMap, json)
}

func Test_ToJSON_BelongsToRelation(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"_id":    noteId,
		"userId": userId,
		"user": wst.M{
			"id": userId,
		},
	}, model.NewBuildCache(), systemContext)
	assert.NoError(t, err)
	originalUser := instance.GetOne("user")
	json := instance.ToJSON()
	user := json.GetM("user")
	assert.NotNil(t, user)
	assert.Equal(t, userId.Hex(), user["id"].(primitive.ObjectID).Hex())
	assert.Equal(t, originalUser.GetObjectId("id").Hex(), user["id"].(primitive.ObjectID).Hex())

}

func Test_ToJSON_HasManyRelation(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"_id": noteId,
		"entries": primitive.A{
			wst.M{"date": "2021-01-01", "text": "Entry 1"},
		},
	}, model.NewBuildCache(), systemContext)
	assert.NoError(t, err)
	originalEntries := instance.GetMany("entries")
	json := instance.ToJSON()
	entries := json["entries"]
	assert.NotNil(t, entries)
	assert.Equal(t, 1, len(entries.(wst.A)))
	assert.Equal(t, "2021-01-01", entries.(wst.A)[0]["date"])
	assert.Equal(t, originalEntries[0].GetString("date"), entries.(wst.A)[0]["date"])
}

func Test_Access_Empty_Relation(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"_id": noteId,
	}, model.NewBuildCache(), systemContext)
	assert.Nil(t, err)
	// assert user is nil
	assert.Nil(t, instance.GetOne("user"))
	// assert entries is empty
	assert.Equal(t, 0, len(instance.GetMany("entries")))
}

type Entry struct {
	Date string `bson:"date"`
	Text string `bson:"text"`
}
type User struct {
	Id primitive.ObjectID `bson:"id"`
}

func Test_Instance_Transform(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"_id":    noteId,
		"userId": userId,
		"user": wst.M{
			"id": userId,
		},
		"entries": primitive.A{
			wst.M{"date": "2021-01-01", "text": "Entry 1"},
		},
	}, model.NewBuildCache(), systemContext)
	assert.Nil(t, err)
	var out struct {
		Id      primitive.ObjectID `bson:"id"`
		UserId  primitive.ObjectID `bson:"userId"`
		User    User               `bson:"user"`
		Entries []Entry            `bson:"entries"`
	}
	err = instance.Transform(&out)
	assert.Nil(t, err)
	assert.Equal(t, noteId.Hex(), out.Id.Hex())
	assert.Equal(t, userId.Hex(), out.UserId.Hex())
	assert.Equal(t, userId.Hex(), out.User.Id.Hex())
	assert.Equal(t, 1, len(out.Entries))
	assert.Equal(t, "2021-01-01", out.Entries[0].Date)
	assert.Equal(t, "Entry 1", out.Entries[0].Text)
}

func Test_Instance_Transform_Error(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"entries": primitive.A{
			wst.M{"date": "2021-01-01"},
		},
	}, model.NewBuildCache(), systemContext)
	assert.Nil(t, err)
	var out struct {
		Entries []struct {
			Date chan string `bson:"date"`
		}
	}
	noteModel.App.Debug = false
	err = instance.Transform(&out)
	noteModel.App.Debug = true
	assert.NotNil(t, err)
}

func Test_Instance_UncheckedTransform(t *testing.T) {

	t.Parallel()

	// recover the panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in f %v\n", r)
			t.Errorf("The code did panic")
		}
	}()

	instance, err := noteModel.Build(wst.M{
		"entries": primitive.A{
			wst.M{"date": "2021-01-01"},
		},
	}, model.NewBuildCache(), systemContext)
	assert.Nil(t, err)
	type SafeType struct {
		Entries []struct {
			Date string `bson:"date"`
		}
	}
	out := instance.UncheckedTransform(new(SafeType))
	assert.NotNil(t, out)
}

func Test_Instance_UncheckedTransform_Panic(t *testing.T) {

	t.Parallel()

	// recover the panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	instance, err := noteModel.Build(wst.M{
		"entries": primitive.A{
			wst.M{"date": "2021-01-01"},
		},
	}, model.NewBuildCache(), systemContext)
	assert.Nil(t, err)
	type UnsafeType struct {
		Entries []struct {
			Date chan string `bson:"date"`
		}
	}
	instance.Model.App.Debug = false
	instance.UncheckedTransform(new(UnsafeType))
	instance.Model.App.Debug = true
}

func Test_UpdateAttributes(t *testing.T) {

	t.Parallel()

	createdNote, err := noteModel.Create(wst.M{
		"title": "Old title",
	}, systemContext)
	assert.Nil(t, err)

	updated, err := createdNote.UpdateAttributes(wst.M{
		"title": "Title from wst.M",
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Title from wst.M", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(&wst.M{
		"title": "Title from *wst.M",
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Title from *wst.M", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(map[string]interface{}{
		"title": "Title from map[string]interface{}",
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Title from map[string]interface{}", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(&map[string]interface{}{
		"title": "Title from *map[string]interface{}",
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Title from *map[string]interface{}", updated.GetString("title"))

	secondNote, err := noteModel.Create(wst.M{
		"title": "Second note",
	}, systemContext)
	assert.Nil(t, err)

	updated, err = createdNote.UpdateAttributes(*secondNote, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Second note", updated.GetString("title"))

	_, err = secondNote.UpdateAttributes(wst.M{
		"title": "Second note updated for *Instance",
	}, systemContext)
	assert.Nil(t, err)

	updated, err = createdNote.UpdateAttributes(secondNote, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Second note updated for *Instance", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(struct {
		Title string `bson:"title"`
	}{
		Title: "Title from struct",
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Title from struct", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(&struct {
		Title chan string `bson:"title"`
	}{
		Title: make(chan string),
	}, systemContext)
	assert.NotNil(t, err)

	updated, err = createdNote.UpdateAttributes("invalid type", systemContext)
	assert.NotNil(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"title": "Title from wst.M",
	}, nil)
	assert.Nil(t, err)
	assert.Equal(t, "Title from wst.M", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(wst.M{
		"title": "Title from wst.M 2",
	}, &model.EventContext{BaseContext: systemContext})
	assert.Nil(t, err)
	assert.Equal(t, "Title from wst.M 2", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__forceError": true,
	}, systemContext)
	assert.NotNil(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": wst.M{
			"title": "Overwritten title",
		},
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Overwritten title", updated.GetString("title"))

	_, err = secondNote.UpdateAttributes(wst.M{
		"title": "Second note updated for overwrite with Instance",
	}, systemContext)
	assert.Nil(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": *secondNote,
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Second note updated for overwrite with Instance", updated.GetString("title"))

	_, err = secondNote.UpdateAttributes(wst.M{
		"title": "Second note updated for overwrite with *Instance",
	}, systemContext)
	assert.Nil(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": secondNote,
	}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "Second note updated for overwrite with *Instance", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": "invalid type",
	}, systemContext)
	assert.NotNil(t, err)

}
