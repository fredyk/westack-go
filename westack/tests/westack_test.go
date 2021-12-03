package tests

import (
	"bytes"
	"encoding/json"
	"github.com/fredyk/westack-go/westack"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

var app = westack.New(westack.WeStackOptions{
	Debug:       true,
	RestApiRoot: "/api/v1",
	Port:        8023,
})

func init() {
	app.Boot(func(app *westack.WeStack) {

	})
}

func Test_WeStack(t *testing.T) {

	m, b := map[string]interface{}{"email": "email1@example.com", "password": "test"}, new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(m)
	if err != nil {
		t.Error(err)
	}
	response, err := app.Server.Test(httptest.NewRequest("POST", "/api/v1/users", b))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 200, response.StatusCode)

}
