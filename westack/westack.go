package westack

import (
	"encoding/json"
	"fmt"
	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt"
	"os"
	"regexp"

	//"go.mongodb.org/mongo-driver/bson"
	"runtime/debug"
	"strings"
	"time"

	// docs are generated by Swag CLI, you have to import them.
	// replace with your own docs folder, usually "github.com/username/reponame/docs"
	//_ "github.com/fredyk/westack-go/docs"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
)

// For HMAC signing method, the key can be any []byte. It is recommended to generate
// a key using crypto/rand or something equivalent. You need the same key for signing
// and validating.
var hmacSampleSecret []byte

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WeStack struct {
	ModelRegistry *map[string]*model.Model
	Datasources   *map[string]*datasource.Datasource
	Server        *fiber.App
	Debug         bool
	RestApiRoot   string
	Port          int32

	_swaggerPaths map[string]wst.M
	init          time.Time
}

func (app WeStack) SwaggerPaths() *map[string]wst.M {
	return &app._swaggerPaths
}

func (app *WeStack) FindModel(modelName string) *model.Model {
	result := (*app.ModelRegistry)[modelName]
	if result == nil {
		panic(fmt.Sprintf("Model %v not found", modelName))
	}
	return result
}

func IsOwnerFunc(args ...interface{}) (interface{}, error) {
	//log.Println(fmt.Sprintf("DEBUG: Check %v <--> %v", args[0], args[1]))
	requestSubj := args[0]
	switch requestSubj.(type) {
	case primitive.ObjectID:
		requestSubj = requestSubj.(primitive.ObjectID).Hex()
		break
	default:
		requestSubj = fmt.Sprintf("%v", requestSubj)
		break
	}

	if asMap, asMapOk := args[1].(map[string]interface{}); asMapOk {
		userId := asMap["userId"]
		if userId != nil {
			switch userId.(type) {
			case primitive.ObjectID:
				userId = userId.(primitive.ObjectID).Hex()
				break
			default:
				userId = fmt.Sprintf("%v", userId)
				break
			}
			//log.Println(fmt.Sprintf("DEBUG: Check %v <--> %v", requestSubj, userId))
			return requestSubj.(string) == userId.(string), nil
		}
	}

	return false, nil
}

