package tests

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

var app *westack.WeStack
var randomUser wst.M
var randomUserToken wst.M
var adminUserToken wst.M
var appInstance *model.StatefulInstance
var appBearer *model.BearerToken

// Decode the jwtInfo as JSON
type jwtInfo struct {
	Bearer string `json:"-"`
	// roles is mandatory
	Roles  []string `json:"roles"`
	UserId string   `json:"userId"`
}

func extractJWTPayload(t *testing.T, bearer string) jwtInfo {
	splt := strings.Split(bearer, ".")
	assert.Equal(t, 3, len(splt))
	//payloadSt, err := base64.StdEncoding.DecodeString(splt[1])
	//assert.NoError(t, err)
	payloadSt, err := jwt.DecodeSegment(splt[1])
	assert.NoError(t, err)
	//assert.Equal(t, payloadSt, decoded)

	var p jwtInfo
	err = json.Unmarshal(payloadSt, &p)
	assert.NoError(t, err)
	p.Bearer = bearer
	return p
}

func createRandomInt() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	return 1e9 + int(n.Int64())
}

func createRandomFloat(min float64, max float64) float64 {
	var maxInt int64 = 1 << (52 - 1)
	// First create a random int between 0 and 2**32-1
	n, _ := rand.Int(rand.Reader, big.NewInt(maxInt))
	// Then convert it to a float64 between 0 and 1
	f := float64(n.Int64()) / float64(1<<64-1)
	// Then scale it to the desired range
	return min + f*(max-min)
}

func invokeApiJsonM(t *testing.T, method string, url string, body wst.M, headers wst.M) (result wst.M, err error) {
	result, err = invokeApiTyped[wst.M](t, method, url, body, headers)
	return result, err
}

func invokeApiJsonA(t *testing.T, method string, url string, body wst.M, headers wst.M) (result wst.A, err error) {
	return invokeApiTyped[wst.A](t, method, url, body, headers)
}

func invokeApiTyped[T any](t *testing.T, method string, url string, body wst.M, headers wst.M) (result T, err error) {
	respBody := invokeApiBytes(t, method, url, body, headers)
	var parsedRespBody T
	err = json.Unmarshal(respBody, &parsedRespBody)
	//err = easyjson.Unmarshal(respBody, parsedRespBody)
	assert.NoError(t, err)

	return parsedRespBody, err
}

func invokeApiBytes(t *testing.T, method string, url string, body wst.M, headers wst.M) []byte {
	resp := invokeApiFullResponse(t, method, url, body, headers)
	if resp == nil || resp.Body == nil {
		t.Error("resp or resp.Body is nil")
		return make([]byte, 0)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	return respBody
}

func invokeApiFullResponse(t *testing.T, method string, url string, body wst.M, headers wst.M) *http.Response {
	//origin := ""
	origin := "http://localhost:8019"
	req, err := http.NewRequest(method, fmt.Sprintf("%v/api/v1%s", origin, url), jsonToReader(body))
	assert.NoError(t, err)
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}
	//resp, err := app.Server.Test(req, 3000)
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}
	//resp, err := app.Server.Test(req, 600000)
	client := &http.Client{
		Timeout: 45 * time.Second,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	return resp
}

func invokeApiAsRandomUser(t *testing.T, method string, url string, body wst.M, headers wst.M) (result wst.M, err error) {
	if headers == nil {
		headers = wst.M{}
	}
	if v, ok := headers["Authorization"]; !ok || v == "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %v", randomUserToken.GetString("id"))
	}
	return invokeApiJsonM(t, method, url, body, headers)
}

func jsonToReader(m wst.M) io.Reader {
	out, err := json.Marshal(m)
	fmt.Printf("Ignoring error %v\n", err)
	return bytes.NewReader(out)
}

func reduceByKey(notes wst.A, key string) []string {
	var titles []string
	for _, note := range notes {
		titles = append(titles, note.GetString(key))
	}
	return titles
}
