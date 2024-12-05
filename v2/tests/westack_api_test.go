package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
)

func createNoteForAccount(userId string, token string, footerId string, t *testing.T) (note wst.M, err error) {
	parsed, err := wstfuncs.InvokeApiJsonM("POST", "/notes", wst.M{
		"title":     "Test Note",
		"content":   "This is a test note",
		"accountId": userId,
		"footerId":  footerId,
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

	user := createAccount(t, wst.M{
		"username": fmt.Sprintf("user-%d", createRandomInt()),
		"password": "Abcd1234.",
	})
	token, err := loginAccount(user.GetString("username"), "Abcd1234.", t)
	assert.Nilf(t, err, "Error while logging in: %v", err)
	assert.NotNilf(t, token, "Token is nil: %v", token)
	assert.Contains(t, token, "id")

	footer, err := createFooter2ForAccount(token["id"].(string), user["id"].(string), t)
	assert.Nilf(t, err, "Error while creating footer: %v", err)
	assert.NotNilf(t, footer, "Footer is nil: %v", footer)
	assert.NotEmpty(t, footer["id"].(string))

	note, err := createNoteForAccount(user["id"].(string), token["id"].(string), footer["id"].(string), t)
	assert.Nilf(t, err, "Error while creating note: %v", err)
	assert.NotNilf(t, note, "Note is nil: %v", note)
	assert.NotEmpty(t, note["id"].(string))

	parsed, err := wstfuncs.InvokeApiJsonA("GET", `/notes?filter={"include":[{"relation":"account"},{"relation":"footer1"},{"relation":"footer2"}]}`, nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", token["id"].(string)),
	})
	assert.NoError(t, err)

	assert.Greaterf(t, len(parsed), 0, "parsed: %v\n", parsed)

}

func Test_Count(t *testing.T) {

	// This test is not parallel, because it is counting the number of notes in the database and creating a new note
	// to check if the count is increased by one.
	// If this test is run in parallel, the count will be increased by more than one and the test will fail.
	t.Parallel()

	// deleteResult, err := noteModel.DeleteMany(&wst.Where{"_id": wst.M{"$ne": nil}}, &model.EventContext{Bearer: &model.BearerToken{Account: &model.BearerUser{System: true}}})
	// assert.NoError(t, err)
	// assert.NotNil(t, deleteResult)

	// Count notes
	countResponse, err := wstfuncs.InvokeApiTyped[wst.CountResult]("GET", "/notes/count?filter={\"where\":{\"title\":\"Note+for+count\"}}", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 0, countResponse.Count)

	// Create a note
	note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"title": "Note for count",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.NotEmptyf(t, note.GetString("id"), "Note ID is nil: %v", note)
	assert.Equal(t, "Note for count", note.GetString("title"))

	// Count notes again
	newCount, err := wstfuncs.InvokeApiTyped[wst.CountResult]("GET", "/notes/count?filter={\"where\":{\"title\":\"Note+for+count\"}}", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.EqualValuesf(t, countResponse.Count+1, newCount.Count, "Count is not increased: %v", newCount)

}

func Test_FindById(t *testing.T) {

	t.Parallel()

	foundAccount, err := invokeApiAsRandomAccount(t, "GET", fmt.Sprintf("/accounts/%v", randomAccount.GetString("id")), nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, foundAccount, "id")
	assert.Equal(t, randomAccount.GetString("id"), foundAccount.GetString("id"))

}

func Test_AccountFindSelf(t *testing.T) {

	t.Parallel()

	foundAccount, err := invokeApiAsRandomAccount(t, "GET", "/accounts/me", nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, foundAccount, "id")
	assert.Equal(t, randomAccount.GetString("id"), foundAccount.GetString("id"))
	assert.Equal(t, "", foundAccount.GetString("password"))
	assert.Nil(t, foundAccount.GetM("error"))
	assert.Nil(t, foundAccount.GetM("error").GetM("details"))
	assert.Equal(t, "", foundAccount.GetM("error").GetString("code"))

}

func Test_PostResetPassword(t *testing.T) {

	t.Parallel()

	// Request password reset
	resetPasswordResponse, err := wstfuncs.InvokeApiJsonM("POST", "/accounts/reset-password", wst.M{}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, "OK", resetPasswordResponse.GetString("result"))
	assert.Equal(t, "Reset password email sent", resetPasswordResponse.GetString("message"))

}

func Test_VerifyEmail(t *testing.T) {

	t.Parallel()

	// Request email verification
	verifyEmailResponse, err := invokeApiAsRandomAccount(t, "POST", "/accounts/verify-mail", wst.M{}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, "OK", verifyEmailResponse.GetString("result"))
	assert.Equal(t, "Verification email sent", verifyEmailResponse.GetString("message"))

	// Request email verification
	performVerificationResponse, _ := wstfuncs.InvokeApiFullResponse("GET", fmt.Sprintf("/accounts/verify-mail?user_id=%s&access_token=%s&redirect_uri=%s",
		randomAccount.GetString("id"),
		verifyEmailResponse.GetString("bearer"),
		encodeUriComponent("/api/v1/accounts/me"),
	), nil, nil)
	assert.NotEmpty(t, performVerificationResponse)
	assert.Equal(t, fiber.StatusFound, performVerificationResponse.StatusCode)
	assert.Equal(t, "/api/v1/accounts/me", performVerificationResponse.Header.Get("Location"))
	//assert.Equal(t, "OK", performVerificationResponse.GetString("result"))
	//assert.Equal(t, "Email verified", performVerificationResponse.GetString("message"))

}

func Test_GetPerformEmailVerification(t *testing.T) {

	t.Parallel()

}

func createFooter2ForAccount(token string, userId string, t *testing.T) (wst.M, error) {
	parsed, err := wstfuncs.InvokeApiJsonM("POST", "/footers", wst.M{
		"accountId": userId,
	}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", token),
	})
	assert.NoError(t, err)

	return parsed, err
}

func Test_EmptyArray(t *testing.T) {

	t.Parallel()

	parsed, err := wstfuncs.InvokeApiJsonA("GET", "/empties", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)

	assert.Equal(t, 0, len(parsed), "parsed: %v", parsed)

}

func loginAccount(email string, password string, t *testing.T) (wst.M, error) {
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
	return wstfuncs.InvokeApiJsonM("POST", "/accounts/login", wst.M{
		mode:       email,
		"password": password,
	}, wst.M{
		"Content-Type": "application/json",
	})
}

func Test_CreateAccountWithoutUsername(t *testing.T) {

	t.Parallel()

	user, err := wstfuncs.InvokeApiJsonM("POST", "/accounts", wst.M{
		"password": "Abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusBadRequest, user.GetInt("error.statusCode"))
	assert.Equal(t, "EMAIL_PRESENCE", user.GetString("error.code"))

}

func Test_LoginAccountWithoutUserOrEmail(t *testing.T) {

	t.Parallel()

	user, err := wstfuncs.InvokeApiJsonM("POST", "/accounts/login", wst.M{
		"password": "Abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusBadRequest, user.GetInt("error.statusCode"))
	assert.Equal(t, "USERNAME_EMAIL_REQUIRED", user.GetString("error.code"))

}

func Test_LoginAccountWithoutPassword(t *testing.T) {

	t.Parallel()

	user, err := wstfuncs.InvokeApiJsonM("POST", "/accounts/login", wst.M{
		"username": fmt.Sprintf("user-%d-doesnotexist", createRandomInt()),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusUnauthorized, user.GetInt("error.statusCode"))
	assert.Equal(t, "PASSWORD_REQUIRED", user.GetString("error.code"))

}

func Test_LoginAccountWithWrongPassword(t *testing.T) {

	t.Parallel()

	user, err := wstfuncs.InvokeApiJsonM("POST", "/accounts/login", wst.M{
		"username": randomAccount.GetString("username"),
		"password": "wrongpassword",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusUnauthorized, user.GetInt("error.statusCode"))
	assert.Equal(t, "LOGIN_FAILED", user.GetString("error.code"))

}

func Test_RequestCache(t *testing.T) {

	t.Parallel()

	// Create a request cache entry
	requestCacheEntry, err := invokeApiAsRandomAccount(t, "POST", "/request-cache-entries", wst.M{
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
	requestCacheEntries, err := wstfuncs.InvokeApiJsonA("GET", endpoint, nil, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Greater(t, len(requestCacheEntries), 0)

}

func Test_EndpointUsingCodecs(t *testing.T) {
	t.Parallel()

	result, err := wstfuncs.InvokeApiJsonM("GET", "/endpoint-using-codecs", nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.GetString("title"))
	assert.NotEmpty(t, result.GetString("id"))
	assert.NotEqualValues(t, result.GetString("id"), primitive.NilObjectID.Hex())
}

func Test_ForceError1719(t *testing.T) {
	t.Parallel()

	result, err := wstfuncs.InvokeApiJsonM("GET", "/notes?forceError1719=true", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, "ERR_1719", result.GetString("error.code"))
}

func Test_FindMe(t *testing.T) {
	t.Parallel()

	result, err := invokeApiAsRandomAccount(t, "GET", "/accounts/me", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, result.GetString("id"), randomAccount.GetString("id"))

}

func Test_Patch(t *testing.T) {
	t.Parallel()

	result, err := invokeApiAsRandomAccount(t, "PATCH", "/accounts/"+randomAccount.GetString("id"), wst.M{
		"attribute1452": "value1452",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, result.GetString("attribute1452"), "value1452")

}

func Test_PatchWithEphemeral(t *testing.T) {
	t.Parallel()

	result, err := invokeApiAsRandomAccount(t, "PATCH", "/accounts/"+randomAccount.GetString("id"), wst.M{
		"testEphemeral": "ephemeralAttribute1503",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, "ephemeralValue1503", result.GetString("ephemeralAttribute1503"))

	// Find user again and check that the ephemeral attribute is not there
	result, err = invokeApiAsRandomAccount(t, "GET", "/accounts/me", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", result.GetString("ephemeralAttribute1503"))

}

// First random user creates a note
// Then the same user updates the note
func Test_AccountUpdatesNote(t *testing.T) {

	t.Parallel()

	// Create a note
	note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"title":     "Test Note",
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.NotEmptyf(t, note.GetString("id"), "Note ID is nil: %v", note)
	assert.Equal(t, "Test Note", note.GetString("title"))

	// Update the note
	updatedNote, err := invokeApiAsRandomAccount(t, "PATCH", "/notes/"+note.GetString("id"), wst.M{
		"title": "Test Note Updated",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, updatedNote)
	assert.Equal(t, "Test Note Updated", updatedNote.GetString("title"))

	// Now recursive permissions. Create a footer associated to the note, and then update the footer
	footer, err := invokeApiAsRandomAccount(t, "POST", "/footers", wst.M{
		"noteId": note.GetString("id"),
		"text":   "Test Footer",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, footer)
	assert.NotEmptyf(t, footer.GetString("id"), "Footer ID is nil: %v", footer)
	assert.Equal(t, note.GetString("id"), footer.GetString("noteId"))

	// Update the footer
	updatedFooter, err := invokeApiAsRandomAccount(t, "PATCH", "/footers/"+footer.GetString("id"), wst.M{
		"text": "Test Footer Updated",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, updatedFooter)
	assert.Equal(t, "Test Footer Updated", updatedFooter.GetString("text"))

	// Now another user tries to update the same footer
	user2Username := fmt.Sprintf("user-%d", createRandomInt())
	createAccount(t, wst.M{
		"username": user2Username,
		"password": "Abcd1234.",
	})

	user2Token, err := loginAccount(user2Username, "Abcd1234.", t)
	assert.NoError(t, err)
	assert.NotEmpty(t, user2Token.GetString("id"))

	// Update the footer
	updatedFooter, err = wstfuncs.InvokeApiJsonM("PATCH", "/footers/"+footer.GetString("id"), wst.M{
		"text": "Test Footer Updated 2",
	}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", user2Token.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, updatedFooter.GetInt("error.statusCode"))

	// Also check that user2 cannot modify the note
	updatedNote, err = wstfuncs.InvokeApiJsonM("PATCH", "/notes/"+note.GetString("id"), wst.M{
		"title": "Test Note Updated 2",
	}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", user2Token.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, updatedNote.GetInt("error.statusCode"))

}

func Test_CreateAccountTwiceByUsername(t *testing.T) {

	t.Parallel()

	user := createAccount(t, wst.M{
		"username": fmt.Sprintf("user-%d", createRandomInt()),
		"password": "Abcd1234.",
	})
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.GetString("id"))

	user2, err := wstfuncs.InvokeApiJsonM("POST", "/accounts", wst.M{
		"username": user.GetString("username"),
		"password": "Abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, user2)
	assert.Equal(t, "USERNAME_UNIQUENESS", user2.GetString("error.code"))

}

func Test_CreateAccountTwiceByEmail(t *testing.T) {

	t.Parallel()

	user := createAccount(t, wst.M{
		"email":    fmt.Sprintf("user-%d@example.com", createRandomInt()),
		"password": "Abcd1234.",
	})
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.GetString("id"))

	user2, err := wstfuncs.InvokeApiJsonM("POST", "/accounts", wst.M{
		"email":    user.GetString("email"),
		"password": "Abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, user2)
	assert.Equal(t, "EMAIL_UNIQUENESS", user2.GetString("error.code"))

}

func Test_CreateAccountInvalidEmail(t *testing.T) {

	t.Parallel()

	user, err := wstfuncs.InvokeApiJsonM("POST", "/accounts", wst.M{
		"email":    "invalidEmail",
		"password": "Abcd1234.",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "EMAIL_FORMAT", user.GetString("error.code"))

}

func Test_CreateAccountPasswordBlank(t *testing.T) {

	t.Parallel()

	user, err := wstfuncs.InvokeApiJsonM("POST", "/accounts", wst.M{
		"username": fmt.Sprintf("user-%d", createRandomInt()),
		"password": "",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "PASSWORD_BLANK", user.GetString("error.code"))

}

// First creates a user
// Then the user changes the password
func Test_UpdateAccountPassword(t *testing.T) {

	t.Parallel()

	user := createAccount(t, wst.M{
		"username": fmt.Sprintf("user-%d", createRandomInt()),
		"password": "Abcd1234.",
	})
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.GetString("id"))

	token, err := loginAccount(user.GetString("username"), "Abcd1234.", t)
	assert.NoError(t, err)
	assert.Contains(t, token, "id")

	user2, err := wstfuncs.InvokeApiJsonM("PATCH", "/accounts/"+user.GetString("id"), wst.M{
		"password": "efgh5678,",
	}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", token.GetString("id")),
	})
	assert.NoError(t, err)
	assert.NotNil(t, user2)
	assert.Equal(t, user.GetString("id"), user2.GetString("id"))

	token2, err := loginAccount(user.GetString("username"), "efgh5678,", t)
	assert.NoError(t, err)
	assert.Contains(t, token2, "id")

	// Find self
	user3, err := wstfuncs.InvokeApiJsonM("GET", "/accounts/me", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", token2.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, user.GetString("id"), user3.GetString("id"))

}

// Tries to delete note with id 000000000000000000000000
func Test_DeleteNonExistentNote(t *testing.T) {

	t.Parallel()

	// Delete note
	result, err := wstfuncs.InvokeApiJsonM("DELETE", "/notes/000000000000000000000000", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", adminAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, result.GetInt("error.statusCode"))

}

// First creates a note
// Then the user deletes the note twice. First time it should succeed, second time it should fail.

func Test_DeleteNoteTwice(t *testing.T) {

	t.Parallel()

	// Create a note
	note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"title":     "Test Note",
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.NotEmptyf(t, note.GetString("id"), "Note ID is nil: %v", note)
	assert.Equal(t, "Test Note", note.GetString("title"))

	// Delete note
	result, _ := wstfuncs.InvokeApiTyped[wst.DeleteResult]("DELETE", "/notes/"+note.GetString("id"), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
	assert.EqualValues(t, 1, result.DeletedCount)

	// Delete note again
	result2, err := wstfuncs.InvokeApiJsonM("DELETE", "/notes/"+note.GetString("id"), nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, result2.GetInt("error.statusCode"))
	assert.Equal(t, fmt.Sprintf(`Deleted 0 instances for ObjectID("%v")`, note.GetString("id")), result2.GetString("error.details.message"))

}

func Test_FindWithNestedRelations(t *testing.T) {

	//t.Parallel()

	// Create a note
	note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"title":     "Test Note",
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.NotEmptyf(t, note.GetString("id"), "Note ID is nil: %v", note)
	assert.Equal(t, "Test Note", note.GetString("title"))

	// Create a footer
	footer, err := invokeApiAsRandomAccount(t, "POST", "/footers", wst.M{
		"noteId": note.GetString("id"),
		"text":   "Test Footer",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotNil(t, footer)
	assert.NotEmptyf(t, footer.GetString("id"), "Footer ID is nil: %v", footer)
	assert.Equal(t, note.GetString("id"), footer.GetString("noteId"))

	// Find note with nested relations
	note2, err := wstfuncs.InvokeApiJsonM("GET", "/notes/"+note.GetString("id")+`?filter={"include":[{"relation":"footer1","scope":{"include":[{"relation":"note"}]}}]}`, nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, note.GetString("id"), note2.GetString("id"))
	assert.Equal(t, footer.GetString("id"), note2.GetM("footer1").GetString("id"))
	assert.Equal(t, note.GetString("id"), note2.GetM("footer1").GetM("note").GetString("id"))

}

func Test_NoteWith2Footers(t *testing.T) {

	// t.Parallel()

	// Create a note
	note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"title":     "Note with 2 footers",
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, note.GetString("id"))

	// Create first footer
	footer1, err := invokeApiAsRandomAccount(t, "POST", "/footers", wst.M{
		"text":      "Footer 1",
		"noteId":    note.GetString("id"),
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, footer1.GetString("id"))

	// Create second footer
	footer2, err := invokeApiAsRandomAccount(t, "POST", "/footers?data-debug-key=footer1202", wst.M{
		"text":      "Footer 2",
		"noteId":    note.GetString("id"),
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, footer2.GetInt("error.statusCode"))
	assert.Equal(t, "UNIQUENESS", footer2.GetString("error.code"))

	// Find note with nested relations
	note2, err := invokeApiAsRandomAccount(t, "GET", fmt.Sprintf("/notes/%s?filter={\"include\":[{\"relation\":\"footer1\"}]}", note.GetString("id")), nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, note.GetString("id"), note2.GetString("id"))
	assert.Equal(t, footer1.GetString("id"), note2.GetString("footer1.id"))
	assert.Equal(t, footer1.GetString("text"), note2.GetString("footer1.text"))

}

func Test_ImageWithTwoThumbnails(t *testing.T) {

	t.Parallel()

	// Create a image
	image, err := invokeApiAsRandomAccount(t, "POST", "/images", wst.M{
		"title":     "Image with 2 thuzmbnails",
		"accountId": randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, image.GetString("id"))

	// Create first Thumbnail
	thumbnail1, err := invokeApiAsRandomAccount(t, "POST", "/images", wst.M{
		"text":            "Thumbnail 1",
		"originalImageId": image.GetString("id"),
		"accountId":       randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, thumbnail1.GetString("id"))

	// Create second Thumbnail
	thumbnail2, err := invokeApiAsRandomAccount(t, "POST", "/images", wst.M{
		"text":            "Thumbnail 2",
		"originalImageId": image.GetString("id"),
		"accountId":       randomAccount.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, thumbnail2.GetInt("error.statusCode"))
	assert.Equal(t, "UNIQUENESS", thumbnail2.GetString("error.code"))

	// Find note with nested relations
	image2, err := invokeApiAsRandomAccount(t, "GET", fmt.Sprintf("/images/%s?filter={\"include\":[{\"relation\":\"thumbnail\"}]}", image.GetString("id")), nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, image.GetString("id"), image2.GetString("id"))
	assert.Equal(t, thumbnail1.GetString("id"), image2.GetString("thumbnail.id"))
	assert.Equal(t, thumbnail1.GetString("text"), image2.GetString("thumbnail.text"))

	invalidThumbnails, err := wstfuncs.InvokeApiJsonA("GET", "/images?filter={\"where\":{\"text\":\"Thumbnail 2\"}}", nil, wst.M{"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id"))})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(invalidThumbnails))
}
