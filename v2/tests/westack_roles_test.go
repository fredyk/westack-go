package tests

import (
	"fmt"
	"github.com/fredyk/westack-go/v2/westack"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	wst "github.com/fredyk/westack-go/v2/common"
)

func Test_NewUserAndRole(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	user, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.GetID())
}

func Test_NewUserAndRoleWithExistingRole(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	roleModel := app.FindModelsWithClass("Role")[0]
	role, err := roleModel.Create(wst.M{
		"name": fmt.Sprintf("role-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, role.GetID())

	user, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.GetID())
}

func Test_NewUserAndRoleWithExistingUser(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	user, err := invokeApiJsonM(t, "POST", "/accounts", wst.M{
		"username": fmt.Sprintf("user-%v", randN),
		"password": fmt.Sprintf("pwD-%v", randN),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, user, "id")

	userFromUpsert, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userFromUpsert.GetID())
}

func Test_NewUserAndRoleWithExistingUserAndRole(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	roleModel := app.FindModelsWithClass("Role")[0]
	role, err := roleModel.Create(wst.M{
		"name": fmt.Sprintf("role-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, role.GetID())

	user, err := invokeApiJsonM(t, "POST", "/accounts", wst.M{
		"username": fmt.Sprintf("user-%v", randN),
		"password": fmt.Sprintf("pwD-%v", randN),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, user, "id")

	userFromUpsert, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userFromUpsert.GetID())
}

func Test_NewUserAndRoleWithExistingUserAndRoleAndUserRolesAndRoleMapping(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	roleModel := app.FindModelsWithClass("Role")[0]
	role, err := roleModel.Create(wst.M{
		"name": fmt.Sprintf("role-%v", randN),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, role.GetID())

	user, err := invokeApiJsonM(t, "POST", "/accounts", wst.M{
		"username": fmt.Sprintf("user-%v", randN),
		"password": fmt.Sprintf("pwD-%v", randN),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, user, "id")

	roleMappingModel, err := app.FindModel("RoleMapping")
	assert.NoError(t, err)
	userRole, err := roleMappingModel.Create(wst.M{
		"type":        "USER",
		"roleId":      role.GetID(),
		"principalId": user.GetString("id"),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userRole.GetID())

	userFromUpsert, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, userFromUpsert.GetID())
}

func Test_NewUserAndRoleEmptyUsername(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	user, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: "",
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_NewUserAndRoleEmptyPassword(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	user, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: "",
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_NewUserAndRoleEmptyRoles(t *testing.T) {

	t.Parallel()

	randN := createRandomInt()
	user, err := westack.UpsertAccountWithRoles(app, westack.AccountWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwD-%v", randN),
		Roles:    []string{},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_RemoteAssignRole(t *testing.T) {

	t.Parallel()

	// Obtain admin user and password from environment variables
	adminUser := os.Getenv("WST_ADMIN_USERNAME")
	adminPassword := os.Getenv("WST_ADMIN_PWD")

	// Login as admin
	adminToken, err := loginAccount(adminUser, adminPassword, t)
	assert.Nil(t, err)
	if !assert.Contains(t, adminToken, "id") {
		t.Fatal("Missing id in result token")
	}
	adminBearer := adminToken["id"].(string)

	// Create a new user
	password := fmt.Sprintf("pwD-%v", createRandomInt())
	user := createAccount(t, wst.M{
		"username": fmt.Sprintf("user-%v", createRandomInt()),
		"password": password,
	})

	desiredRoles := []string{fmt.Sprintf("role-1-%v", createRandomInt()), fmt.Sprintf("role-2-%v", createRandomInt())}

	url := fmt.Sprintf("/accounts/%v/roles", user["id"])
	updateRolesResponse, err := invokeApiJsonM(t, "PUT", url, wst.M{
		"roles": desiredRoles,
	}, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", adminBearer),
		"Content-Type":  "application/json",
	})
	assert.Nil(t, err)
	assert.Contains(t, updateRolesResponse, "result")
	assert.Equal(t, "OK", updateRolesResponse["result"])

	// Login as the user
	userToken, err := loginAccount(user.GetString("username"), password, t)
	assert.Nil(t, err)
	if !assert.Contains(t, userToken, "id") {
		t.Fatal("Missing id in result token")
	}
	userBearer := userToken["id"].(string)

	//Decode the jwtInfo of the userBearer token as a JWT
	jwtPayload := extractJWTPayload(t, userBearer)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(jwtPayload.Roles))
	assert.Contains(t, jwtPayload.Roles, "USER")
	assert.Contains(t, jwtPayload.Roles, desiredRoles[0])
	assert.Contains(t, jwtPayload.Roles, desiredRoles[1])

	// Invoke remote method to assign role, but with a non-admin user. This should fail with 401
	newDesiredRoles := []string{fmt.Sprintf("role-3-%v", createRandomInt()), fmt.Sprintf("role-4-%v", createRandomInt())}
	resp, err := invokeApiJsonM(t, "PUT", url, wst.M{
		"roles": newDesiredRoles,
	}, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", userBearer),
		"Content-Type":  "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, resp, "error")
	assert.EqualValues(t, 401, resp["error"].(map[string]interface{})["statusCode"])

	// Login again
	userToken, err = loginAccount(user.GetString("username"), password, t)
	assert.Nil(t, err)
	if !assert.Contains(t, userToken, "id") {
		t.Fatal("Missing id in result token")
	}
	userBearer = userToken["id"].(string)

	//Decode the jwtInfo of the userBearer token as a JWT
	jwtPayload = extractJWTPayload(t, userBearer)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(jwtPayload.Roles))
	// The new roles should not be present
	assert.NotContainsf(t, jwtPayload.Roles, newDesiredRoles[0], "Role %v should not be present", newDesiredRoles[0])
	assert.NotContainsf(t, jwtPayload.Roles, newDesiredRoles[1], "Role %v should not be present", newDesiredRoles[1])
	// The old roles should still be present
	assert.Containsf(t, jwtPayload.Roles, "USER", "Role %v should be present", "USER")
	assert.Containsf(t, jwtPayload.Roles, desiredRoles[0], "Role %v should be present", desiredRoles[0])
	assert.Containsf(t, jwtPayload.Roles, desiredRoles[1], "Role %v should be present", desiredRoles[1])
}
