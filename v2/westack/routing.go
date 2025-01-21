package westack

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/v2/utils"
)

func (app *WeStack) loadNotFoundRoutes() {
	for _, entry := range *app.modelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.debug {
				log.Println("[WARNING] Model", loadedModel.Name, "is not public")
			}
			continue
		}
		(*loadedModel.Router).Use(func(ctx *fiber.Ctx) error {
			log.Println("[WARNING] Unresolved method in " + loadedModel.Name + ": " + ctx.Method() + " " + ctx.Path())
			return ctx.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Shared class %#v has no method handling %v %v", loadedModel.Name, ctx.Method(), ctx.Path())}})
		})
	}
}

func (app *WeStack) loadModelsFixedRoutes() error {
	for _, entry := range *app.modelRegistry {
		loadedModel := entry

		e, err := casbin.NewEnforcer(*loadedModel.CasbinModel, *loadedModel.CasbinAdapter, app.debug)
		if err != nil {
			return fmt.Errorf("could not create casbin enforcer: %v", err)
		}

		loadedModel.Enforcer = e

		e.EnableAutoSave(true)
		e.AddFunction("isOwner", casbinOwnerFn(loadedModel))

		err = addDefaultCasbinRoles(app, err, e)
		if err != nil {
			return err
		}

		err = e.SavePolicy()
		if err != nil {
			return fmt.Errorf("could not save policy: %v", err)
		}

		err = e.LoadPolicy()
		if err != nil {
			return fmt.Errorf("could not load policy: %v", err)
		}

		if app.debug {
			loadedModel.CasbinModel.PrintModel()
		}

		if app.Viper.GetBool("casbin.dumpModels") {
			text := loadedModel.CasbinModel.ToText()
			modelsDumpDir := "common/models"
			if v := app.Viper.GetString("casbin.models.dumpDirectory"); v != "" {
				modelsDumpDir = v
			}
			err = os.WriteFile(fmt.Sprintf("%v/%v.casbin.dump.conf", modelsDumpDir, loadedModel.Name), []byte(text), 0600)
			if err != nil {
				return fmt.Errorf("could not write casbin dump: %v", err)
			}
		}

		if !loadedModel.Config.Public {
			if app.debug {
				log.Println("[WARNING] Model", loadedModel.Name, "is not public")
			}
			continue
		}

		mountBaseModelFixedRoutes(app, loadedModel)

		if loadedModel.Config.Base == "Account" {

			mountAccountModelFixedRoutes(loadedModel, app)

		} else if loadedModel.Config.Base == "App" {
			mountAppDynamicRoutes(loadedModel, app)
		}
	}
	return nil
}

