package westack

import (
	"fmt"
	"os"
	"strings"
	"time"

	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
	"github.com/fredyk/westack-go/v2/model"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var accountCredentialsProperties = []string{"email", "username", "password", "access_token", "refresh_token"}

func createCasbinModel(loadedModel *model.StatefulModel, app *WeStack, config *model.Config) error {
	casbModel := casbinmodel.NewModel()

	basePoliciesDirectory := app.Viper.GetString("casbin.policies.outputDirectory")
	_, err := os.Stat(basePoliciesDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(basePoliciesDirectory, os.ModePerm)
			if err != nil {
				return fmt.Errorf("could not create policies directory %v: %v", basePoliciesDirectory, err)
			}
		} else {
			return fmt.Errorf("could not check policies directory %v: %v", basePoliciesDirectory, err)
		}
	}

	f, err := os.OpenFile(fmt.Sprintf("%v/%v.policies.csv", basePoliciesDirectory, loadedModel.Name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("could not open policies file %v: %v", loadedModel.Name, err)
	}
	// TODO: How to test this error?
	_ = f.Close()
	//err = f.Close()
	//if err != nil {
	//	return err
	//}

	adapter := fileadapter.NewAdapter(fmt.Sprintf("%v/%v.policies.csv", basePoliciesDirectory, loadedModel.Name))

	requestDefinition := "sub, obj, act"
	policyDefinition := "sub, obj, act, eft"
	roleDefinition := "_, _"
	policyEffect := "subjectPriority(p.eft) || deny"
	matchersDefinition := fmt.Sprintf("(	((p.sub == '$owner' && isOwner(r.sub, r.obj, p.obj)) || g(r.sub, p.sub)) && keyMatch(r.obj, p.obj) && (g(r.act, p.act) || keyMatch(r.act, p.act))  )")
	if loadedModel.Config.Casbin.RequestDefinition != "" {
		requestDefinition = loadedModel.Config.Casbin.RequestDefinition
	}
	if loadedModel.Config.Casbin.PolicyDefinition != "" {
		policyDefinition = loadedModel.Config.Casbin.PolicyDefinition
	}
	if loadedModel.Config.Casbin.RoleDefinition != "" {
		roleDefinition = loadedModel.Config.Casbin.RoleDefinition
	}
	if loadedModel.Config.Casbin.PolicyEffect != "" {
		policyEffect = strings.ReplaceAll(loadedModel.Config.Casbin.PolicyEffect, "$default", policyEffect)
	}
	if loadedModel.Config.Casbin.MatchersDefinition != "" {
		matchersDefinition = strings.ReplaceAll(loadedModel.Config.Casbin.MatchersDefinition, "$default", " ( "+matchersDefinition+" ) ")
	}

	casbModel.AddDef("r", "r", replaceVarNames(requestDefinition))
	casbModel.AddDef("p", "p", replaceVarNames(policyDefinition))
	casbModel.AddDef("g", "g", replaceVarNames(roleDefinition))
	casbModel.AddDef("e", "e", replaceVarNames(policyEffect))
	casbModel.AddDef("m", "m", replaceVarNames(matchersDefinition))

	if len(loadedModel.Config.Casbin.Policies) > 0 {
		for _, p := range loadedModel.Config.Casbin.Policies {
			casbModel.AddPolicy("p", "p", []string{replaceVarNames(p)})
		}
	} else if config.Base != "Account" && config.Base != "App" {
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,read_write,allow")})
	}

	if config.Base == "Account" {
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,create,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,login,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,resetPassword,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,findSelf,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,sendVerificationEmail,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,performEmailVerification,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,refreshToken,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,findById,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,instance_updateAttributes,allow")})
		// TODO: check https://github.com/fredyk/westack-go/issues/447
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,instance_delete,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("admin,*,user_upsertRoles,allow")})
	}
	if config.Base == "App" {
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("admin,*,create,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,read_write,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,createToken,allow")})
	}
	loadedModel.CasbinModel = &casbModel
	loadedModel.CasbinAdapter = &adapter

	return adapter.SavePolicy(casbModel)
}

