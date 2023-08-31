package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fredyk/westack-go/westack"
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

func Test_CreateWithDefaultStringValue(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, "default", created.GetString("defaultString"))
}

func Test_CreateWithDefaultIntValue(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), created.GetInt("defaultInt"))
}

func Test_CreateWithDefaultFloatValue(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, 87436874647.8761781676, created.GetFloat64("defaultFloat"))
}

func Test_CreateWithDefaultBooleanValue(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Equal(t, true, created.GetBoolean("defaultBoolean", false))
}

func Test_CreateWithDefaultListValue(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Contains(t, created.ToJSON(), "defaultList")
	assert.IsType(t, primitive.A{}, created.ToJSON()["defaultList"])
	assert.Equal(t, primitive.A{"default"}, created.ToJSON()["defaultList"].(primitive.A))
}

func Test_CreateWithDefaultMapValue(t *testing.T) {

	t.Parallel()

	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Contains(t, created.ToJSON(), "defaultMap")
	assert.IsType(t, wst.M{}, created.ToJSON()["defaultMap"])
	assert.Contains(t, created.ToJSON()["defaultMap"].(wst.M), "defaultKey")
	assert.Equal(t, wst.M{"defaultKey": "defaultValue"}, created.ToJSON()["defaultMap"].(wst.M))
}

func Test_CreateWithDefaultTimeValue(t *testing.T) {

	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix()
	// Should be 15 milliseconds after at most
	upperSeconds := probablyTime.Unix() + 3
	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Contains(t, created.ToJSON(), "defaultTimeNow")
	assert.IsType(t, primitive.DateTime(0), created.ToJSON()["defaultTimeNow"])
	assert.GreaterOrEqual(t, created.ToJSON()["defaultTimeNow"].(primitive.DateTime).Time().Unix(), lowerSeconds)
	assert.LessOrEqual(t, created.ToJSON()["defaultTimeNow"].(primitive.DateTime).Time().Unix(), upperSeconds)

}

func Test_CreateWithDefaultTimeHourAgo(t *testing.T) {
	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix() - 3600
	// Should be 15 milliseconds after at most
	upperSeconds := probablyTime.Unix() - 3597
	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Contains(t, created.ToJSON(), "defaultTimeHourAgo")
	assert.IsType(t, primitive.DateTime(0), created.ToJSON()["defaultTimeHourAgo"])
	assert.GreaterOrEqual(t, created.ToJSON()["defaultTimeHourAgo"].(primitive.DateTime).Time().Unix(), lowerSeconds)
	assert.LessOrEqual(t, created.ToJSON()["defaultTimeHourAgo"].(primitive.DateTime).Time().Unix(), upperSeconds)

}

func Test_CreateWithDefaultTimeHourFromNow(t *testing.T) {

	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix() + 3600
	// Should be 15 milliseconds after at most
	upperSeconds := probablyTime.Unix() + 3603
	created, err := noteModel.Create(wst.M{}, systemContext)
	assert.Nil(t, err)
	assert.Contains(t, created.ToJSON(), "defaultTimeHourFromNow")
	assert.IsType(t, primitive.DateTime(0), created.ToJSON()["defaultTimeHourFromNow"])
	assert.GreaterOrEqual(t, created.ToJSON()["defaultTimeHourFromNow"].(primitive.DateTime).Time().Unix(), lowerSeconds)
	assert.LessOrEqual(t, created.ToJSON()["defaultTimeHourFromNow"].(primitive.DateTime).Time().Unix(), upperSeconds)

}

