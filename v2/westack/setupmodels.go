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

var accountCredentialsProperties = []string{"email", "username", "password", "accessToken", "refreshToken", "scope", "expiry", "tokenType", "provider"}

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
	matchersDefinition := "(	((HasPrefix(p.sub, '_OWNER') && r.sub != '_EVERYONE_' && isOwner(r.sub, r.obj, p.sub, p.obj, p.act)) || g(r.sub, p.sub)) && keyMatch(r.obj, p.obj) && (g(r.act, p.act) || keyMatch(r.act, p.act))  )"
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
		addOAuthLoginPolicies(casbModel)
		// TODO: any other oauth login providers
		casbModel.AddPolicy("p", "p", []string{replaceVarNames(fmt.Sprintf("$everyone,*,%s,allow", wst.OperationNameCreate))})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames(fmt.Sprintf("$everyone,*,%s,allow", wst.OperationNameLogin))})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,resetPassword,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames(fmt.Sprintf("$authenticated,*,%s,allow", wst.OperationNameFindSelf))})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,sendVerificationEmail,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,performEmailVerification,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames(fmt.Sprintf("$authenticated,*,%s,allow", wst.OperationNameRefreshToken))})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames(fmt.Sprintf("$authenticated,*,%s,allow", wst.OperationNameValidateToken))})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,findById,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,instance_updateAttributes,allow")})
		// TODO: check https://github.com/fredyk/westack-go/issues/447
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,instance_delete,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("admin,*,user_upsertRoles,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,user_enableMfa,allow")})
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

func addOAuthLoginPolicies(casbModel casbinmodel.Model) {
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,googleLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,googleLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,metaLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,metaLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,googleLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,googleLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,amazonLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,amazonLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,bitbucketLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,bitbucketLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,cernLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,cernLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,facebookLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,facebookLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,fitbitLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,fitbitLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,foursquareLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,foursquareLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,githubLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,githubLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,githubGetOauthCredentials,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,gitlabLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,gitlabLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,gitlabGetOauthCredentials,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,herokuLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,herokuLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,hipchatLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,hipchatLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,instagramLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,instagramLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,kakaoLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,kakaoLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,linkedinLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,linkedinLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,mailchimpLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,mailchimpLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,mailruLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,mailruLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,mediamathLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,mediamathLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,microsoftLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,microsoftLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,nokiahealthLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,nokiahealthLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,odnoklassnikiLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,odnoklassnikiLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,paypalLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,paypalLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,slackLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,slackLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,spotifyLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,spotifyLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,stackoverflowLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,stackoverflowLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,twitchLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,twitchLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,uberLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,uberLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,vkLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,vkLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,yahooLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,yahooLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,yandexLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,yandexLoginCallback,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,westackLogin,allow")})
	casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,westackLoginCallback,allow")})
}

