package tests

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"

	wst "github.com/fredyk/westack-go/westack/common"
)

func createNoteForUser(userId string, token string, footerId string, t *testing.T) (note wst.M, err error) {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/notes", jsonToReader(wst.M{
		"title":    "Test Note",
		"content":  "This is a test note",
		"userId":   userId,
		"footerId": footerId,
	}))
	assert.NoError(t, err)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.NoError(t, err)

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
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

	request, err := http.NewRequest("GET", `http://localhost:8019/api/v1/notes?filter={"include":[{"relation":"user"},{"relation":"footer1"},{"relation":"footer2"}]}`, nil)
	assert.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	var out []byte
	var parsed wst.A
	//out = make([]byte, 1)
	//n, err := io.ReadFull(response.Body, out)
	//assert.NoError(t, err)
	//assert.Equal(t, 1, n)
	//assert.Equal(t, "[", string(out))
	//
	//var decoder *json.Decoder
	//decoder = json.NewDecoder(response.Body)
	//decoder.DisallowUnknownFields()
	//decoder.UseNumber()
	//
	//for {
	//	var parsedItem wst.M
	//	err = decoder.Decode(&parsedItem)
	//	if err != nil {
	//		if err == io.EOF {
	//			break
	//		}
	//
	//		buffered := decoder.Buffered()
	//		if buffered != nil {
	//			bufOut := make([]byte, 1)
	//			n, err := io.ReadFull(buffered, bufOut)
	//			assert.NoError(t, err)
	//			assert.Equal(t, 1, n)
	//			if string(bufOut) == "]" {
	//				break
	//			}
	//		}
	//
	//		t.Errorf("Error: %v", err)
	//		time.Sleep(5 * time.Minute)
	//		break
	//	}
	//	fmt.Printf("parsedItem: %v\n", parsedItem)
	//	parsed = append(parsed, parsedItem)
	//}

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.NoError(t, err)

	assert.Greaterf(t, len(out), 0, "Received bytes <-- %v, %v\n", len(out), string(out))

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		assert.Nilf(t, err, "Error: %v, received bytes: %v <--", err, string(out))
	}

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
		encodeUriComponent("http://localhost:8019/api/v1/users/me"),
	), nil, nil, err)
	assert.NotEmpty(t, performVerificationResponse)
	assert.Equal(t, fiber.StatusFound, performVerificationResponse.StatusCode)
	assert.Equal(t, "http://localhost:8019/api/v1/users/me", performVerificationResponse.Header.Get("Location"))
	//assert.Equal(t, "OK", performVerificationResponse.GetString("result"))
	//assert.Equal(t, "Email verified", performVerificationResponse.GetString("message"))

}

func Test_GetPerformEmailVerification(t *testing.T) {

	t.Parallel()

}

func createFooter2ForUser(token string, userId string, t *testing.T) (wst.M, error) {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/footers", jsonToReader(wst.M{
		"userId": userId,
	}))
	assert.NoError(t, err)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.NoError(t, err)

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	assert.NoError(t, err)

	return parsed, err
}

func Test_EmptyArray(t *testing.T) {

	t.Parallel()

	request, err := http.NewRequest("GET", "http://localhost:8019/api/v1/empties", nil)
	assert.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	assert.Equal(t, 200, response.StatusCode)

	var out []byte
	var parsed wst.A

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(out), "Received bytes <-- %v, %v\n", len(out), string(out))
	assert.Equal(t, "[]", string(out))

	err = json.Unmarshal(out, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(parsed), "parsed: %v", parsed)

}

func loginUser(email string, password string, t *testing.T) (wst.M, error) {
	res, err := loginAsUsernameOrEmail(email, password, "email", t)
	if err != nil {
		// try to login as username
		res, err = loginAsUsernameOrEmail(email, password, "username", t)
		if err != nil {
			return res, err
		}
		return res, nil
	}
	return res, nil
}

func loginAsUsernameOrEmail(email string, password string, mode string, t *testing.T) (wst.M, error) {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/users/login", jsonToReader(wst.M{
		mode:       email,
		"password": password,
	}))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	var out []byte
	var parsed wst.M

	if response.StatusCode != 200 {
		// read response body bytes
		out, err = io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Received %d %s bytes <-- %v, %v\n", response.StatusCode, response.Status, len(out), string(out))
	}

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		return nil, err
	}

	return parsed, nil
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
