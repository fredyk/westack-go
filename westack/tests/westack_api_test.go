package tests

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"net/http"
	"testing"

	wst "github.com/fredyk/westack-go/westack/common"
)

func createUserThroughNetwork(t *testing.T) wst.M {
	randUserN := createRandomInt()
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/users", jsonToReader(wst.M{
		"username": fmt.Sprintf("user%v", randUserN),
		"email":    fmt.Sprintf("user.%v@example.com", randUserN),
		"password": "abcd1234.",
	}))
	assert.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.NoError(t, err)

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	assert.Nilf(t, err, "Error: %v, received bytes: %v <--", err, string(out))

	return parsed
}

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

	user := createUserThroughNetwork(t)
	token, err := loginUser(user["email"].(string), "abcd1234.", t)
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
	count, err := noteModel.Count(nil, systemContext)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))

	// Create a note
	note, err := noteModel.Create(wst.M{
		"title": "Test Note",
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.NotEqualValuesf(t, primitive.NilObjectID, note.Id, "Note ID is nil: %v", note.Id)
	assert.Equal(t, "Test Note", note.GetString("title"))

	// Count notes again
	newCount, err := noteModel.Count(nil, systemContext)
	assert.NoError(t, err)
	assert.EqualValuesf(t, count+1, newCount, "Count is not increased: %v", newCount)

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