func (app *WeStack) loadModels() {

	fileInfos, err := ioutil.ReadDir("./common/models")
	if err != nil {
		panic("Error while loading models: " + err.Error())
	}

	var globalModelConfig *map[string]*model.Config
	if err := wst.LoadFile("./model-config.json", &globalModelConfig); err != nil {
		panic("Missing or invalid ./model-config.json: " + err.Error())
	}

	app._swaggerPaths = map[string]wst.M{}
	for _, fileInfo := range fileInfos {

		if strings.Split(fileInfo.Name(), ".")[1] != "json" {
			continue
		}
		var config *model.Config
		err := wst.LoadFile("./common/models/"+fileInfo.Name(), &config)
		if err != nil {
			panic(err)
		}

		configFromGlobal := (*globalModelConfig)[config.Name]

		if configFromGlobal == nil {
			panic("ERROR: Missing model " + config.Name + " in model-config.json")
		}

		//noinspection GoUnusedVariable
		dataSource := (*app.Datasources)[configFromGlobal.Datasource]

		if dataSource == nil {
			panic(fmt.Sprintf("ERROR: Missing or invalid datasource file for %v", dataSource))
		}

		loadedModel := model.New(config, app.ModelRegistry)
		loadedModel.App = app.AsInterface()
		loadedModel.Datasource = dataSource

		if loadedModel.Config.Public {
			var plural string
			if config.Plural != "" {
				plural = config.Plural
			} else {
				plural = wst.DashedCase(config.Name) + "s"
			}
			config.Plural = plural

			modelRouter := app.Server.Group(app.RestApiRoot+"/"+plural, func(ctx *fiber.Ctx) error {
				//log.Println("Resolve " + loadedModel.Name + " " + ctx.Method() + " " + ctx.Path())
				return ctx.Next()
			})
			loadedModel.Router = &modelRouter

			loadedModel.BaseUrl = app.RestApiRoot + "/" + plural

			if len(loadedModel.Config.Acls) == 0 {
				loadedModel.Config.Acls = append(loadedModel.Config.Acls, model.ACL{
					AccessType:    "*",
					PrincipalType: "ROLE",
					PrincipalId:   "$everyone",
					Permission:    "DENY",
					Property:      "",
				}, model.ACL{
					AccessType:    "*",
					PrincipalType: "ROLE",
					PrincipalId:   "$authenticated",
					Permission:    "ALLOW",
					Property:      "",
				})
			}

			casbModel := casbinmodel.NewModel()

			f, err := os.OpenFile(fmt.Sprintf("common/models/%v.policies.csv", loadedModel.Name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(err)
			}
			err = f.Close()
			if err != nil {
				panic(err)
			}

			adapter := fileadapter.NewAdapter(fmt.Sprintf("common/models/%v.policies.csv", loadedModel.Name))

			requestDefinition := "sub, obj, act"
			policyDefinition := "sub, obj, act, eft"
			roleDefinition := "_, _"
			policyEffect := "subjectPriority(p.eft) || deny"
			matchersDefinition := fmt.Sprintf("" +
				"(p.sub == '$owner' && isOwner(r.sub, r.obj)) || " +
				"(" +
				"	g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)" +
				")")
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
			//casbModel.AddDef("e", "e", "subjectPriority(p.eft) || deny")
			casbModel.AddDef("e", "e", replaceVarNames(policyEffect))
			casbModel.AddDef("m", "m", replaceVarNames(matchersDefinition))
			//casbModel.AddDef("m", "m2", "r_sub == 'bob'")

			//err := casbModel.LoadModel("basic_model.conf")
			//if err != nil {
			//	panic(err)
			//}

			//policy, err := casbModel.AddPolicy("p", "alice,password,read")
			if len(loadedModel.Config.Casbin.Policies) > 0 {
				for _, p := range loadedModel.Config.Casbin.Policies {
					casbModel.AddPolicy("p", "p", []string{replaceVarNames(p)})
				}
			} else {
				casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,read,allow")})
				casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,write,allow")})
			}

			if config.Base == "User" {
				casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,create,*,allow")})
				casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,login,*,allow")})
				casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,*,allow")})
			}

			//casbModel.GetLogger().EnableLog(true)
			loadedModel.CasbinModel = &casbModel
			loadedModel.CasbinAdapter = &adapter

			err = adapter.SavePolicy(casbModel)
			if err != nil {
				panic(err)
			}

			loadedModel.On("findMany", func(ctx *model.EventContext) error {
				result, err := loadedModel.FindMany(ctx.Filter, ctx)
				out := make(wst.A, len(result))
				for idx, item := range result {
					item.HideProperties()
					out[idx] = item.ToJSON()
				}

				if err != nil {
					return err
				}
				ctx.StatusCode = fiber.StatusOK
				ctx.Result = out
				return nil
			})
			loadedModel.On("findById", func(ctx *model.EventContext) error {
				result, err := loadedModel.FindById(ctx.ModelID, ctx.Filter, ctx)
				if result != nil {
					result.HideProperties()
				}
				if err != nil {
					return err
				}
				ctx.StatusCode = fiber.StatusOK
				ctx.Result = result.ToJSON()
				return nil
			})
			loadedModel.On("create", func(ctx *model.EventContext) error {

				data := ctx.Data
				if config.Base == "User" {

					if (*data)["email"] == nil || strings.TrimSpace((*data)["email"].(string)) == "" {
						// TODO: Validate email
						return ctx.RestError(fiber.ErrBadRequest, fiber.Map{"error": "Invalid email"})
					}
					filter := wst.Filter{Where: &wst.Where{"email": (*data)["email"]}}
					existent, err2 := loadedModel.FindOne(&filter, ctx)
					if err2 != nil {
						return err2
					}
					if existent != nil {
						return ctx.RestError(fiber.ErrConflict, fiber.Map{"error": "User exists"})
					}

					if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
						return ctx.RestError(fiber.ErrBadRequest, fiber.Map{"error": "Invalid password"})
					}
					hashed, err := bcrypt.GenerateFromPassword([]byte((*data)["password"].(string)), 10)
					if err != nil {
						return err
					}
					(*data)["password"] = string(hashed)

					if app.Debug {
						log.Println("Create User")
					}
				}
				created, err := loadedModel.Create(*data, ctx)
				if err != nil {
					return err
				}
				ctx.StatusCode = fiber.StatusOK
				ctx.Result = created.ToJSON()
				return nil
			})

			loadedModel.On("instance_updateAttributes", func(ctx *model.EventContext) error {

				inst, err := loadedModel.FindById(ctx.ModelID, nil, ctx)
				if err != nil {
					return err
				}

				if config.Base == "User" && (*ctx.Data)["password"] != nil && (*ctx.Data)["password"] != "" {
					log.Println("Update User")
					hashed, err := bcrypt.GenerateFromPassword([]byte((*ctx.Data)["password"].(string)), 10)
					if err != nil {
						return err
					}
					(*ctx.Data)["password"] = string(hashed)
				}
				updated, err := inst.UpdateAttributes(ctx.Data, ctx)
				if err != nil {
					return err
				}
				ctx.StatusCode = fiber.StatusOK
				ctx.Result = updated.ToJSON()
				return nil
			})

			deleteByIdHandler := func(ctx *model.EventContext) error {
				deletedCount, err := loadedModel.DeleteById(ctx.ModelID)
				if err != nil {
					return err
				}
				if deletedCount != 1 {
					return ctx.RestError(fiber.ErrBadRequest, fiber.Map{"error": fmt.Sprintf("Deleted %v instances for %v", deletedCount, ctx.ModelID)})
				}
				ctx.StatusCode = fiber.StatusNoContent
				ctx.Result = ""
				return nil
			}
			loadedModel.On("deleteById", deleteByIdHandler)

			if config.Base == "User" {

				loadedModel.On("login", func(ctx *model.EventContext) error {
					var loginBody *LoginBody
					var data *wst.M
					err := json.Unmarshal(ctx.Ctx.Body(), &loginBody)
					err = json.Unmarshal(ctx.Ctx.Body(), &data)
					if err != nil {
						ctx.StatusCode = fiber.StatusBadRequest
						ctx.Result = fiber.Map{"error": err}
						return err
					}
					ctx.Data = data

					if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
						return ctx.RestError(fiber.ErrBadRequest, fiber.Map{"error": "Invalid password"})
					}

					email := loginBody.Email
					users, err := loadedModel.FindMany(&wst.Filter{
						Where: &wst.Where{
							"email": email,
						},
					}, ctx)
					if len(users) == 0 {
						return ctx.RestError(fiber.ErrNotFound, fiber.Map{"error": "User not found"})
					}
					firstUser := users[0]
					ctx.Instance = &firstUser

					firstUserData := firstUser.ToJSON()
					savedPassword := firstUserData["password"]
					err = bcrypt.CompareHashAndPassword([]byte(savedPassword.(string)), []byte(loginBody.Password))
					if err != nil {
						return ctx.RestError(fiber.ErrBadRequest, fiber.Map{"error": "Invalid credentials"})
					}

					userIdHex := firstUser.Id.(primitive.ObjectID).Hex()

					// Create a new token object, specifying signing method and the claims
					// you would like it to contain.
					token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
						"userId":  userIdHex,
						"created": time.Now().UnixMilli(),
					})

					// Sign and get the complete encoded token as a string using the secret
					tokenString, err := token.SignedString(hmacSampleSecret)

					ctx.StatusCode = fiber.StatusOK
					ctx.Result = fiber.Map{"id": tokenString, "userId": userIdHex}
					return nil
				})

			}

		}
	}
}

