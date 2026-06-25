package tests

import (
	"fmt"
	"testing"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/westack"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Test_ApiKey_CreateAndRead(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	name := fmt.Sprintf("test-key-%v", randN)
	key, err := westack.CreateApiKey(app, name, []string{"apikey:read"}, nil, systemContext)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	m, err := app.FindModel("ApiKey")
	assert.NoError(t, err)

	inst, err := m.FindOne(&wst.Filter{Where: &wst.Where{"key": key}}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, inst)

	assert.True(t, inst.GetBoolean("enabled", false))
	roles, ok := inst.ToJSON()["roles"].(primitive.A)
	assert.True(t, ok)
	assert.Equal(t, 1, len(roles))
	assert.Equal(t, "apikey:read", roles[0])
}

func Test_ApiKey_SetRoles(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	name := fmt.Sprintf("test-key-%v", randN)
	key, err := westack.CreateApiKey(app, name, []string{"apikey:read"}, nil, systemContext)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	err = westack.SetApiKeyRoles(app, key, []string{"apikey:read", "apikey:write"}, systemContext)
	assert.NoError(t, err)

	m, err := app.FindModel("ApiKey")
	assert.NoError(t, err)

	inst, err := m.FindOne(&wst.Filter{Where: &wst.Where{"key": key}}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, inst)

	roles, ok := inst.ToJSON()["roles"].(primitive.A)
	assert.True(t, ok)
	assert.Equal(t, 2, len(roles))
}

func Test_ApiKey_Revoke(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	name := fmt.Sprintf("test-key-%v", randN)
	key, err := westack.CreateApiKey(app, name, []string{"apikey:read"}, nil, systemContext)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	err = westack.RevokeApiKey(app, key, systemContext)
	assert.NoError(t, err)

	m, err := app.FindModel("ApiKey")
	assert.NoError(t, err)

	inst, err := m.FindOne(&wst.Filter{Where: &wst.Where{"key": key}}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, inst)

	assert.False(t, inst.GetBoolean("enabled", true))
}