func mountAccountModelFixedRoutes(loadedModel *model.StatefulModel, app *WeStack) {

	systemContext := &model.EventContext{
		Bearer: &model.BearerToken{
			Account: &model.BearerAccount{
				System: true,
			},
			Roles: []model.BearerRole{},
		},
	}

	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		return handleEvent(eventContext, loadedModel, string(wst.OperationNameLogin))
	}, model.RemoteMethodOptions{
		Name:        string(wst.OperationNameLogin),
		Description: "Logins an account",
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "data",
				Type:        "object",
				Description: "",
				Http:        model.ArgHttp{Source: "body"},
				Required:    false,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/login",
			Verb: "post",
		},
	},
	)

	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

		token, err := eventContext.GetBearer(loadedModel)
		if err != nil {
			return err
		}
		eventContext.ModelID = token.Account.Id
		return loadedModel.HandleRemoteMethod(string(wst.OperationNameFindById), eventContext)

	}, model.RemoteMethodOptions{
		Name:        string(wst.OperationNameFindSelf),
		Description: "Find user with their bearer",
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "filter",
				Type:        "string",
				Description: "",
				Http: model.ArgHttp{
					Source: "query",
				},
				Required: false,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/me",
			Verb: "get",
		},
	},
	)

	if app.debug {
		log.Println("Mount POST " + loadedModel.BaseUrl + "/reset-password")
	}
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		// Developer must implement this event
		return handleEvent(eventContext, loadedModel, "sendResetPasswordEmail")
	}, model.RemoteMethodOptions{
		Name: "resetPassword",
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "data",
				Type:        "object",
				Description: "",
				Http:        model.ArgHttp{Source: "body"},
				Required:    false,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/reset-password",
			Verb: "post",
		},
	})

	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		fmt.Println("verify user ", eventContext.Bearer.Account.Id)
		eventContext.Bearer.Claims["created"] = time.Now().Unix()
		eventContext.Bearer.Claims["ttl"] = 86400 * 2 * 1000
		eventContext.Bearer.Claims["allowsEmailVerification"] = true

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, eventContext.Bearer.Claims)

		tokenString, err := token.SignedString(loadedModel.App.JwtSecretKey)
		if err != nil {
			return err
		}
		eventContext.Bearer.Raw = tokenString

		return handleEvent(eventContext, loadedModel, "sendVerificationEmail")
	}, model.RemoteMethodOptions{
		Name: "sendVerificationEmail",
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "data",
				Type:        "object",
				Description: "",
				Http:        model.ArgHttp{Source: "body"},
				Required:    false,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/verify-mail",
			Verb: "post",
		},
	})

	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		// Developer must implement this event
		userId := eventContext.Bearer.Account.Id
		if userId == "" {
			return errors.New("no user id found in bearer")
		}
		if eventContext.Bearer.Claims["allowsEmailVerification"] == true {
			user, err := loadedModel.FindById(userId, nil, eventContext)
			if err != nil {
				return err
			}
			eventContext.SkipFieldProtection = true
			updated, err := user.UpdateAttributes(wst.M{
				"emailVerified": true,
			}, eventContext)
			if err != nil {
				return err
			}
			if app.debug {
				log.Println("Updated user ", updated)
			}
			redirectToUrl := eventContext.Query.GetString("redirect_uri")
			return eventContext.Ctx.Redirect(redirectToUrl)
		}

		return handleEvent(eventContext, loadedModel, "performEmailVerification")
	}, model.RemoteMethodOptions{
		Name: "performEmailVerification",
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "access_token",
				Type:        "string",
				Description: "",
				Http:        model.ArgHttp{Source: "query"},
				Required:    true,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/verify-mail",
			Verb: "get",
		},
	})

	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		authHeader := strings.TrimSpace(string(eventContext.Ctx.Request().Header.Peek("Authorization")))

		authBearerPair := strings.Split(authHeader, "Bearer ")
		tokenString := ""
		userIdHex := ""

		if len(authBearerPair) == 2 {

			bearerValue := authBearerPair[1]
			token, err := jwt.Parse(bearerValue, func(token *jwt.Token) (interface{}, error) {

				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}

				return loadedModel.App.JwtSecretKey, nil
			})

			if err != nil {
				fmt.Printf("[DEBUG] Invalid token: %s\n", err.Error())
			} else if token != nil {

				if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
					userIdHex = claims["accountId"].(string)
					userId, _ := primitive.ObjectIDFromHex(userIdHex)
					roleNames, err := GetRoleNames(app.roleMappingModel, userIdHex, userId)
					if err != nil {
						return err
					}

					newToken, err := CreateNewToken(userIdHex, loadedModel, roleNames)
					if err != nil {
						return err
					}
					tokenString = newToken
				} else {
					fmt.Println("[DEBUG] Invalid token: wrong claims")

					return errors.New("invalid token")
				}

			} else {
				return errors.New("invalid token")
			}

		} else {
			return errors.New("invalid Authorization header")
		}

		return eventContext.Ctx.JSON(fiber.Map{"id": tokenString, "accountId": userIdHex})
	}, model.RemoteMethodOptions{
		Name:        string(wst.OperationNameRefreshToken),
		Description: "Obtains current user",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/refresh-token",
			Verb: "post",
		},
	})

	// oauth2 client
	clientID := app.Viper.GetString("oauth2.google.clientID")
	googleOauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: app.Viper.GetString("oauth2.google.clientSecret"),
		RedirectURL:  fmt.Sprintf("%s%s/oauth/google/callback", app.Viper.GetString("publicOrigin"), loadedModel.BaseUrl),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}

	// google login endpoint
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

		oauthStateString := wst.CreateOauthStateString()
		cookie := ""
		if eventContext.Ctx.Cookies("SSID") != "" {
			cookie = eventContext.Ctx.Cookies("SSID")
		} else {
			cookie = wst.GenerateCookie()
			eventContext.Ctx.Cookie(&fiber.Cookie{
				Name:  "SSID",
				Value: cookie,
			})
		}
		utils.RegisterOauthState(cookie, oauthStateString)

		url := googleOauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
		return eventContext.Ctx.Redirect(url)
	}, model.RemoteMethodOptions{
		Name:        string(wst.OperationNameGoogleLogin),
		Description: "Logins with Google",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/oauth/google",
			Verb: "get",
		},
	})

	// google callback endpoint
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

		cookie := eventContext.Ctx.Cookies("SSID")
		oauthState, ok := utils.GetOauthState(cookie)
		if !ok {
			return wst.CreateError(fiber.ErrForbidden, "ERR_MISSING_OAUTH_STATE", fiber.Map{"message": "missing oauth state"}, "Error")
		}

		receivedState := eventContext.Query.GetString("state")
		if oauthState != receivedState {
			return wst.CreateError(fiber.ErrForbidden, "ERR_INVALID_OAUTH_STATE", fiber.Map{"message": "invalid oauth state"}, "Error")
		}

		oauthCode := eventContext.Query.GetString("code")
		token, err := googleOauthConfig.Exchange(eventContext.Ctx.Context(), oauthCode)
		if err != nil {
			return wst.CreateError(fiber.ErrForbidden, "ERR_OAUTH_EXCHANGE", fiber.Map{"message": "oauth exchange failed: " + err.Error()}, "Error")
		}

		userInfo, err := googleOauthConfig.Client(eventContext.Ctx.Context(), token).Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			return wst.CreateError(fiber.ErrForbidden, "ERR_OAUTH_USERINFO", fiber.Map{"message": "failed to get user info: " + err.Error()}, "Error")
		}

		defer userInfo.Body.Close()

		type userInfoResponse struct {
			Email string `json:"email"`
		}

		var userInfoData userInfoResponse
		err = json.NewDecoder(userInfo.Body).Decode(&userInfoData)
		if err != nil {
			return wst.CreateError(fiber.ErrForbidden, "ERR_OAUTH_USERINFO_DECODE", fiber.Map{"message": "failed to decode user info: " + err.Error()}, "Error")
		}

		// check if userCredentials exists
		userCredentials, err := app.accountCredentialsModel.FindOne(&wst.Filter{
			Where: &wst.Where{
				"email":    userInfoData.Email,
				"provider": ProviderGoogleOAuth2,
			},
			Include: &wst.Include{
				{
					Relation: "account",
				},
			},
		}, systemContext)

		if err != nil {
			fmt.Printf("[DEBUG] Error while fetching credentials by email-provider: %v\n", err)
			return err
		}

		var account *model.StatefulInstance
		var accountId string

		if userCredentials == nil {
			// search by password
			userCredentials, err = app.accountCredentialsModel.FindOne(&wst.Filter{
				Where: &wst.Where{
					"email": userInfoData.Email,
					"$or": []wst.M{
						{"provider": ProviderPassword},
						{"password": wst.M{"$exists": true}},
					},
				},
				Include: &wst.Include{
					{
						Relation: "account",
					},
				},
			}, systemContext)

			if err != nil {
				fmt.Printf("[DEBUG] Error while fetching credentials by email-password: %v\n", err)
				return err
			}

			if userCredentials == nil {

				// create new account
				fmt.Printf("[DEBUG] Creating new account for email: %v\n", userInfoData.Email)
				createdAccount, err := loadedModel.Create(wst.M{
					"email":         userInfoData.Email,
					"emailVerified": true,
					"provider":      ProviderGoogleOAuth2,
				}, systemContext)

				if err != nil {
					fmt.Printf("[DEBUG] Error while creating account: %v\n", err)
					return err
				}

				account = createdAccount.(*model.StatefulInstance)
				accountId = account.GetString("id")

			} else {

				accountId = userCredentials.GetString("accountId")

				account = userCredentials.GetOne("account").(*model.StatefulInstance)

			}

			// create new credentials
			fmt.Printf("[DEBUG] Creating new credentials for email: %v\n", userInfoData.Email)
			_, err = app.accountCredentialsModel.Create(wst.M{
				"accountId":    accountId,
				"email":        userInfoData.Email,
				"provider":     ProviderGoogleOAuth2,
				"accessToken":  token.AccessToken,
				"refreshToken": token.RefreshToken,
				"expiry":       token.Expiry,
				"tokenType":    token.TokenType,
				"scope":        token.Extra("scope"),
			}, systemContext)

			if err != nil {
				fmt.Printf("[DEBUG] Error while creating credentials: %v\n", err)
				return err
			}

		} else {
			account = userCredentials.GetOne("account").(*model.StatefulInstance)

			// update credentials
			fmt.Printf("[DEBUG] Updating credentials for email: %v\n", userInfoData.Email)
			_, err = userCredentials.UpdateAttributes(wst.M{
				"accessToken": token.AccessToken,
				"expiry":      token.Expiry,
				"tokenType":   token.TokenType,
				"scope":       token.Extra("scope"),
			}, systemContext)

			if err != nil {
				fmt.Printf("[DEBUG] Error while updating credentials: %v\n", err)
				return err
			}

		}

		roleNames := []string{"USER"}

		roleContext := &model.EventContext{
			BaseContext:            systemContext,
			DisableTypeConversions: true,
		}

		roleEntries, _ := app.roleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
			"principalType": "USER",
			"$or": []wst.M{
				{
					"principalId": accountId,
				},
				{
					"principalId": account.Id,
				},
			},
		}, Include: &wst.Include{{Relation: "role"}}}, roleContext).All()

		for _, roleEntry := range roleEntries {
			role := roleEntry.GetOne("role")
			roleNames = append(roleNames, role.ToJSON()["name"].(string))
		}

		ttl := 30 * 86400.0
		bearer := model.CreateBearer(eventContext.ModelID, float64(time.Now().Unix()), ttl, roleNames)
		// sign the bearer
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, bearer.Claims)
		tokenString, err := jwtToken.SignedString(loadedModel.App.JwtSecretKey)
		if err != nil {
			return err
		}

		return eventContext.Ctx.Redirect(app.Viper.GetString("oauth2.successRedirect") + "?access_token=" + tokenString)

	}, model.RemoteMethodOptions{
		Name:        string(wst.OperationNameGoogleLoginCallback),
		Description: "Google OAuth2 callback",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/oauth/google/callback",
			Verb: "get",
		},
	})

}

