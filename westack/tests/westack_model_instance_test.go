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

func Test_ToJSON_BelongsToRelation(t *testing.T) {
	instance := noteModel.Build(wst.M{
		"_id":    noteId,
		"userId": userId,
		"user": wst.M{
			"id": userId,
		},
	}, systemContext)
	originalUser := instance.GetOne("user")
	json := instance.ToJSON()
	user := json.GetM("user")
	assert.NotNil(t, user)
	assert.Equal(t, userId.Hex(), user["id"].(primitive.ObjectID).Hex())
	assert.Equal(t, originalUser.GetObjectId("id").Hex(), user["id"].(primitive.ObjectID).Hex())

}

func Test_ToJSON_HasManyRelation(t *testing.T) {
	instance := noteModel.Build(wst.M{
		"_id": noteId,
		"entries": primitive.A{
			wst.M{"date": "2021-01-01", "text": "Entry 1"},
		},
	}, systemContext)
	originalEntries := instance.GetMany("entries")
	json := instance.ToJSON()
	entries := json["entries"]
	assert.NotNil(t, entries)
	assert.Equal(t, 1, len(entries.(wst.A)))
	assert.Equal(t, "2021-01-01", entries.(wst.A)[0]["date"])
	assert.Equal(t, originalEntries[0].GetString("date"), entries.(wst.A)[0]["date"])
}

func Test_Access_Empty_Relation(t *testing.T) {
	instance := noteModel.Build(wst.M{
		"_id": noteId,
	}, systemContext)
	// assert user is nil
	assert.Nil(t, instance.GetOne("user"))
	// assert entries is empty
	assert.Equal(t, 0, len(instance.GetMany("entries")))
}

func Test_Instance_Transform(t *testing.T) {
	instance := noteModel.Build(wst.M{
		"_id":    noteId,
		"userId": userId,
		"user": wst.M{
			"id": userId,
		},
		"entries": primitive.A{
			wst.M{"date": "2021-01-01", "text": "Entry 1"},
		},
	}, systemContext)
	var out struct {
		Id     primitive.ObjectID `bson:"id"`
		UserId primitive.ObjectID `bson:"userId"`
		User   struct {
			Id primitive.ObjectID `bson:"id"`
		}
		Entries []struct {
			Date string `bson:"date"`
			Text string `bson:"text"`
		}
	}
	err := instance.Transform(&out)
	assert.Nil(t, err)
	assert.Equal(t, noteId.Hex(), out.Id.Hex())
	assert.Equal(t, userId.Hex(), out.UserId.Hex())
	assert.Equal(t, userId.Hex(), out.User.Id.Hex())
	assert.Equal(t, 1, len(out.Entries))
	assert.Equal(t, "2021-01-01", out.Entries[0].Date)
	assert.Equal(t, "Entry 1", out.Entries[0].Text)
}

func Test_Instance_Transform_Error(t *testing.T) {
	instance := noteModel.Build(wst.M{
		"entries": primitive.A{
			wst.M{"date": "2021-01-01"},
		},
	}, systemContext)
	var out struct {
		Entries []struct {
			Date chan string `bson:"date"`
		}
	}
	err := instance.Transform(&out)
	assert.NotNil(t, err)
}

func Test_Instance_UncheckedTransform(t *testing.T) {
	// recover the panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in f %v\n", r)
			t.Errorf("The code did panic")
		}
	}()

	instance := noteModel.Build(wst.M{
		"entries": primitive.A{
			wst.M{"date": "2021-01-01"},
		},
	}, systemContext)
	type SafeType struct {
		Entries []struct {
			Date string `bson:"date"`
		}
	}
	out := instance.UncheckedTransform(new(SafeType))
	assert.NotNil(t, out)
}

func Test_Instance_UncheckedTransform_Panic(t *testing.T) {
	// recover the panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	instance := noteModel.Build(wst.M{
		"entries": primitive.A{
			wst.M{"date": "2021-01-01"},
		},
	}, systemContext)
	type UnsafeType struct {
		Entries []struct {
			Date chan string `bson:"date"`
		}
	}
	instance.UncheckedTransform(new(UnsafeType))
}