func setupAccountModel(loadedModel *model.StatefulModel, app *WeStack) {
	loadedModel.On(string(wst.OperationNameLogin), func(ctx *model.EventContext) error {
		data := ctx.Data
		email := data.GetString("email")
		username := data.GetString("username")

		if email == "" && username == "" {
			return wst.CreateError(fiber.ErrBadRequest, "USERNAME_EMAIL_REQUIRED", fiber.Map{"message": "username or email is required"}, "ValidationError")
		}

		if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
			return wst.CreateError(fiber.ErrUnauthorized, "PASSWORD_REQUIRED", fiber.Map{"message": "password is required"}, "ValidationError")
		}

		var where wst.Where
		if email != "" {
			where = wst.Where{"email": email}
		} else {
			where = wst.Where{"username": username}
		}
		accountCredentialsCursor := app.accountCredentialsModel.FindMany(&wst.Filter{
			Where: &where,
			Include: &wst.Include{
				{Relation: "account"},
			},
		}, &model.EventContext{Bearer: &model.BearerToken{Account: &model.BearerAccount{System: true}}})
		accounts, err := accountCredentialsCursor.All()
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			if accountCredentialsCursor.(*model.ChannelCursor).Err != nil {
				return accountCredentialsCursor.(*model.ChannelCursor).Err
			}
			if loadedModel.App.Debug {
				app.Logger().Printf("no user found with email or username %v\n", email)
			}
			return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
		}
		firstAccountCredentials := accounts[0]
		//accountCredentialsData := firstAccountCredentials.ToJSON()
		savedPassword := firstAccountCredentials.GetString("password")

		fullAccount := firstAccountCredentials.GetOne("account").(*model.StatefulInstance)
		ctx.Instance = fullAccount

		saltedPassword := fmt.Sprintf("%s%s", string(loadedModel.App.JwtSecretKey), (*data)["password"].(string))
		err = bcrypt.CompareHashAndPassword([]byte(savedPassword), []byte(saltedPassword))
		if err != nil {
			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("bcrypt.CompareHashAndPassword error: %v\n", err)
			}
			err = bcrypt.CompareHashAndPassword([]byte(savedPassword), []byte((*data)["password"].(string)))
		} else {
			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("bcrypt.CompareHashAndPassword success with salt\n")
			}
		}
		if err != nil {
			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("bcrypt.CompareHashAndPassword error: %v\n", err)
			}
			return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
		}

		userIdHex := fullAccount.Id.(primitive.ObjectID).Hex()

		roleNames := []string{"USER"}
		if app.roleMappingModel != nil {
			ctx.Bearer = &model.BearerToken{
				Account: &model.BearerAccount{
					System: true,
				},
				Roles: []model.BearerRole{},
			}
			roleContext := &model.EventContext{
				BaseContext:            ctx,
				DisableTypeConversions: true,
			}
			//roleEntries, err := app.roleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
			// TODO: How to test this error?
			roleEntries, _ := app.roleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
				"principalType": "USER",
				"$or": []wst.M{
					{
						"principalId": userIdHex,
					},
					{
						"principalId": fullAccount.Id,
					},
				},
			}, Include: &wst.Include{{Relation: "role"}}}, roleContext).All()
			//if err != nil {
			//	return err
			//}
			for _, roleEntry := range roleEntries {
				role := roleEntry.GetOne("role")
				roleNames = append(roleNames, role.ToJSON()["name"].(string))
			}
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"accountId": userIdHex,
			"created":   time.Now().UnixMilli(),
			"ttl":       604800 * 2 * 1000,
			"roles":     roleNames,
		})

		tokenString, err := token.SignedString(loadedModel.App.JwtSecretKey)

		ctx.StatusCode = fiber.StatusOK
		ctx.Result = wst.LoginResult{Id: tokenString, AccountId: userIdHex}
		return nil
	})
}