func mountBaseModelFixedRoutes(app *WeStack, loadedModel *model.StatefulModel) {
	if app.debug {
		log.Println("Mount GET " + loadedModel.BaseUrl)
	}
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		return handleEvent(eventContext, loadedModel, string(wst.OperationNameFindMany))
	}, model.RemoteMethodOptions{
		Name: string(wst.OperationNameFindMany),
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "filter",
				Type:        "string",
				Description: "",
				Http: model.ArgHttp{
					Source: "query",
				},
				Required: false,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/",
			Verb: "get",
		},
	})

	if app.debug {
		log.Println("Mount GET " + loadedModel.BaseUrl + "/count")
	}
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		return handleEvent(eventContext, loadedModel, string(wst.OperationNameCount))
	}, model.RemoteMethodOptions{
		Name: string(wst.OperationNameCount),
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "filter",
				Type:        "string",
				Description: "",
				Http: model.ArgHttp{
					Source: "query",
				},
				Required: false,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/count",
			Verb: "get",
		},
	})

	if app.debug {
		log.Println("Mount POST " + loadedModel.BaseUrl)
	}
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		return handleEvent(eventContext, loadedModel, string(wst.OperationNameCreate))
	}, model.RemoteMethodOptions{
		Name: string(wst.OperationNameCreate),
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "body",
				Type:        "object",
				Description: "",
				Http:        model.ArgHttp{Source: "body"},
				Required:    true,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/",
			Verb: "post",
		},
	})
}

