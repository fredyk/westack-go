package westack

import (
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

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
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
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		return handleEvent(eventContext, loadedModel, "login")
	}, model.RemoteMethodOptions{
		Name:        "login",
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
		return loadedModel.HandleRemoteMethod("findById", eventContext)

	}, model.RemoteMethodOptions{
		Name:        "findSelf",
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
					fmt.Printf("[DEBUG] Invalid token: %s\n", err.Error())

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
		Name:        "refreshToken",
		Description: "Obtains current user",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/refresh-token",
			Verb: "post",
		},
	})

}

func mountBaseModelFixedRoutes(app *WeStack, loadedModel *model.StatefulModel) {
	if app.debug {
		log.Println("Mount GET " + loadedModel.BaseUrl)
	}
	loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
		return handleEvent(eventContext, loadedModel, "findMany")
	}, model.RemoteMethodOptions{
		Name: "findMany",
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
		return handleEvent(eventContext, loadedModel, "count")
	}, model.RemoteMethodOptions{
		Name: "count",
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
		//var data *wst.M
		//err := json.Unmarshal(eventContext.Ctx.Body(), &data)
		//if err != nil {
		//	return err
		//}
		//eventContext.Data = data
		return handleEvent(eventContext, loadedModel, "create")
	}, model.RemoteMethodOptions{
		Name: "create",
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
		Name: "createToken",
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

			return handleEvent(eventContext, loadedModel, "findById")

		}, model.RemoteMethodOptions{
			Name: "findById",
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
			return handleEvent(eventContext, loadedModel, "instance_updateAttributes")
		}, model.RemoteMethodOptions{
			Name: "instance_updateAttributes",
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
			return handleEvent(eventContext, loadedModel, "instance_delete")
		}, model.RemoteMethodOptions{
			Name: "instance_delete",
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
				return handleEvent(eventContext, loadedModel, "user_upsertRoles")
			}, model.RemoteMethodOptions{
				Name: "user_upsertRoles",
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

		subId := arguments[0].(string)
		objId := arguments[1].(string)
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

		objUserId := ""
		if loadedModel.Config.Base == "Account" || loadedModel.Config.Base == "App" {
			objUserId = model.GetIDAsString(objId)
			if subId == objUserId {
				return true, nil
			}
		}

		roleKey := fmt.Sprintf("%v_OWNERS", objId)
		var accountsForRole []string
		var err error
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
			objUserId := model.GetIDAsString(user.GetID())

			_, err := loadedModel.Enforcer.AddRoleForUser(objUserId, roleKey)
			if err != nil {
				return err
			}
			err = loadedModel.Enforcer.SavePolicy()
			if err != nil {
				return err
			}

			if loadedModel.App.Debug {
				loadedModel.App.Logger().Printf("[DEBUG] Added role %v for user %v\n", roleKey, objUserId)
			}

			*ownersForRole = append(*ownersForRole, objUserId)

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
