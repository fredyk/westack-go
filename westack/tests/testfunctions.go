package tests

import (
	"bytes"
	"crypto/rand"
	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
)

var app *westack.WeStack

// Decode the payload as JSON
type payload struct {
	// roles is mandatory
	Roles []string `json:"roles"`
}

func extractJWTPayload(t *testing.T, bearer string, err error) (payload, error) {
	splt := strings.Split(bearer, ".")
	assert.Equal(t, 3, len(splt))
	//payloadSt, err := base64.StdEncoding.DecodeString(splt[1])
	//assert.NoError(t, err)
	payloadSt, err := jwt.DecodeSegment(splt[1])
	assert.NoError(t, err)
	//assert.Equal(t, payloadSt, decoded)

	var p payload
	err = json.Unmarshal(payloadSt, &p)
	return p, err
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

func invokeApi(t *testing.T, method string, url string, body wst.M, headers wst.M) (result wst.M, err error) {
	req, err := http.NewRequest(method, url, jsonToReader(body))
	assert.NoError(t, err)
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}
	resp, err := app.Server.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	var parsedRespBody wst.M
	err = json.Unmarshal(respBody, &parsedRespBody)
	assert.NoError(t, err)

	return parsedRespBody, err
}

func jsonToReader(m wst.M) io.Reader {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(out)
}

func reduceByKey(notes wst.A, key string) []string {
	var titles []string
	for _, note := range notes {
		titles = append(titles, note.GetString(key))
	}
	return titles
}
