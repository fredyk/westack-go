package tests

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_NewUserAndRole(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)
}

func Test_NewUserAndRoleWithExistingRole(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	roleModel := app.FindModelsWithClass("Role")[0]
	role, err := roleModel.Create(wst.M{
		"name": fmt.Sprintf("role-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, role.Id)

	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)
}

func Test_NewUserAndRoleWithExistingUser(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	user, err := userModel.Create(wst.M{
		"username": fmt.Sprintf("user-%v", randN),
		"password": fmt.Sprintf("pwd-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)

	user, err = westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)
}

func Test_NewUserAndRoleWithExistingUserAndRole(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	roleModel := app.FindModelsWithClass("Role")[0]
	role, err := roleModel.Create(wst.M{
		"name": fmt.Sprintf("role-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, role.Id)

	user, err := userModel.Create(wst.M{
		"username": fmt.Sprintf("user-%v", randN),
		"password": fmt.Sprintf("pwd-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)

	user, err = westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)
}

func Test_NewUserAndRoleWithExistingUserAndRoleAndUserRolesAndRoleMapping(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	roleModel := app.FindModelsWithClass("Role")[0]
	role, err := roleModel.Create(wst.M{
		"name": fmt.Sprintf("role-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, role.Id)

	user, err := userModel.Create(wst.M{
		"username": fmt.Sprintf("user-%v", randN),
		"password": fmt.Sprintf("pwd-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)

	roleMappingModel, err := app.FindModel("RoleMapping")
	userRole, err := roleMappingModel.Create(wst.M{
		"type":        "USER",
		"roleId":      role.Id,
		"principalId": user.Id,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userRole.Id)

	user, err = westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)
}

func Test_NewUserAndRoleEmptyUsername(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: "",
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_NewUserAndRoleEmptyPassword(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: "",
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_NewUserAndRoleEmptyRoles(t *testing.T) {
	randN := 1e6 + rand.Intn(8999999)
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}
