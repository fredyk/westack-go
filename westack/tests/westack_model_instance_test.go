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
	}, systemContext)
	assert.NoError(t, err)
	originalUser := instance.GetOne("user")
	json := instance.ToJSON()
	user := json.GetM("user")
	assert.NotNil(t, user)
	assert.Equal(t, userId.Hex(), user.GetString("id"))
	assert.Equal(t, originalUser.GetObjectId("id").Hex(), user.GetString("id"))

}

func Test_ToJSON_HasManyRelation(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"_id": noteId,
		"entries": primitive.A{
			wst.M{"date": "2021-01-01", "text": "Entry 1"},
		},
	}, systemContext)
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
	}, systemContext)
	assert.NoError(t, err)
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
	}, systemContext)
	assert.NoError(t, err)
	var out struct {
		Id      primitive.ObjectID `bson:"id"`
		UserId  primitive.ObjectID `bson:"userId"`
		User    User               `bson:"user"`
		Entries []Entry            `bson:"entries"`
	}
	err = instance.Transform(&out)
	assert.NoError(t, err)
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
	}, systemContext)
	assert.NoError(t, err)
	var out struct {
		Entries []struct {
			Date chan string `bson:"date"`
		}
	}
	noteModel.App.Debug = false
	err = instance.Transform(&out)
	noteModel.App.Debug = true
	assert.Error(t, err)
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
	}, systemContext)
	assert.NoError(t, err)
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
	}, systemContext)
	assert.NoError(t, err)
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

	// keep using code directly because we want to test multiple input types for UpdateAttributes()
	createdNote, err := noteModel.Create(wst.M{
		"title": "Old title",
	}, systemContext)
	assert.NoError(t, err)

	updated, err := createdNote.UpdateAttributes(wst.M{
		"title": "Title from wst.M",
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Title from wst.M", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(&wst.M{
		"title": "Title from *wst.M",
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Title from *wst.M", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(map[string]interface{}{
		"title": "Title from map[string]interface{}",
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Title from map[string]interface{}", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(&map[string]interface{}{
		"title": "Title from *map[string]interface{}",
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Title from *map[string]interface{}", updated.GetString("title"))

	secondNote, err := noteModel.Create(wst.M{
		"title": "Second note",
	}, systemContext)
	assert.NoError(t, err)

	updated, err = createdNote.UpdateAttributes(*secondNote, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Second note", updated.GetString("title"))

	_, err = secondNote.UpdateAttributes(wst.M{
		"title": "Second note updated for *Instance",
	}, systemContext)
	assert.NoError(t, err)

	updated, err = createdNote.UpdateAttributes(secondNote, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Second note updated for *Instance", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(struct {
		Title string `bson:"title"`
	}{
		Title: "Title from struct",
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Title from struct", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(&struct {
		Title chan string `bson:"title"`
	}{
		Title: make(chan string),
	}, systemContext)
	assert.Error(t, err)

	updated, err = createdNote.UpdateAttributes("invalid type", systemContext)
	assert.Error(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"title": "Title from wst.M",
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Title from wst.M", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(wst.M{
		"title": "Title from wst.M 2",
	}, &model.EventContext{BaseContext: systemContext})
	assert.NoError(t, err)
	assert.Equal(t, "Title from wst.M 2", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__forceError": true,
	}, systemContext)
	assert.Error(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": wst.M{
			"title": "Overwritten title",
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Overwritten title", updated.GetString("title"))

	_, err = secondNote.UpdateAttributes(wst.M{
		"title": "Second note updated for overwrite with Instance",
	}, systemContext)
	assert.NoError(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": *secondNote,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Second note updated for overwrite with Instance", updated.GetString("title"))

	_, err = secondNote.UpdateAttributes(wst.M{
		"title": "Second note updated for overwrite with *Instance",
	}, systemContext)
	assert.NoError(t, err)

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": secondNote,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "Second note updated for overwrite with *Instance", updated.GetString("title"))

	updated, err = createdNote.UpdateAttributes(wst.M{
		"__overwriteWith": "invalid type",
	}, systemContext)
	assert.Error(t, err)

}

func Test_Instance_GetStringNonExistent(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "", instance.GetString("nonExistent"))
}

func Test_Instance_GetFloat64FromFloat32(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"float32": float32(1.2),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, float32(1.2), float32(instance.GetFloat64("float32")))
}

