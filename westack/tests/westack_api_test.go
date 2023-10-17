package tests

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"

	wst "github.com/fredyk/westack-go/westack/common"
)

func createNoteForUser(userId string, token string, footerId string, t *testing.T) (note wst.M, err error) {
	parsed, err := invokeApiJsonM(t, "POST", "/notes", wst.M{
		"title":    "Test Note",
		"content":  "This is a test note",
		"userId":   userId,
		"footerId": footerId,
	}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", token),
	})
	assert.NoError(t, err)

	return parsed, err

}

func Test_FindMany(t *testing.T) {

	t.Parallel()

	var err error

	user := createUser(t, wst.M{
		"username": fmt.Sprintf("user-%d", createRandomInt()),
		"password": "abcd1234.",
	})
	token, err := loginUser(user.GetString("username"), "abcd1234.", t)
	assert.Nilf(t, err, "Error while logging in: %v", err)
	assert.NotNilf(t, token, "Token is nil: %v", token)
	assert.Contains(t, token, "id")

	footer, err := createFooter2ForUser(token["id"].(string), user["id"].(string), t)
	assert.Nilf(t, err, "Error while creating footer: %v", err)
	assert.NotNilf(t, footer, "Footer is nil: %v", footer)
	assert.NotEmpty(t, footer["id"].(string))

	note, err := createNoteForUser(user["id"].(string), token["id"].(string), footer["id"].(string), t)
	assert.Nilf(t, err, "Error while creating note: %v", err)
	assert.NotNilf(t, note, "Note is nil: %v", note)
	assert.NotEmpty(t, note["id"].(string))

	parsed, err := invokeApiJsonA(t, "GET", `/notes?filter={"include":[{"relation":"user"},{"relation":"footer1"},{"relation":"footer2"}]}`, nil, nil)
	assert.NoError(t, err)

	assert.Greaterf(t, len(parsed), 0, "parsed: %v\n", parsed)

}

func Test_Count(t *testing.T) {

	// This test is not parallel, because it is counting the number of notes in the database and creating a new note
	// to check if the count is increased by one.
	// If this test is run in parallel, the count will be increased by more than one and the test will fail.
	// t.Parallel()

	// Count notes
	countResponse, err := invokeApiAsRandomUser(t, "GET", "/notes/count", nil, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, countResponse["count"])

	// Create a note
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"title": "Test Note",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.NotEmptyf(t, note.GetString("id"), "Note ID is nil: %v", note)
	assert.Equal(t, "Test Note", note.GetString("title"))

	// Count notes again
	newCount, err := invokeApiAsRandomUser(t, "GET", "/notes/count", nil, nil)
	assert.NoError(t, err)
	assert.EqualValuesf(t, countResponse["count"].(float64)+1, newCount["count"], "Count is not increased: %v", newCount)

}

func Test_FindById(t *testing.T) {

	t.Parallel()

	foundUser, err := invokeApiAsRandomUser(t, "GET", fmt.Sprintf("/users/%v", randomUser.GetString("id")), nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, foundUser, "id")
	assert.Equal(t, randomUser.GetString("id"), foundUser.GetString("id"))

}

func Test_UserFindSelf(t *testing.T) {

	t.Parallel()

	foundUser, err := invokeApiAsRandomUser(t, "GET", "/users/me", nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, foundUser, "id")
	assert.Equal(t, randomUser.GetString("id"), foundUser.GetString("id"))

}

func Test_PostResetPassword(t *testing.T) {

	t.Parallel()

	// Request password reset
	resetPasswordResponse, err := invokeApiJsonM(t, "POST", "/users/reset-password", wst.M{}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, "OK", resetPasswordResponse.GetString("result"))
	assert.Equal(t, "Reset password email sent", resetPasswordResponse.GetString("message"))

}

func Test_VerifyEmail(t *testing.T) {

	t.Parallel()

	// Request email verification
	verifyEmailResponse, err := invokeApiAsRandomUser(t, "POST", "/users/verify-mail", wst.M{}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, "OK", verifyEmailResponse.GetString("result"))
	assert.Equal(t, "Verification email sent", verifyEmailResponse.GetString("message"))

	// Request email verification
	performVerificationResponse := invokeApiFullResponse(t, "GET", fmt.Sprintf("/users/verify-mail?user_id=%s&access_token=%s&redirect_uri=%s",
		randomUser.GetString("id"),
		verifyEmailResponse.GetString("bearer"),
		encodeUriComponent("/api/v1/users/me"),
	), nil, nil)
	assert.NotEmpty(t, performVerificationResponse)
	assert.Equal(t, fiber.StatusFound, performVerificationResponse.StatusCode)
	assert.Equal(t, "/api/v1/users/me", performVerificationResponse.Header.Get("Location"))
	//assert.Equal(t, "OK", performVerificationResponse.GetString("result"))
	//assert.Equal(t, "Email verified", performVerificationResponse.GetString("message"))

}