func mountAppDynamicRoutes(loadedModel *model.StatefulModel, app *WeStack) {
	if app.debug {
		log.Println("Mount POST " + loadedModel.BaseUrl + "/:id/token")
	}
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		id := eventContext.Ctx.Params("id")
		ttl := eventContext.Data.GetFloat64("ttl")
		additionalRolesSt := eventContext.Data.GetString("roles")
		additionalRoles := make([]string, 0)
		if len(strings.TrimSpace(additionalRolesSt)) > 0 {
			additionalRoles = strings.Split(additionalRolesSt, ",")
			for i, role := range additionalRoles {
				additionalRoles[i] = strings.TrimSpace(role)
			}
		}
		if ttl <= 0.0 {
			ttl = 30 * 24 * 60 * 60
		}
		var asSt string
		var asStOk bool
		if eventContext.ModelID != nil {
			asSt, asStOk = eventContext.ModelID.(string)
		}
		if eventContext.ModelID == nil || asStOk && len(strings.TrimSpace(asSt)) == 0 {
			eventContext.ModelID = id
		}
		roles := []string{"APP"}
		roles = append(roles, additionalRoles...)
		bearer := model.CreateBearer(eventContext.ModelID, float64(time.Now().Unix()), ttl, roles)
		// sign the bearer
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, bearer.Claims)
		tokenString, err := token.SignedString(loadedModel.App.JwtSecretKey)
		if err != nil {
			return err
		}
		bearer.Raw = tokenString
		eventContext.Result = wst.M{
			"id":    bearer.Raw,
			"appId": eventContext.ModelID,
		}
		return nil
	}, model.RemoteMethodOptions{
		Name: string(wst.OperationNameCreateToken),
		Accepts: model.RemoteMethodOptionsHttpArgs{
			{
				Arg:         "data",
				Type:        "object",
				Description: "",
				Http:        model.ArgHttp{Source: "body"},
				Required:    true,
			},
		},
		Http: model.RemoteMethodOptionsHttp{
			Path: "/:id/token",
			Verb: "POST",
		},
	})
}