// https://github.com/fredyk/westack-go/issues/464
func Test_ProtectedFields(t *testing.T) {

	t.Parallel()

	// Create a random user
	plainUser1 := wst.M{
		"username": fmt.Sprintf("user%v", createRandomInt()),
		"password": "abcd1234.",
		"phone":    "1234567890",
	}
	user1, err := createUser(t, plainUser1)
	assert.NoError(t, err)

	// Another random user
	plainUser2 := wst.M{
		"username": fmt.Sprintf("user%v", createRandomInt()),
		"password": "abcd1234.",
		"phone":    "9876543210",
	}
	_, err = createUser(t, plainUser2)
	assert.NoError(t, err)

	// User with special privilege "__protectedFieldsPrivileged"
	plainUser3 := wst.M{
		"username": fmt.Sprintf("__protectedFieldsPrivileged_user%v", createRandomInt()),
		"password": "abcd1234.",
	}

	// Add the privilege "__protectedFieldsPrivileged" to the user3
	plainUserWithPrivileges := westack.UserWithRoles{
		Username: plainUser3.GetString("username"),
		Password: plainUser3.GetString("password"),
		Roles:    []string{"__protectedFieldsPrivileged"},
	}
	userWithPrivileges, err := westack.UpsertUserWithRoles(app, plainUserWithPrivileges, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userWithPrivileges.Id)
	assert.NotEmptyf(t, userWithPrivileges.GetString("id"), "User should have an id")

	// Login the user1
	username1 := plainUser1.GetString("username")
	password1 := plainUser1.GetString("password")
	user1Token, err := loginUser(username1, password1, t)
	assert.NoError(t, err)

	// Login the user2
	user2Token, err := loginUser(plainUser2.GetString("username"), plainUser2.GetString("password"), t)
	assert.NoError(t, err)

	// Login with admin
	adminUsername := os.Getenv("WST_ADMIN_USERNAME")
	adminPwd := os.Getenv("WST_ADMIN_PWD")
	adminToken, err := loginUser(
		adminUsername,
		adminPwd,
		t,
	)
	assert.NoError(t, err)

	// Login with userWithPrivileges
	userWithPrivilegesToken, err := loginUser(
		plainUserWithPrivileges.Username,
		plainUserWithPrivileges.Password,
		t,
	)
	assert.NoError(t, err)
	privilegedUserBearer := userWithPrivilegesToken.GetString("id")
	// Extract the payload from the bearer
	privilegedUserPayload, err := extractJWTPayload(t, privilegedUserBearer, err)
	assert.NoError(t, err)
	assert.Contains(t, privilegedUserPayload.Roles, "__protectedFieldsPrivileged")

	// Get the user 1 through API with the user2
	// Phone should not be returned with user2
	user1RetrievedWithUser2, err := invokeApi(
		t,
		"GET",
		fmt.Sprintf("/api/v1/public-users/%v", user1.GetString("id")),
		nil,
		wst.M{
			"Authorization": fmt.Sprintf("Bearer %v", user2Token["id"]),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUser2.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUser2.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUser2, "password", "Password should not be returned")
	assert.NotContainsf(t, user1RetrievedWithUser2, "phone", "Phone should not be returned")

	// Now get the user 1 through API with the admin token
	// Phone should be returned with admin
	user1RetrievedWithAdmin, err := invokeApi(
		t,
		"GET",
		fmt.Sprintf("/api/v1/public-users/%v", user1.GetString("id")),
		nil,
		wst.M{
			"Authorization": fmt.Sprintf("Bearer %v", adminToken["id"]),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithAdmin.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithAdmin.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithAdmin, "password", "Password should not be returned")
	assert.Containsf(t, user1RetrievedWithAdmin, "phone", "Phone should be returned")

	// Phone should be returned also with user1 because it is the $owner
	user1RetrievedWithUser1, err := invokeApi(
		t,
		"GET",
		fmt.Sprintf("/api/v1/public-users/%v", user1.GetString("id")),
		nil,
		wst.M{
			"Authorization": fmt.Sprintf("Bearer %v", user1Token["id"]),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUser1.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUser1.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUser1, "password", "Password should not be returned")
	assert.Containsf(t, user1RetrievedWithUser1, "phone", "Phone should be returned")
	assert.Equalf(t, user1.GetString("phone"), user1RetrievedWithUser1.GetString("phone"), "Phone should be the same")

	// And Phone should be returned also with userWithPrivileges
	user1RetrievedWithUserWithPrivileges, err := invokeApi(
		t,
		"GET",
		fmt.Sprintf("/api/v1/public-users/%v", user1.GetString("id")),
		nil,
		wst.M{
			"Authorization": fmt.Sprintf("Bearer %v", userWithPrivilegesToken["id"]),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUserWithPrivileges.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUserWithPrivileges.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUserWithPrivileges, "password", "Password should not be returned")
	assert.Containsf(t, user1RetrievedWithUserWithPrivileges, "phone", "Phone should be returned")
	assert.Equalf(t, user1.GetString("phone"), user1RetrievedWithUserWithPrivileges.GetString("phone"), "Phone should be the same")

}