func setupInternalModels(config *model.Config, app *WeStack, dataSource *datasource.Datasource) {
	roleMappingModel := model.New(&model.Config{
		Name:   "RoleMapping",
		Plural: "role-mappings",
		Base:   "PersistedModel",
		//Datasource: config.Datasource,
		Public:     true,
		Properties: nil,
		Relations: &map[string]*model.Relation{
			"role": {
				Type:  "belongsTo",
				Model: config.Name,
				//PrimaryKey: "",
				//ForeignKey: "",
			},
			"account": {
				Type:  "belongsTo",
				Model: "Account",
				//PrimaryKey: "",
				//ForeignKey: "",
			},
		},
		Casbin: model.CasbinConfig{
			Policies: []string{
				"$owner,*,__get__role,allow",
				"roleManager,*,read,allow",
				"roleManager,*,write,allow",
			},
		},
	}, app.modelRegistry)
	roleMappingModel.(*model.StatefulModel).App = app.asInterface()
	roleMappingModel.(*model.StatefulModel).Datasource = dataSource

	app.roleMappingModel = roleMappingModel.(*model.StatefulModel)

	foreignKey := "accountId"
	accountCredentialsModel := model.New(&model.Config{
		Name:   "AccountCredentials",
		Plural: "account-credentials",
		Base:   "AccountCredentials",
		Public: false,
		Properties: map[string]model.Property{
			"type": {
				Type: "string",
			},
			"email": {
				Type: "email",
			},
			"password": {
				Type: "string",
			},
			"access_token": {
				Type: "string",
			},
			"refresh_token": {
				Type: "string",
			},
		},
		Validations: []model.Validation{
			{
				If: map[string]model.Condition{
					"type": {
						Equals: "password",
					},
				},
				Then: &model.Validation{
					Properties: map[string]model.Validation{
						"password": {
							NotEmpty: true,
						},
					},
					OneOf: []model.Validation{
						{
							Properties: map[string]model.Validation{
								"email": {
									NotEmpty: true,
								},
							},
						},
						{
							Properties: map[string]model.Validation{
								"username": {
									NotEmpty: true,
								},
							},
						},
					},
				},
			},
			{
				If: map[string]model.Condition{
					"type": {
						Equals: "oauth",
					},
				},
				Then: &model.Validation{
					Properties: map[string]model.Validation{
						"access_token": {
							NotEmpty: true,
						},
						"refresh_token": {
							NotEmpty: true,
						},
					},
				},
			},
		},
		Relations: &map[string]*model.Relation{
			"account": {
				Type:       "belongsTo",
				Model:      "Account",
				ForeignKey: &foreignKey,
			},
		},
		Hidden: []string{"password"},
		Casbin: model.CasbinConfig{
			Policies: []string{
				"$owner,*,__get__account,allow",
			},
		},
	}, app.modelRegistry)
	accountCredentialsModel.(*model.StatefulModel).App = app.asInterface()
	accountCredentialsModel.(*model.StatefulModel).Datasource = dataSource

	app.accountCredentialsModel = accountCredentialsModel.(*model.StatefulModel)
}

func GetRoleNames(RoleMappingModel *model.StatefulModel, userIdHex string, userId primitive.ObjectID) ([]string, error) {
	roleNames := []string{"USER"}

	if RoleMappingModel != nil {
		ctx := &model.EventContext{Bearer: &model.BearerToken{
			Account: &model.BearerAccount{
				System: true,
			},
			Roles: []model.BearerRole{},
		}}
		roleContext := &model.EventContext{
			BaseContext:            ctx,
			DisableTypeConversions: true,
		}
		roleEntries, err := RoleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
			"principalType": "USER",
			"$or": []wst.M{
				{
					"principalId": userIdHex,
				},
				{
					"principalId": userId,
				},
			},
		}, Include: &wst.Include{{Relation: "role"}}}, roleContext).All()
		if err != nil {
			return roleNames, err
		}
		for _, roleEntry := range roleEntries {
			role := roleEntry.GetOne("role")
			roleNames = append(roleNames, role.ToJSON()["name"].(string))
		}
	}
	return roleNames, nil
}

func CreateNewToken(userIdHex string, AccountModel *model.StatefulModel, roles []string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountId": userIdHex,
		"created":   time.Now().UnixMilli(),
		"ttl":       604800 * 2 * 1000,
		"roles":     roles,
	})
	tokenString, err := token.SignedString(AccountModel.App.JwtSecretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