func replaceVarNames(definition string) string {
	return regexp.MustCompile("\\$(\\w+)").ReplaceAllStringFunc(definition, func(match string) string {
		return "_" + strings.ToUpper(match[1:]) + "_"
	})
}

func handleEvent(eventContext *model.EventContext, loadedModel *model.Model, event string) error {
	err := loadedModel.GetHandler(event)(eventContext)
	if err != nil {
		return loadedModel.SendError(eventContext.Ctx, err)
	}
	if eventContext.StatusCode == 0 {
		eventContext.StatusCode = fiber.StatusNotImplemented
	}
	if eventContext.Ephemeral != nil {
		for k, v := range *eventContext.Ephemeral {
			(eventContext.Result.(wst.M))[k] = v
		}
	}
	return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
}

func (app *WeStack) loadDataSources() {
	var allDatasources *map[string]*model.DataSourceConfig
	if err := wst.LoadFile("./datasources.json", &allDatasources); err != nil {
		panic(err)
	}

	for key, dsConfig := range *allDatasources {
		dsName := dsConfig.Name
		if dsName == "" {
			dsName = key
		}
		if dsConfig.Connector == "mongodb" {
			ds := datasource.New(wst.M{
				"name":      dsConfig.Name,
				"connector": dsConfig.Connector,
				"database":  dsConfig.Database,
				"url":       fmt.Sprintf("mongodb://%v:%v/%v", dsConfig.Host, dsConfig.Port, dsConfig.Database),
			})
			err := ds.Initialize()
			if err != nil {
				panic(err)
			}
			(*app.Datasources)[dsName] = ds
			if app.Debug {
				log.Println("Connected to database", dsConfig.Database)
			}
		} else {
			panic("ERROR: connector " + dsConfig.Connector + " not supported")
		}
	}
}

