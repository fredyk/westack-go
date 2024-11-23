package tests

import (
	"fmt"
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	"github.com/fredyk/westack-go/v2/westack"
	"github.com/golang-jwt/jwt"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/fredyk/westack-go/v2/model"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/v2/common"
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
	}, systemContext)
	assert.Nil(t, err2)
	created, err := noteModel.Create(build, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, "bar", created.GetString("foo"))
}

func Test_CreateWithInstancePointer(t *testing.T) {

	t.Parallel()

	v, err := noteModel.Build(wst.M{
		"foo2": "bar2",
	}, systemContext)
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

	result, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"__forceError": true,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, result, "error")
}

func Test_CreateWithOverrideInvalid(t *testing.T) {

	t.Parallel()

	result, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
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

	result, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
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

func Test_BearerTokenWithObjectId(t *testing.T) {

	t.Parallel()

	createdNote, err := noteModel.Create(wst.M{
		"title":     "Test",
		"accountId": randomAccountToken.GetString("accountId"),
		"appId":     appInstance.Id,
	}, systemContext)
	assert.NoError(t, err)
	subjId, err := primitive.ObjectIDFromHex(randomAccountToken.GetString("accountId"))
	assert.NoError(t, err)
	userBearer := model.CreateBearer(subjId, float64(time.Now().Unix()), float64(60), []string{"USER"})
	// sign the bearer
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userBearer.Claims)
	tokenString, err := token.SignedString(appInstance.Model.App.JwtSecretKey)
	assert.NoError(t, err)
	userBearer.Raw = tokenString
	refreshedByAccount, err := noteModel.FindById(createdNote.GetID(), &wst.Filter{
		Include: &wst.Include{
			{Relation: "account"},
			{Relation: "footer1"},
			{Relation: "footer2"},
			{Relation: "publicFooter"},
			{Relation: "entries"},
		},
	}, &model.EventContext{
		Bearer: userBearer,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Test", refreshedByAccount.GetString("title"))
	updatedByAccount, err := refreshedByAccount.UpdateAttributes(wst.M{
		"title": "Test2",
	}, &model.EventContext{
		Bearer: userBearer,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Test2", updatedByAccount.GetString("title"))

	refreshedByApp, err := noteModel.FindById(createdNote.GetID(), &wst.Filter{
		Include: &wst.Include{
			{Relation: "account"},
			{Relation: "footer1"},
			{Relation: "footer2"},
			{Relation: "publicFooter"},
			{Relation: "entries"},
		},
	}, &model.EventContext{
		Bearer: appBearer,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Test2", refreshedByApp.GetString("title"))
	updatedByApp, err := refreshedByApp.UpdateAttributes(wst.M{
		"title": "Test3",
	}, &model.EventContext{
		Bearer: appBearer,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Test3", updatedByApp.GetString("title"))
}

func Test_CreateWithDefaultStringValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Equal(t, "default", created.GetString("defaultString"))
}

func Test_CreateWithDefaultIntValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.EqualValues(t, 1, created["defaultInt"])
}

func Test_CreateWithDefaultFloatValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.EqualValues(t, 87436874647.8761781676, created["defaultFloat"])
}

func Test_CreateWithDefaultBooleanValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.EqualValues(t, true, created["defaultBoolean"])
}

func Test_CreateWithDefaultListValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultList")
	assert.IsType(t, []interface{}{}, created["defaultList"])
	assert.EqualValues(t, []interface{}{"default"}, created["defaultList"])
}

func Test_CreateWithDefaultMapValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultMap")
	assert.IsType(t, map[string]interface{}{}, created["defaultMap"])
	assert.Contains(t, created["defaultMap"].(map[string]interface{}), "defaultKey")
	assert.Equal(t, map[string]interface{}{"defaultKey": "defaultValue"}, created["defaultMap"].(map[string]interface{}))
}

func Test_CreateWithDefaultNilValue(t *testing.T) {

	t.Parallel()

	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultNull")
	assert.Nil(t, created["defaultNull"])
}

func Test_CreateWithDefaultTimeValue(t *testing.T) {

	t.Parallel()

	probablyTime := time.Now()
	lowerSeconds := probablyTime.Unix()
	// Should be 16 seconds after at most
	upperSeconds := probablyTime.Unix() + 16
	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
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
	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
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
	// Should be 15 seconds after at most
	upperSeconds := probablyTime.Unix() + 3615
	created, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{}, wst.M{"Content-Type": "application/json"})
	assert.Nil(t, err)
	assert.Contains(t, created, "defaultTimeHourFromNow")
	var parsedTime time.Time
	parsedTime, err = time.Parse(time.RFC3339, created["defaultTimeHourFromNow"].(string))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, parsedTime.Unix(), lowerSeconds)
	assert.LessOrEqual(t, parsedTime.Unix(), upperSeconds)

}