func Test_GetPerformEmailVerification(t *testing.T) {

	t.Parallel()

}

func createFooter2ForUser(token string, userId string, t *testing.T) (wst.M, error) {
	parsed, err := invokeApiJsonM(t, "POST", "/footers", wst.M{
		"userId": userId,
	}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", token),
	})
	assert.NoError(t, err)

	return parsed, err
}

func Test_EmptyArray(t *testing.T) {

	t.Parallel()

	parsed, err := invokeApiJsonA(t, "GET", "/empties", nil, nil)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(parsed), "parsed: %v", parsed)

}

func loginUser(email string, password string, t *testing.T) (wst.M, error) {
	res, err := loginAsExplicitMode(email, password, "email", t)
	if err != nil {
		return res, err
	}
	if res.GetString("id") == "" {
		// try to login as username
		res, err = loginAsExplicitMode(email, password, "username", t)
		return res, err
	}
	return res, nil
}

func loginAsExplicitMode(email string, password string, mode string, t *testing.T) (wst.M, error) {
	return invokeApiJsonM(t, "POST", "/users/login", wst.M{
		mode:       email,
		"password": password,
	}, wst.M{
		"Content-Type": "application/json",
	})
}

func Test_CreateUserWithoutUsername(t *testing.T) {

	t.Parallel()

	user, err := invokeApiJsonM(t, "POST", "/users", wst.M{
		"password": "abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusBadRequest, user.GetM("error").GetInt("statusCode"))
	assert.Equal(t, "EMAIL_PRESENCE", user.GetM("error").GetString("code"))

}

func Test_LoginUserWithoutUserOrEmail(t *testing.T) {

	t.Parallel()

	user, err := invokeApiJsonM(t, "POST", "/users/login", wst.M{
		"password": "abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusBadRequest, user.GetM("error").GetInt("statusCode"))
	assert.Equal(t, "USERNAME_EMAIL_REQUIRED", user.GetM("error").GetString("code"))

}

func Test_LoginUserWithoutPassword(t *testing.T) {

	t.Parallel()

	user, err := invokeApiJsonM(t, "POST", "/users/login", wst.M{
		"username": fmt.Sprintf("user-%d-doesnotexist", createRandomInt()),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusUnauthorized, user.GetM("error").GetInt("statusCode"))
	assert.Equal(t, "LOGIN_FAILED", user.GetM("error").GetString("code"))

}

func Test_LoginUserWithWrongPassword(t *testing.T) {

	t.Parallel()

	user, err := invokeApiJsonM(t, "POST", "/users/login", wst.M{
		"username": randomUser.GetString("username"),
		"password": "wrongpassword",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusUnauthorized, user.GetM("error").GetInt("statusCode"))
	assert.Equal(t, "LOGIN_FAILED", user.GetM("error").GetString("code"))

}

func Test_RequestCache(t *testing.T) {

	t.Parallel()

	// Create a request cache entry
	requestCacheEntry, err := invokeApiAsRandomUser(t, "POST", "/request-cache-entries", wst.M{
		"_entries": wst.A{
			{
				"key":   "test-key",
				"value": "test-value",
			},
		},
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, requestCacheEntry, "_redId")

	time.Sleep(1 * time.Second)

	// Get all request cache entries
	endpoint := fmt.Sprintf("/request-cache-entries?filter={\"where\":{\"_redId\":\"%s\"}}", requestCacheEntry.GetString("_redId"))
	requestCacheEntries, err := invokeApiJsonA(t, "GET", endpoint, nil, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", randomUserToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Greater(t, len(requestCacheEntries), 0)

}

func Test_EndpointUsingCodecs(t *testing.T) {
	t.Parallel()

	result, err := invokeApiJsonM(t, "GET", "/endpoint-using-codecs", nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.GetString("title"))
	assert.NotEmpty(t, result.GetString("id"))
	assert.NotEqualValues(t, result.GetString("id"), primitive.NilObjectID.Hex())
}

func Test_ForceError1719(t *testing.T) {
	t.Parallel()

	result, err := invokeApiJsonM(t, "GET", "/notes?forceError1719=true", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, result.GetM("error").GetString("code"), "ERR_1719")
}

func Test_FindMe(t *testing.T) {
	t.Parallel()

	result, err := invokeApiAsRandomUser(t, "GET", "/users/me", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, result.GetString("id"), randomUser.GetString("id"))

}