func (app *WeStack) Boot(customRoutesCallback func(app *WeStack)) {

	app.loadDataSources()

	app.loadModels()
	app.loadModelsFixedRoutes()

	if customRoutesCallback != nil {
		(customRoutesCallback)(app)
	}

	app.loadModelsDynamicRoutes()
	app.loadNotFoundRoutes()

	app.Server.Get("/swagger/doc.json", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"schemes": []string{"http"},
			"swagger": "2.0",
			"info": fiber.Map{
				"description":    "This is your go-based API Server.",
				"title":          "Swagger API",
				"termsOfService": "http://swagger.io/terms/",
				"contact": fiber.Map{
					"name":  "API Support",
					"url":   "http://www.swagger.io/support",
					"email": "support@swagger.io",
				},
				"license": fiber.Map{
					"name": "Apache 2.0",
					"url":  "http://www.apache.org/licenses/LICENSE-2.0.html",
				},
				"version": "2.0",
			},
			"host":     fmt.Sprintf("127.0.0.1:%v", app.Port),
			"basePath": "/",
			"paths":    app.SwaggerPaths(),
		})
	})

	app.Server.Get("/swagger/*", swagger.Handler) // default
	//app.Server.Get("/swagger*", swagger.NewModel(swagger.ModelConfig{ // custom
	//	URL: "http://localhost:8023/swagger.json",
	//	DeepLinking: false,
	//	// Expand ("list") or Collapse ("none") tag groups by default
	//	DocExpansion: "none",
	//	// Prefill OAuth ClientId on Authorize popup
	//	OAuth: &swagger.OAuthConfig{
	//		AppName:  "OAuth Provider",
	//		ClientId: "21bb4edc-05a7-4afc-86f1-2e151e4ba6e2",
	//	},
	//	// Ability to change OAuth2 redirect uri location
	//	OAuth2RedirectUrl: "http://localhost:8080/swagger/oauth2-redirect.html",
	//}))

}

func (app *WeStack) loadNotFoundRoutes() {
	for _, entry := range *app.ModelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.Debug {
				log.Println("WARNING: Model", loadedModel.Name, "is not public")
			}
			continue
		}
		(*loadedModel.Router).Use(func(ctx *fiber.Ctx) error {
			log.Println("WARNING: Unresolved method in " + loadedModel.Name + ": " + ctx.Method() + " " + ctx.Path())
			return ctx.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Shared class %#v has no method handling %v %v", loadedModel.Name, ctx.Method(), ctx.Path())}})
			//return ctx.Next()
		})
	}
}

func (app *WeStack) AsInterface() *wst.IApp {
	return &wst.IApp{
		Debug: app.Debug,
		FindModel: func(modelName string) interface{} {
			return app.FindModel(modelName)
		},
		SwaggerPaths: func() *map[string]wst.M {
			return app.SwaggerPaths()
		},
	}
}

