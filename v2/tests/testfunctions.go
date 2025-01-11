package tests

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/v2/westack"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

var app *westack.WeStack
var randomAccount wst.M
var randomAccountToken wst.M
var adminAccountToken wst.M
var appInstance *model.StatefulInstance
var appBearer *model.BearerToken

// Decode the jwtInfo as JSON
type jwtInfo struct {
	Bearer string `json:"-"`
	// roles is mandatory
	Roles     []string `json:"roles"`
	AccountId string   `json:"accountId"`
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

func invokeApiAsRandomAccount(method, url string, body, headers wst.M) (result wst.M, err error) {
	if headers == nil {
		headers = wst.M{}
	}
	if v, ok := headers["Authorization"]; !ok || v == "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id"))
	}
	return wstfuncs.InvokeApiJsonM(method, url, body, headers)
}

func reduceByKey(notes wst.A, key string) []string {
	var titles []string
	for _, note := range notes {
		titles = append(titles, note.GetString(key))
	}
	return titles
}