func addDefaultCasbinRoles(app *WeStack, err error, e *casbin.Enforcer) error {
	_, err = e.AddRoleForUser("findMany", replaceVarNames("read"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role findMany for user %v, err: %v\n", replaceVarNames("read"), err)
	}
	_, err = e.AddRoleForUser("findById", replaceVarNames("read"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role findById for user %v, err: %v\n", replaceVarNames("read"), err)
	}
	_, err = e.AddRoleForUser("count", replaceVarNames("read"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role count for user %v, err: %v\n", replaceVarNames("read"), err)
	}
	_, err = e.AddRoleForUser("create", replaceVarNames("write"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role create for user %v, err: %v\n", replaceVarNames("write"), err)
	}
	_, err = e.AddRoleForUser("instance_updateAttributes", replaceVarNames("write"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role instance_updateAttributes for user %v, err: %v\n", replaceVarNames("write"), err)
	}
	_, err = e.AddRoleForUser("instance_delete", replaceVarNames("write"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role instance_delete for user %v, err: %v\n", replaceVarNames("write"), err)
	}
	_, err = e.AddRoleForUser("read", replaceVarNames("read_write"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role read for user %v, err: %v\n", replaceVarNames("read_write"), err)
	}
	_, err = e.AddRoleForUser("write", replaceVarNames("read_write"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role write for user %v, err: %v\n", replaceVarNames("read_write"), err)
	}
	_, err = e.AddRoleForUser("read_write", replaceVarNames("*"))
	if app.debug {
		app.logger.Printf("[DEBUG] Added role read_write for user %v, err: %v\n", replaceVarNames("*"), err)
	}
	return nil
}

func (app *WeStack) loadModelsDynamicRoutes() {
	for _, entry := range *app.modelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.debug {
				log.Println("[WARNING] Model", loadedModel.Name, "is not public")
			}
			continue
		}

		if app.debug {
			log.Println("Mount GET " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

			id := eventContext.Ctx.Params("id")
			var asSt string
			var asStOk bool
			if eventContext.ModelID != nil {
				asSt, asStOk = eventContext.ModelID.(string)
			}
			if eventContext.ModelID == nil || asStOk && len(strings.TrimSpace(asSt)) == 0 {
				eventContext.ModelID = id
			}

			return handleEvent(eventContext, loadedModel, string(wst.OperationNameFindById))

		}, model.RemoteMethodOptions{
			Name: string(wst.OperationNameFindById),
			Accepts: model.RemoteMethodOptionsHttpArgs{
				{
					Arg:         "filter",
					Type:        "string",
					Description: "",
					Http:        model.ArgHttp{Source: "query"},
					Required:    false,
				},
			},
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "get",
			},
		})

		if app.debug {
			log.Println("Mount PATCH " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
			id, err := primitive.ObjectIDFromHex(eventContext.Ctx.Params("id"))
			if err != nil {
				return err
			}
			eventContext.ModelID = &id
			return handleEvent(eventContext, loadedModel, string(wst.OperationNameUpdateAttributes))
		}, model.RemoteMethodOptions{
			Name: string(wst.OperationNameUpdateAttributes),
			Accepts: model.RemoteMethodOptionsHttpArgs{
				{
					Arg:         "data",
					Type:        "object",
					Description: "",
					Http:        model.ArgHttp{Source: "body"},
					Required:    true,
				},
			},
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "patch",
			},
		})

		if app.debug {
			log.Println("Mount DELETE " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
			id, err := primitive.ObjectIDFromHex(eventContext.Ctx.Params("id"))
			if err != nil {
				return err
			}
			eventContext.ModelID = &id
			return handleEvent(eventContext, loadedModel, string(wst.OperationNameDeleteById))
		}, model.RemoteMethodOptions{
			Name: string(wst.OperationNameDeleteById),
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "delete",
			},
		})

		if loadedModel.Config.Base == "Account" {
			loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
				id, err := primitive.ObjectIDFromHex(eventContext.Ctx.Params("id"))
				if err != nil {
					return err
				}
				eventContext.ModelID = &id
				return handleEvent(eventContext, loadedModel, string(wst.OperationNameUpsertRoles))
			}, model.RemoteMethodOptions{
				Name: string(wst.OperationNameUpsertRoles),
				Http: model.RemoteMethodOptionsHttp{
					Path: "/:id/roles",
					Verb: "put",
				},
			})
		}
	}
}

func handleEvent(eventContext *model.EventContext, loadedModel *model.StatefulModel, event string) (err error) {
	if loadedModel.DisabledHandlers[event] != true {
		err = loadedModel.GetHandler(event)(eventContext)
		if err != nil {
			return
		}
	}
	if !eventContext.Handled {
		if eventContext.StatusCode == 0 {
			eventContext.StatusCode = fiber.StatusNotImplemented
		}
		err = eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
	}
	return
}

func casbinOwnerFn(loadedModel *model.StatefulModel) func(arguments ...interface{}) (interface{}, error) {
	modelConfigsByName := make(map[string]*model.Config)
	return func(arguments ...interface{}) (interface{}, error) {

		var subId string
		rawToken := arguments[0].(string)
		objId := arguments[1].(string)

		// Decode the token
		token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
			return loadedModel.App.JwtSecretKey, nil
		})
		if err != nil {
			return false, err
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			subId = claims["accountId"].(string)
		} else {
			return false, errors.New("invalid token")
		}

		policyObj := arguments[2]

		if loadedModel.App.Debug {
			log.Println(fmt.Sprintf("isOwner() of %v ?", policyObj))
		}

		objId = strings.TrimSpace(objId)

		if objId == "" || objId == "*" {
			return false, nil
		}

		subId = strings.TrimSpace(subId)

		if subId == "" || subId == "*" {
			return false, nil
		}

		objOwnerId := ""
		if loadedModel.Config.Base == "Account" || loadedModel.Config.Base == "App" {
			objOwnerId = model.GetIDAsString(objId)
			if subId == objOwnerId {
				return true, nil
			}
		}

		roleKey := fmt.Sprintf("%v_OWNERS", objId)
		var accountsForRole []string
		if accountsForRole, err = loadedModel.Enforcer.GetUsersForRole(roleKey); err == nil {
			for _, userInRole := range accountsForRole {
				if subId == userInRole {
					return true, nil
				}
			}
		}

		var recursiveSearchStart time.Time
		if loadedModel.App.Debug {
			recursiveSearchStart = time.Now()
		}
		sortedRelationKeys := obtainSortedRelationKeys(loadedModel, modelConfigsByName)
		for _, relationKey := range sortedRelationKeys {

			err = findOwnerRecursiveInRelation(loadedModel, modelConfigsByName, relationKey, objId, roleKey, &accountsForRole)
			if err != nil {
				if loadedModel.App.Debug {
					loadedModel.App.Logger().Printf("[DEBUG] Recursive owner check for %v[%v]-->%v[%v] failed: %v\n", loadedModel.Name, objId, relationKey, objId, err)
				}
				return false, err
			}

		}
		if loadedModel.App.Debug {
			loadedModel.App.Logger().Printf("[DEBUG] Recursive owner check for %v took %v ms\n", loadedModel.Name, time.Since(recursiveSearchStart).Milliseconds())
		}

		for _, userInRole := range accountsForRole {
			if subId == userInRole {
				return true, nil
			}
		}

		return false, err

	}

}