func (app *WeStack) loadModelsFixedRoutes() {
	for _, entry := range *app.ModelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.Debug {
				log.Println("WARNING: Model", loadedModel.Name, "is not public")
			}
			continue
		}

		if app.Debug {
			log.Println("Mount GET " + loadedModel.BaseUrl)
		}
		loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
			filterSt := ctx.Query("filter")
			filterMap := model.ParseFilter(filterSt)

			eventContext := model.EventContext{
				Filter: filterMap,
				Ctx:    ctx,
			}
			return handleEvent(&eventContext, loadedModel, "findMany")
		}, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/",
				Verb: "get",
			},
		})

		if app.Debug {
			log.Println("Mount POST " + loadedModel.BaseUrl)
		}
		loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
			var data *wst.M
			err := json.Unmarshal(ctx.Body(), &data)
			if err != nil {
				return err
			}
			eventContext := model.EventContext{
				Ctx:  ctx,
				Data: data,
			}
			return handleEvent(&eventContext, loadedModel, "create")
		}, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/",
				Verb: "post",
			},
		})

		if loadedModel.Config.Base == "User" {

			loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
				eventContext := model.EventContext{
					Ctx: ctx,
				}
				return handleEvent(&eventContext, loadedModel, "login")
			}, model.RemoteMethodOptions{
				Description: "Logins a user",
				Http: model.RemoteMethodOptionsHttp{
					Path: "/login",
					Verb: "post",
				},
			},
			)

		}

	}
}

