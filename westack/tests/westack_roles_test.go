package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_NewUserAndRole(t *testing.T) {
	randN := createRandomInt()
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, user.Id)
}

func Test_NewUserAndRoleWithExistingRole(t *testing.T) {
	randN := createRandomInt()
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
	randN := createRandomInt()
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
	randN := createRandomInt()
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
	randN := createRandomInt()
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
	randN := createRandomInt()
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: "",
		Password: fmt.Sprintf("pwd-%v", randN),
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_NewUserAndRoleEmptyPassword(t *testing.T) {
	randN := createRandomInt()
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: "",
		Roles:    []string{fmt.Sprintf("role-%v", randN)},
	}, systemContext)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_NewUserAndRoleEmptyRoles(t *testing.T) {
	randN := createRandomInt()
	user, err := westack.UpsertUserWithRoles(app, westack.UserWithRoles{
		Username: fmt.Sprintf("user-%v", randN),
		Password: fmt.Sprintf("pwd-%v", randN),
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
	adminToken, err := loginUser(adminUser, adminPassword, t)
	assert.Nil(t, err)
	if !assert.Contains(t, adminToken, "id") {
		t.Fatal("Missing id in result token")
	}
	adminBearer := adminToken["id"].(string)

	// Create a new user
	password := fmt.Sprintf("pwd-%v", createRandomInt())
	user, err := createUser(t, wst.M{
		"username": fmt.Sprintf("user-%v", createRandomInt()),
		"password": password,
	})
	assert.Nil(t, err)
	assert.Contains(t, user, "id")

	desiredRoles := []string{fmt.Sprintf("role-1-%v", createRandomInt()), fmt.Sprintf("role-2-%v", createRandomInt())}

	// Invoke remote method to assign role
	// Update Roles Definition:
	// method: PUT
	// url: http://localhost:8019/api/v1/users/{userId}/roles
	// request body: { roles: [role1, role2, ..., roleN] }
	// headers: { Authorization: Bearer {token}, Content-Type: application/json }
	// response: 200 { result: "OK" }
	// or 400 { error: { code: "ERROR_CODE", message: "Error message", details: { ... } } }
	// or 401 { error: { code: "AUTHORIZATION_REQUIRED", message: "Authorization required" } }
	// or 404 { error: { code: "USER_NOT_FOUND", message: "User not found" } }

	url := fmt.Sprintf("/api/v1/users/%v/roles", user["id"])
	updateRolesResponse, err := invokeApi(t, "PUT", url, wst.M{
		"roles": desiredRoles,
	}, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", adminBearer),
		"Content-Type":  "application/json",
	})
	assert.Nil(t, err)
	assert.Contains(t, updateRolesResponse, "result")
	assert.Equal(t, "OK", updateRolesResponse["result"])

	// Login as the user
	userToken, err := loginUser(user["username"].(string), password, t)
	assert.Nil(t, err)
	if !assert.Contains(t, userToken, "id") {
		t.Fatal("Missing id in result token")
	}
	userBearer := userToken["id"].(string)

	//Decode the payload of the userBearer token as a JWT
	jwtPayload, err := extractJWTPayload(t, userBearer, err)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(jwtPayload.Roles))
	assert.Contains(t, jwtPayload.Roles, "USER")
	assert.Contains(t, jwtPayload.Roles, desiredRoles[0])
	assert.Contains(t, jwtPayload.Roles, desiredRoles[1])

	// Invoke remote method to assign role, but with a non-admin user. This should fail with 401
	newDesiredRoles := []string{fmt.Sprintf("role-3-%v", createRandomInt()), fmt.Sprintf("role-4-%v", createRandomInt())}
	resp, err := invokeApi(t, "PUT", url, wst.M{
		"roles": newDesiredRoles,
	}, wst.M{
		"Authorization": fmt.Sprintf("Bearer %v", userBearer),
		"Content-Type":  "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, resp, "error")
	assert.EqualValues(t, 401, resp["error"].(map[string]interface{})["statusCode"])

	// Login again
	userToken, err = loginUser(user["username"].(string), password, t)
	assert.Nil(t, err)
	if !assert.Contains(t, userToken, "id") {
		t.Fatal("Missing id in result token")
	}
	userBearer = userToken["id"].(string)

	//Decode the payload of the userBearer token as a JWT
	jwtPayload, err = extractJWTPayload(t, userBearer, err)
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