func obtainSortedRelationKeys(loadedModel *model.StatefulModel, modelConfigsByName map[string]*model.Config) []string {
	allRelatedKeys := make([]string, 0)
	for key, r := range *loadedModel.Config.Relations {
		relatedModelConfig := modelConfigsByName[r.Model]
		if relatedModelConfig == nil {
			// Ignore error because we already checked for it at boot time
			relatedModelI, _ := loadedModel.App.FindModel(r.Model)
			relatedModel := relatedModelI.(*model.StatefulModel)
			relatedModelConfig = relatedModel.Config
			modelConfigsByName[r.Model] = relatedModelConfig
		}
		// userId goes first, others later
		if r.Type == "belongsTo" && (relatedModelConfig.Base == "Account" || relatedModelConfig.Base == "App") {
			allRelatedKeys = append([]string{key}, allRelatedKeys...)
		} else {
			allRelatedKeys = append(allRelatedKeys, key)
		}
	}
	return allRelatedKeys
}

func findOwnerRecursiveInRelation(loadedModel *model.StatefulModel, modelConfigsByName map[string]*model.Config, relationKey string, objId interface{}, roleKey string, ownersForRole *[]string) error {
	r := (*loadedModel.Config.Relations)[relationKey]

	if r.Type == "belongsTo" {

		thisInstance, err := loadedModel.FindById(objId, &wst.Filter{
			Include: &wst.Include{{Relation: relationKey}},
		}, &model.EventContext{
			Bearer: &model.BearerToken{
				Account: &model.BearerAccount{System: true},
			},
		})
		if err != nil {
			return err
		}
		if thisInstance == nil {
			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("[DEBUG] Instance %v[%v] not found\n", loadedModel.Name, objId)
			}
			return wst.CreateError(fiber.ErrNotFound, "NOT_FOUND", fiber.Map{"message": fmt.Sprintf("document %v not found", objId)}, "Error")
		}

		relatedInstance := thisInstance.GetOne(relationKey)
		if relatedInstance == nil {
			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("[DEBUG] Related instance %v[%v]-->%v[%v] unreachable\n", loadedModel.Name, objId, r.Model, thisInstance.ToJSON()[*r.ForeignKey])
			}
			return nil
		}
		relatedModel := relatedInstance.GetModel()

		if relatedModel.GetConfig().Base == "Account" && *r.ForeignKey == "accountId" || relatedModel.GetConfig().Base == "App" && *r.ForeignKey == "appId" {
			user := relatedInstance

			// if user != nil && user.GetID() != nil {
			objOwnerId := model.GetIDAsString(user.GetID())

			_, err := loadedModel.Enforcer.AddRoleForUser(objOwnerId, roleKey)
			if err != nil {
				return err
			}
			err = loadedModel.Enforcer.SavePolicy()
			if err != nil {
				return err
			}

			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("[DEBUG] Added role %v for user %v\n", roleKey, objOwnerId)
			}

			*ownersForRole = append(*ownersForRole, objOwnerId)

			// }

		} else if relatedModel.GetConfig().Base != "Account" && relatedModel.GetConfig().Base != "App" {
			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("[DEBUG] Recursive owner check for %v\n", relatedModel.GetName())
			}
			sortedRelationKeys := obtainSortedRelationKeys(relatedModel.(*model.StatefulModel), modelConfigsByName)
			for _, key := range sortedRelationKeys {
				if loadedModel.App.Debug {
					loadedModel.App.Logger().Printf("[DEBUG] Recursive owner check for %v[%v]-->%v[%v]\n", loadedModel.Name, objId, relatedModel.GetName(), relatedInstance.GetID())
				}
				err = findOwnerRecursiveInRelation(relatedModel.(*model.StatefulModel), modelConfigsByName, key, relatedInstance.GetID(), roleKey, ownersForRole)
				if err != nil {
					return err
				}

			}
		} else {
			fmt.Printf("[WARNING] What to do with %v?", relatedModel)
		}
	}
	return nil
}
