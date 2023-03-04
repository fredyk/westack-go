package westack

import (
	"fmt"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/model"
)

type UserWithRoles struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

func UpsertUserWithRoles(app *WeStack, userToUpsert UserWithRoles, eventContext *model.EventContext) (user *model.Instance, err error) {
	var role *model.Instance
	var roleMapping *model.Instance

	var userModel *model.Model
	var roleModel *model.Model

	// validate:
	if userToUpsert.Username == "" {
		err = fmt.Errorf("username is required")
		return
	}
	if userToUpsert.Password == "" {
		err = fmt.Errorf("password is required")
		return
	}
	if len(userToUpsert.Roles) == 0 {
		err = fmt.Errorf("roles is required")
		return
	}

	foundModels := app.FindModelsWithClass("User")
	if len(foundModels) == 0 {
		err = fmt.Errorf("user model not found")
		return
	}
	foundModels = app.FindModelsWithClass("Role")
	if len(foundModels) == 0 {
		err = fmt.Errorf("role model not found")
		return
	}
	userModel = foundModels[0]
	roleModel = foundModels[0]

	// Check if the user exists
	user, err = userModel.FindOne(&wst.Filter{
		Where: &wst.Where{
			"username": userToUpsert.Username,
		},
	}, eventContext)
	if err != nil {
		return
	}
	if user == nil {
		// Create the user
		user, err = userModel.Create(wst.M{
			"username": userToUpsert.Username,
			"password": userToUpsert.Password,
		}, eventContext)
		if err != nil {
			return
		}
		fmt.Printf("User %v created with id %v\n", userToUpsert.Username, user.Id)
	} else {
		fmt.Printf("User %v already exists\n", userToUpsert.Username)
	}

	for _, roleName := range userToUpsert.Roles {

		// find role model to add the role
		role, err = findRoleWithModel(eventContext, roleName, roleModel)
		if err != nil {
			return
		}
		// check if role mapping exists
		roleMapping, err = app.roleMappingModel.FindOne(&wst.Filter{
			Where: &wst.Where{
				"principalType": "USER",
				"principalId":   user.Id,
				"roleId":        role.Id,
			},
		}, eventContext)
		if err != nil {
			return
		}
		if roleMapping == nil {
			roleMapping, err = app.roleMappingModel.Create(wst.M{
				"principalType": "USER",
				"principalId":   user.Id,
				"roleId":        role.Id,
			}, eventContext)
			if err != nil {
				return
			}
			fmt.Printf("Assigned role %v to user %v with mapping id %v\n", roleName, userToUpsert.Username, roleMapping.Id)
		} else {
			fmt.Printf("Role mapping already exists for user %s and role %s\n", userToUpsert.Username, roleName)
		}

	}

	return
}

func findRoleWithModel(eventContext *model.EventContext, roleName string, roleModel *model.Model) (roleInstance *model.Instance, err error) {

	// check if the role exists
	roleInstance, err = roleModel.FindOne(&wst.Filter{
		Where: &wst.Where{
			"name": roleName,
		},
	}, eventContext)
	if err != nil {
		return
	}

	if roleInstance == nil {
		// create the role
		roleInstance, err = roleModel.Create(wst.M{
			"name": roleName,
		}, eventContext)
		if err != nil {
			return
		}
		fmt.Printf("Created role %s with ID: %s\n", roleName, roleInstance.Id)
	} else {
		fmt.Printf("Role %s already exists\n", roleName)
	}

	return
}