func setupAccountModel(loadedModel *model.StatefulModel, app *WeStack) {
	loadedModel.On(string(wst.OperationNameLogin), func(ctx *model.EventContext) error {
		data := ctx.Data
		email := data.GetString("email")
		username := data.GetString("username")
		mfaVerificationCode := data.GetString("verificationCode")
		password := data.GetString("password")
		clientIP := ctx.Ctx.IP()
		userAgent := ctx.Ctx.Get("User-Agent")

		// Input sanitization
		email = strings.TrimSpace(email)
		username = strings.TrimSpace(username)

		// Rate limiting check
		identifier := clientIP
		if email != "" {
			identifier = email
		} else if username != "" {
			identifier = username
		}

		if err := app.checkLoginRateLimit(identifier); err != nil {
			app.logSecurityEvent("LOGIN_RATE_LIMITED", identifier, false, map[string]interface{}{
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			return err
		}

		var ttl int64 = 604800 * 2 * 1000
		receivedTtl := data.GetInt64("ttl")
		if receivedTtl > 0 {
			ttl = receivedTtl
		}

		if email == "" && username == "" {
			app.logSecurityEvent("LOGIN_INVALID_INPUT", identifier, false, map[string]interface{}{
				"error":     "missing_credentials",
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			return wst.CreateError(fiber.ErrBadRequest, "USERNAME_EMAIL_REQUIRED", fiber.Map{"message": "username or email is required"}, "ValidationError")
		}

		if password == "" {
			app.logSecurityEvent("LOGIN_INVALID_INPUT", identifier, false, map[string]interface{}{
				"error":     "missing_password",
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			return wst.CreateError(fiber.ErrUnauthorized, "PASSWORD_REQUIRED", fiber.Map{"message": "password is required"}, "ValidationError")
		}

		// Check account lockout
		if err := app.checkAccountLockout(identifier); err != nil {
			app.logSecurityEvent("LOGIN_ACCOUNT_LOCKED", identifier, false, map[string]interface{}{
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			return err
		}

		// Password provider filter - reusable logic
		passwordProviderFilter := []wst.M{
			{"provider": string(ProviderPassword)},
			{"provider": ""},
			{"password": wst.M{"$exists": true}},
		}

		var where wst.Where
		if email != "" {
			where = wst.Where{
				"email": email,
				"$or":   passwordProviderFilter,
			}
		} else {
			where = wst.Where{
				"username": username,
				"$or":      passwordProviderFilter,
			}
		}

		// Use FindOne instead of FindMany for security
		firstAccountCredentials, err := app.accountCredentialsModel.FindOne(&wst.Filter{
			Where: &where,
			Include: &wst.Include{
				{Relation: "account"},
			},
		}, &model.EventContext{Bearer: &model.BearerToken{Account: &model.BearerAccount{System: true}}})

		// Timing attack protection - always perform password comparison
		var savedPassword string
		var linkedAccount model.Instance
		var fullAccount *model.StatefulInstance

		if err != nil || firstAccountCredentials == nil {
			// Perform dummy password comparison to prevent timing attacks
			bcrypt.CompareHashAndPassword([]byte("$2a$11$dummy.hash.to.prevent.timing.attacks"), []byte("dummy"))
			app.logSecurityEvent("LOGIN_FAILED", identifier, false, map[string]interface{}{
				"error":     "invalid_credentials",
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			app.recordFailedLogin(identifier)
			return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
		}

		savedPassword = firstAccountCredentials.GetString("password")
		linkedAccount = firstAccountCredentials.GetOne("account")

		if linkedAccount == nil {
			// Perform dummy password comparison to prevent timing attacks
			bcrypt.CompareHashAndPassword([]byte("$2a$11$dummy.hash.to.prevent.timing.attacks"), []byte("dummy"))
			app.logSecurityEvent("LOGIN_FAILED", identifier, false, map[string]interface{}{
				"error":     "orphan_credentials",
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			app.recordFailedLogin(identifier)
			return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
		}

		fullAccount = linkedAccount.(*model.StatefulInstance)
		ctx.Instance = fullAccount

		// Password comparison with timing attack protection
		saltedPassword := fmt.Sprintf("%s%s", string(loadedModel.App.JwtSecretKey), password)
		err = bcrypt.CompareHashAndPassword([]byte(savedPassword), []byte(saltedPassword))

		if err != nil {
			app.logSecurityEvent("LOGIN_FAILED", identifier, false, map[string]interface{}{
				"error":     "invalid_password",
				"accountId": fullAccount.GetString("id"),
				"ip":        clientIP,
				"userAgent": userAgent,
			})
			app.recordFailedLogin(identifier)
			return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
		}

		// Successful login
		app.logSecurityEvent("LOGIN_SUCCESS", identifier, true, map[string]interface{}{
			"accountId": fullAccount.GetString("id"),
			"ip":        clientIP,
			"userAgent": userAgent,
		})
		app.clearFailedLogins(identifier)

		userIdHex := fullAccount.Id.(primitive.ObjectID).Hex()

		mfaEnabled, err := model.HasMfaEnabled(app.mfaModel, userIdHex)
		if err != nil {
			return wst.CreateError(fiber.ErrInternalServerError, "MFA_INTERNAL_ERROR", fiber.Map{"message": "MFA internal error"}, "Error")
		}

		if mfaEnabled {
			if mfaVerificationCode == "" {
				return wst.CreateError(fiber.ErrUnauthorized, "MFA_REQUIRED", fiber.Map{"message": "MFA verification code required"}, "Error")
			}
			ok, err := model.LoginVerifyMfa(app.mfaModel, userIdHex, mfaVerificationCode)
			if err != nil {
				return wst.CreateError(fiber.ErrBadRequest, "MFA_ERROR", fiber.Map{"message": "MFA error"}, "Error")
			}
			if !ok {
				return wst.CreateError(fiber.ErrUnauthorized, "MFA_FAILED", fiber.Map{"message": "MFA failed"}, "Error")
			}
		}

		roleNames := []string{"USER"}
		if mfaEnabled {
			roleNames = append(roleNames, "USER:mfa")
		}
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
			roleEntries := app.roleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
				"principalType": "USER",
				"$or": []wst.M{
					{
						"principalId": userIdHex,
					},
					{
						"principalId": fullAccount.Id,
					},
				},
			}, Include: &wst.Include{{Relation: "role"}}}, roleContext)
			roles, _ := roleEntries.All()
			for _, roleEntry := range roles {
				role := roleEntry.GetOne("role")
				roleNames = append(roleNames, role.GetString("name"))
			}
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"accountId": userIdHex,
			"created":   time.Now().UnixMilli(),
			"ttl":       ttl,
			"roles":     roleNames,
		})

		tokenString, err := token.SignedString(loadedModel.App.JwtSecretKey)
		if err != nil {
			return wst.CreateError(fiber.ErrInternalServerError, "JWT_ERROR", fiber.Map{"message": fmt.Sprintf("jwt error: %v", err)}, "Error")
		}

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
		Public:     false,
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
			"provider": {
				Type: "string",
			},
			"email": {
				Type: "email",
			},
			"password": {
				Type: "string",
			},
			"accessToken": {
				Type: "string",
			},
			"refreshToken": {
				Type: "string",
			},
			"mfaEnabled": {
				Type: "boolean",
			},
		},
		Validations: []model.Validation{
			{
				If: map[string]model.Condition{
					"provider": {
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
					"provider": {
						Equals: "oauth",
					},
				},
				Then: &model.Validation{
					Properties: map[string]model.Validation{
						"accessToken": {
							NotEmpty: true,
						},
						"refreshToken": {
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

	app.accountCredentialsModel = accountCredentialsModel.(*model.StatefulModel)

	// Mfa model
	mfaModel := model.New(&model.Config{
		Name:   "Mfa",
		Plural: "mfas",
		Base:   "Mfa",
		Public: false,
		Properties: map[string]model.Property{
			"provider": {
				Type:    "string",
				Default: "totp",
			},
			"status": {
				Type:     "string",
				Required: true,
			},
			"secretKey": {
				Type:     "string",
				Required: true,
			},
			"backupCodes": {
				Type:     "string",
				Required: true,
			},
		},
		Relations: &map[string]*model.Relation{
			"account": {
				Type:       "belongsTo",
				Model:      "Account",
				ForeignKey: &foreignKey,
			},
		},
		Casbin: model.CasbinConfig{
			Policies: []string{
				"$owner,*,__get__account,allow",
			},
		},
	}, app.modelRegistry)
	mfaModel.(*model.StatefulModel).App = app.asInterface()

	app.mfaModel = mfaModel.(*model.StatefulModel)

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
