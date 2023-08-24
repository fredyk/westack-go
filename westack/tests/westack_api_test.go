package tests

import (
	"bytes"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"math/rand"
	"net/http"
	"testing"

	wst "github.com/fredyk/westack-go/westack/common"
)

func createUserThroughNetwork(t *testing.T) wst.M {
	randUserN := 100000000 + rand.Intn(899999999)
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/users", jsonToReader(wst.M{
		"username": fmt.Sprintf("user%v", randUserN),
		"email":    fmt.Sprintf("user.%v@example.com", randUserN),
		"password": "abcd1234.",
	}))
	assert.Nil(t, err)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.Nil(t, err)

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	assert.Nilf(t, err, "Error: %v, received bytes: %v <--", err, string(out))

	return parsed
}

func jsonToReader(m wst.M) io.Reader {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(out)
}

func createNoteForUser(userId string, token string, footerId string, t *testing.T) (note wst.M, err error) {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/notes", jsonToReader(wst.M{
		"title":    "Test Note",
		"content":  "This is a test note",
		"userId":   userId,
		"footerId": footerId,
	}))
	assert.Nil(t, err)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.Nil(t, err)

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	assert.Nil(t, err)

	return parsed, err

}

func Test_FindMany(t *testing.T) {

	t.Parallel()

	var err error

	user := createUserThroughNetwork(t)
	token := loginUser(user["email"].(string), "abcd1234.", t)

	footer, err := createFooter2ForUser(token["id"].(string), user["id"].(string), t)
	assert.Nilf(t, err, "Error while creating footer: %v", err)
	assert.NotNilf(t, footer, "Footer is nil: %v", footer)
	assert.NotEmpty(t, footer["id"].(string))

	note, err := createNoteForUser(user["id"].(string), token["id"].(string), footer["id"].(string), t)
	assert.Nilf(t, err, "Error while creating note: %v", err)
	assert.NotNilf(t, note, "Note is nil: %v", note)
	assert.NotEmpty(t, note["id"].(string))

	request, err := http.NewRequest("GET", `http://localhost:8019/api/v1/notes?filter={"include":[{"relation":"user"},{"relation":"footer1"},{"relation":"footer2"}]}`, nil)
	assert.Nil(t, err)

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Equal(t, 200, response.StatusCode)

	var out []byte
	var parsed wst.A
	//out = make([]byte, 1)
	//n, err := io.ReadFull(response.Body, out)
	//assert.Nil(t, err)
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
	//			assert.Nil(t, err)
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
	assert.Nil(t, err)

	assert.Greaterf(t, len(out), 0, "Received bytes <-- %v, %v\n", len(out), string(out))

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		assert.Nilf(t, err, "Error: %v, received bytes: %v <--", err, string(out))
	}

	assert.Greaterf(t, len(parsed), 0, "parsed: %v\n", parsed)

}

func Test_Count(t *testing.T) {

	t.Parallel()

	// Count notes
	count, err := noteModel.Count(nil, systemContext)
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, count, int64(0))

	// Create a note
	note, err := noteModel.Create(wst.M{
		"title": "Test Note",
	}, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, note)
	assert.NotEqualValuesf(t, primitive.NilObjectID, note.Id, "Note ID is nil: %v", note.Id)
	assert.Equal(t, "Test Note", note.GetString("title"))

	// Count notes again
	newCount, err := noteModel.Count(nil, systemContext)
	assert.Nil(t, err)
	assert.EqualValuesf(t, count+1, newCount, "Count is not increased: %v", newCount)

}

func createFooter2ForUser(token string, userId string, t *testing.T) (wst.M, error) {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/footers", jsonToReader(wst.M{
		"userId": userId,
	}))
	assert.Nil(t, err)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.Nil(t, err)

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	assert.Nil(t, err)

	return parsed, err
}

func Test_EmptyArray(t *testing.T) {

	t.Parallel()

	request, err := http.NewRequest("GET", "http://localhost:8019/api/v1/empties", nil)
	assert.Nil(t, err)

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)

	assert.Equal(t, 200, response.StatusCode)

	var out []byte
	var parsed wst.A

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(out), "Received bytes <-- %v, %v\n", len(out), string(out))
	assert.Equal(t, "[]", string(out))

	err = json.Unmarshal(out, &parsed)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(parsed), "parsed: %v", parsed)

}

func loginUser(email string, password string, t *testing.T) wst.M {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/users/login", jsonToReader(wst.M{
		"email":    email,
		"password": password,
	}))
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	var out []byte
	var parsed wst.M

	// read response body bytes
	out, err = io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	return parsed
}