func Test_Instance_GetFloat64FromInt64(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"int64": int64(1),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), int64(instance.GetFloat64("int64")))
}

func Test_Instance_GetFloat64FromInt32(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"int32": int32(1),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), int32(instance.GetFloat64("int32")))
}

func Test_Instance_GetFloat64FromInt(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"int": 1,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(instance.GetFloat64("int")))
}

func Test_Instance_GetFloat64NonExistent(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, instance.GetFloat64("nonExistent"))
}

func Test_Instance_GetFloatIntFromInt64(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"int64": int64(1),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), instance.GetInt("int64"))
}

func Test_Instance_GetFloatIntFromInt32(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"int32": int32(1),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), int32(instance.GetInt("int32")))
}

func Test_Instance_GetFloatIntFromInt(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"int": 1,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), instance.GetInt("int"))
}

func Test_Instance_GetFloatIntFromFloat64(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"float64": 1.2,
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), instance.GetInt("float64"))
}

func Test_Instance_GetFloatIntFromFloat32(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"float32": float32(1.2),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), instance.GetInt("float32"))
}

func Test_Instance_GetFloatIntNonExistent(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), instance.GetInt("nonExistent"))
}

func Test_Instance_GetBooleanNonExistentDefaultFalse(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.False(t, instance.GetBoolean("nonExistent", false))
}

func Test_Instance_GetBooleanNonExistentDefaultTrue(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.True(t, instance.GetBoolean("nonExistent", true))
}

func Test_Instance_GetObjectIdFromString(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"objectId": noteId.Hex(),
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, noteId.Hex(), instance.GetObjectId("objectId").Hex())
}

func Test_Instance_GetMFromM(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"m": wst.M{
			"key": "value",
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", instance.GetM("m").GetString("key"))
}

func Test_Instance_GetMFromPrimitiveM(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"m": primitive.M{
			"key": "value",
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", instance.GetM("m").GetString("key"))
}

func Test_Instance_GetMFromMapStringInterface(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"m": map[string]interface{}{
			"key": "value",
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", instance.GetM("m").GetString("key"))
}

func Test_Instance_GetMNonExistent(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.Nil(t, instance.GetM("nonExistent"))
}

func Test_Instance_GetMDefaultType(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"m": 1,
	}, systemContext)
	assert.NoError(t, err)
	assert.Nil(t, instance.GetM("m"))
}

func Test_Instance_GetAFromA(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"a": wst.A{
			{"key": "value"},
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", (*instance.GetA("a"))[0].GetString("key"))
}

func Test_Instance_GetAFromPrimitiveA(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"a": primitive.A{
			primitive.M{"key": "value"},
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", (*instance.GetA("a"))[0].GetString("key"))
}

func Test_Instance_GetAFromInterfaceOfMList(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"a": []interface{}{
			wst.M{"key": "value"},
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", (*instance.GetA("a"))[0].GetString("key"))
}

func Test_Instance_GetAFromInterfaceOfPrimitiveMList(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"a": []interface{}{
			primitive.M{"key": "value"},
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", (*instance.GetA("a"))[0].GetString("key"))
}

func Test_Instance_GetAFromMapStringInterfaceList(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"a": []map[string]interface{}{
			{"key": "value"},
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "value", (*instance.GetA("a"))[0].GetString("key"))
}

func Test_Instance_GetANonExistent(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{}, systemContext)
	assert.NoError(t, err)
	assert.Nil(t, instance.GetA("nonExistent"))
}

func Test_Instance_GetADefaultType(t *testing.T) {

	t.Parallel()

	instance, err := noteModel.Build(wst.M{
		"a": 1,
	}, systemContext)
	assert.NoError(t, err)
	assert.Nil(t, instance.GetA("a"))
}

func Test_Instance_AToJSON(t *testing.T) {

	t.Parallel()

	var instanceA model.InstanceA
	singleInstance, err := noteModel.Build(wst.M{
		"_id": noteId,
	}, systemContext)
	assert.NoError(t, err)
	instanceA = append(instanceA, singleInstance)
	json := instanceA.ToJSON()
	assert.Equal(t, 1, len(json))
	assert.Equal(t, noteId.Hex(), json[0].GetString("id"))
}