func (app *WeStack) loadModelsDynamicRoutes() {
	for _, entry := range *app.ModelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.Debug {
				log.Println("WARNING: Model", loadedModel.Name, "is not public")
			}
			continue
		}

		e, err := casbin.NewEnforcer(*loadedModel.CasbinModel, *loadedModel.CasbinAdapter, true)
		if err != nil {
			panic(err)
		}

		e.EnableAutoSave(true)
		e.AddFunction("isOwner", IsOwnerFunc)

		//user, err := e.AddRoleForUser("alice", replaceVarNames("$authenticated"))
		//if err != nil {
		//	panic(err)
		//}
		//log.Println("added role", user)

		//err = adapter.SavePolicy(casbModel)
		//if err != nil {
		//	panic(err)
		//}

		err = e.SavePolicy()
		if err != nil {
			panic(err)
		}

		err = e.LoadPolicy()
		if err != nil {
			panic(err)
		}

		//casbModel.AddPolicy()
		if app.Debug {
			loadedModel.CasbinModel.PrintModel()
		}

		text := loadedModel.CasbinModel.ToText()
		err = os.WriteFile(fmt.Sprintf("common/models/%v.casbin.dump.conf", loadedModel.Name), []byte(text), os.ModePerm)
		if err != nil {
			panic(err)
		}

		//aliceRoles, err := e.GetImplicitRolesForUser("alice")
		//if err != nil {
		//	panic(err)
		//}
		//log.Println("aliceRoles", aliceRoles)
		//
		//bobRoles, err := e.GetImplicitRolesForUser("bob")
		//if err != nil {
		//	panic(err)
		//}
		//log.Println("bobRoles", bobRoles)

		//		if res, err := e.Enforce(/*casbin.EnforceContext{
		//			RType: "r",
		//			PType: "p",
		//			EType: "e",
		//			MType: "m",
		//		},*/ "alice", "photo", "read"); res {
		//			// permit alice to read data1
		//			log.Println(fmt.Sprintf("allow access on %v", loadedModel.Name))
		//		} else {
		//			if err != nil {
		//				// deny the request, show an error
		//				panic(err)
		//			} else {
		//				log.Println(fmt.Sprintf("deny access on %v", loadedModel.Name))
		//			}
		//		}
		//
		//		if res, err := e.Enforce(
		///*			casbin.EnforceContext{
		//				RType: "r",
		//				PType: "p",
		//				EType: "e",
		//				MType: "m",
		//			},*/
		//			"bob", "photo", "read"); res {
		//			// permit alice to read data1
		//			log.Println(fmt.Sprintf("allow access on %v", loadedModel.Name))
		//		} else {
		//			if err != nil {
		//				// deny the request, show an error
		//				panic(err)
		//			} else {
		//				log.Println(fmt.Sprintf("deny access on %v", loadedModel.Name))
		//			}
		//		}
		//
		//		if res, err := e.Enforce(/*casbin.EnforceContext{
		//			RType: "r",
		//			PType: "p",
		//			EType: "e",
		//			MType: "m",
		//		},*/ "alice", "photo", "write"); res {
		//			// permit alice to read data1
		//			log.Println(fmt.Sprintf("allow access on %v", loadedModel.Name))
		//		} else {
		//			if err != nil {
		//				// deny the request, show an error
		//				panic(err)
		//			} else {
		//				log.Println(fmt.Sprintf("deny access on %v", loadedModel.Name))
		//			}
		//		}
		//
		//		if res, err := e.Enforce(/*casbin.EnforceContext{
		//			RType: "r",
		//			PType: "p",
		//			EType: "e",
		//			MType: "m",
		//		},*/ "bob", "photo", "write"); res {
		//			// permit alice to read data1
		//			log.Println("allow access \"bob\", \"photo\", \"write\"")
		//		} else {
		//			if err != nil {
		//				// deny the request, show an error
		//				panic(err)
		//			} else {
		//				log.Println("deny access \"bob\", \"photo\", \"write\"")
		//			}
		//		}
		//
		//		if res, err := e.Enforce(/*casbin.EnforceContext{
		//			RType: "r",
		//			PType: "p",
		//			EType: "e",
		//			MType: "m",
		//		},*/ "000000000000000000000000", map[string]interface{}{"photo": "a", "userId": "000000000000000000000000"}, "write"); res {
		//			// permit alice to read data1
		//			log.Println("allow access \"000000000000000000000000\", \"photo\", \"write\"")
		//		} else {
		//			if err != nil {
		//				// deny the request, show an error
		//				panic(err)
		//			} else {
		//				log.Println("deny access \"000000000000000000000000\", \"photo\", \"write\"")
		//			}
		//		}

		//authz := fibercasbin.New(fibercasbin.Config{
		//	ModelFilePath: "path/to/rbac_model.conf",
		//	PolicyAdapter: xormadapter.NewAdapter("mysql", "root:@tcp(127.0.0.1:3306)/"),
		//	Lookup: func(c *fiber.Ctx) string {
		//		// fetch authenticated user subject
		//	},
		//})
		//
		//modelRouter.Use(authz.)

		if app.Debug {
			log.Println("Mount GET " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
			id := ctx.Params("id")
			filterSt := ctx.Query("filter")
			filterMap := model.ParseFilter(filterSt)

			eventContext := model.EventContext{
				ModelID: id,
				Filter:  filterMap,
				Ctx:     ctx,
			}
			return handleEvent(&eventContext, loadedModel, "findById")

		}, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "get",
			},
		})

		if app.Debug {
			log.Println("Mount PATCH " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
			id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
			if err != nil {
				return err
			}
			var data *wst.M
			err = json.Unmarshal(ctx.Body(), &data)
			if err != nil {
				return err
			}
			eventContext := model.EventContext{
				Ctx:     ctx,
				ModelID: &id,
				Data:    data,
			}
			return handleEvent(&eventContext, loadedModel, "instance_updateAttributes")
		}, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "patch",
			},
		})

		if app.Debug {
			log.Println("Mount DELETE " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
			id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
			if err != nil {
				return err
			}
			eventContext := model.EventContext{
				Ctx:     ctx,
				ModelID: &id,
			}
			return handleEvent(&eventContext, loadedModel, "deleteById")
		}, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "delete",
			},
		})
	}
}

// Listen is an alias for Start()
//
// Deprecated: Start() should be used instead
func (app WeStack) Listen(addr string) interface{} {
	return app.Start(addr)
}

func (app WeStack) Start(addr string) interface{} {
	log.Printf("DEBUG Server took %v ms to start\n", time.Now().UnixMilli()-app.init.UnixMilli())
	return app.Server.Listen(addr)
}

type WeStackOptions struct {
	Debug       bool
	RestApiRoot string
	Port        int32
}

func New(options WeStackOptions) *WeStack {
	server := fiber.New()

	modelRegistry := make(map[string]*model.Model)
	datasources := make(map[string]*datasource.Datasource)

	app := WeStack{
		ModelRegistry: &modelRegistry,
		Server:        server,
		Datasources:   &datasources,
		Debug:         options.Debug,
		RestApiRoot:   options.RestApiRoot,
		Port:          options.Port,

		init: time.Now(),
	}

	// Default middleware config
	server.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Println(e)
			debug.PrintStack()
		},
	}))

	return &app
}