// https://github.com/fredyk/westack-go/issues/464
func Test_ProtectedFields(t *testing.T) {

	t.Parallel()

	// Create a random user
	plainUser1 := wst.M{
		"username": fmt.Sprintf("user%v", createRandomInt()),
		"password": "Abcd1234.",
		"phone":    "1234567890",
	}
	user1 := createAccount(t, plainUser1)

	// Another random user
	plainUser2 := wst.M{
		"username": fmt.Sprintf("user%v", createRandomInt()),
		"password": "Abcd1234.",
		"phone":    "9876543210",
	}
	createAccount(t, plainUser2)

	// Account with special privilege "__protectedFieldsPrivileged"
	plainUser3 := wst.M{
		"username": fmt.Sprintf("__protectedFieldsPrivileged_user%v", createRandomInt()),
		"password": "Abcd1234.",
	}

	// Add the privilege "__protectedFieldsPrivileged" to the user3
	plainUserWithPrivileges := westack.AccountWithRoles{
		Username: plainUser3.GetString("username"),
		Password: plainUser3.GetString("password"),
		Roles:    []string{"__protectedFieldsPrivileged"},
	}
	userWithPrivileges, err := westack.UpsertAccountWithRoles(app, plainUserWithPrivileges, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userWithPrivileges.GetID())
	assert.NotEmptyf(t, userWithPrivileges.GetString("id"), "Account should have an id")

	// Login the user1
	username1 := plainUser1.GetString("username")
	password1 := plainUser1.GetString("password")
	user1Token, err := loginAccount(username1, password1, t)
	assert.NoError(t, err)

	// Login the user2
	user2Token, err := loginAccount(plainUser2.GetString("username"), plainUser2.GetString("password"), t)
	assert.NoError(t, err)

	// Login with admin
	adminUsername := os.Getenv("WST_ADMIN_USERNAME")
	adminPwd := os.Getenv("WST_ADMIN_PWD")
	adminToken, err := loginAccount(
		adminUsername,
		adminPwd,
		t,
	)
	assert.NoError(t, err)

	// Login with userWithPrivileges
	userWithPrivilegesToken, err := loginAccount(
		plainUserWithPrivileges.Username,
		plainUserWithPrivileges.Password,
		t,
	)
	assert.NoError(t, err)
	privilegedUserBearer := userWithPrivilegesToken.GetString("id")
	// Extract the payload from the bearer
	privilegedUserPayload := extractJWTPayload(t, privilegedUserBearer)
	assert.Contains(t, privilegedUserPayload.Roles, "__protectedFieldsPrivileged")

	// Get the user 1 through API with the user2
	// Phone should not be returned with user2
	user1RetrievedWithUser2, err := wstfuncs.InvokeApiJsonM("GET", fmt.Sprintf("/public-accounts/%v", user1.GetString("id")), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", user2Token["id"]),
	})
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUser2.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUser2.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUser2, "password", "Password should not be returned")
	assert.NotContainsf(t, user1RetrievedWithUser2, "phone", "Phone should not be returned")

	// Get the user 1 through API with the user2
	// Phone should not be returned with user2
	encodedFilter, err := json.Marshal(wst.M{
		"where": wst.M{
			"_id": user1.GetString("id"),
		},
	})
	users1RetrievedWithUser2, err := wstfuncs.InvokeApiJsonA("GET", fmt.Sprintf("/public-accounts?filter=%v", string(encodedFilter)), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", user2Token["id"]),
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(users1RetrievedWithUser2))
	user1RetrievedWithUser2 = users1RetrievedWithUser2[0]

	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUser2.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUser2.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUser2, "password", "Password should not be returned")
	assert.NotContainsf(t, user1RetrievedWithUser2, "phone", "Phone should not be returned")

	// Now get the user 1 through API with the admin token
	// Phone should be returned with admin
	user1RetrievedWithAdmin, err := wstfuncs.InvokeApiJsonM("GET", fmt.Sprintf("/public-accounts/%v", user1.GetString("id")), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", adminToken["id"]),
	})
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithAdmin.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithAdmin.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithAdmin, "password", "Password should not be returned")
	assert.Containsf(t, user1RetrievedWithAdmin, "phone", "Phone should be returned")

	// Phone should be returned also with user1 because it is the $owner
	user1RetrievedWithUser1, err := wstfuncs.InvokeApiJsonM("GET", fmt.Sprintf("/public-accounts/%v", user1.GetString("id")), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", user1Token["id"]),
	})
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUser1.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUser1.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUser1, "password", "Password should not be returned")
	assert.Containsf(t, user1RetrievedWithUser1, "phone", "Phone should be returned")
	assert.Equalf(t, user1.GetString("phone"), user1RetrievedWithUser1.GetString("phone"), "Phone should be the same")

	// And Phone should be returned also with userWithPrivileges
	user1RetrievedWithUserWithPrivileges, err := wstfuncs.InvokeApiJsonM("GET", fmt.Sprintf("/public-accounts/%v", user1.GetString("id")), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", userWithPrivilegesToken["id"]),
	})
	assert.NoError(t, err)
	assert.Equal(t, user1.GetString("id"), user1RetrievedWithUserWithPrivileges.GetString("id"))
	assert.Equal(t, user1.GetString("username"), user1RetrievedWithUserWithPrivileges.GetString("username"))
	assert.NotContainsf(t, user1RetrievedWithUserWithPrivileges, "password", "Password should not be returned")
	assert.Containsf(t, user1RetrievedWithUserWithPrivileges, "phone", "Phone should be returned")
	assert.Equalf(t, user1.GetString("phone"), user1RetrievedWithUserWithPrivileges.GetString("phone"), "Phone should be the same")

}
