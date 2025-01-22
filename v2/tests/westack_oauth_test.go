package tests

import (
	"fmt"
	"testing"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

func Test_OauthAuthorize(t *testing.T) {

	t.Parallel()

	t.Run("Test_OauthAuthorizeTwice", func(t *testing.T) {
		// Run twice, to cover the case when the user was already created
		doTestOauth(t)
		doTestOauth(t)
	})

}

func doTestOauth(t *testing.T) {
	res, err := wstfuncs.InvokeApiFullResponse("GET", "/accounts/oauth/westack", nil, nil, wstfuncs.RequestOptions{FollowRedirects: true})
	assert.NoError(t, err)

	assert.Equal(t, 404, res.StatusCode)

	defer res.Body.Close()

	assert.Contains(t, res.Request.URL.String(), "/dashboard/oauth/success?access_token=")

	var accessToken string
	queryParams := res.Request.URL.Query()
	accessToken = queryParams.Get("access_token")
	assert.NotEmpty(t, accessToken)

	// decode it to get the user info
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(app.Options.JwtSecretKey), nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, token)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.NotEmpty(t, claims["accountId"])
	assert.NotEmpty(t, claims["roles"])
	assert.Equal(t, "USER", claims["roles"].([]interface{})[0])
}
