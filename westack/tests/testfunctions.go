package tests

import (
	"bytes"
	"fmt"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

// Decode the payload as JSON
type payload struct {
	// roles is mandatory
	Roles []string `json:"roles"`
}

func extractJWTPayload(t *testing.T, bearer string, err error) (payload, error) {
	splt := strings.Split(bearer, ".")
	assert.Equal(t, 3, len(splt))
	//payloadSt, err := base64.StdEncoding.DecodeString(splt[1])
	//assert.Nil(t, err)
	payloadSt, err := jwt.DecodeSegment(splt[1])
	assert.Nil(t, err)
	//assert.Equal(t, payloadSt, decoded)

	var p payload
	err = json.Unmarshal(payloadSt, &p)
	return p, err
}

func createRandomInt() int {
	return 1e9 + rand.Intn(899999999)
}

func invokeApi(t *testing.T, method string, url string, body wst.M, headers wst.M) (result wst.M, err error) {
	req, err := http.NewRequest(method, url, jsonToReader(body))
	assert.Nil(t, err)
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	var parsedRespBody wst.M
	err = json.Unmarshal(respBody, &parsedRespBody)
	assert.Nil(t, err)

	if !assert.GreaterOrEqual(t, resp.StatusCode, 200) || !assert.LessOrEqual(t, resp.StatusCode, 299) {
		return parsedRespBody, fmt.Errorf("unexpected status code: %v. Body: %v", resp.StatusCode, parsedRespBody)
	}
	return parsedRespBody, err
}

func jsonToReader(m wst.M) io.Reader {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(out)
}
