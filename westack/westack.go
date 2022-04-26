package westack

import (
	"context"
	"errors"
	"fmt"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/lib"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WeStack struct {
	ModelRegistry    *map[string]*model.Model
	Datasources      *map[string]*datasource.Datasource
	Server           *fiber.App
	Debug            bool
	RestApiRoot      string
	Port             int32
	RoleModel        *model.Model
	RoleMappingModel *model.Model

	_swaggerPaths map[string]wst.M
	init          time.Time
	JwtSecretKey  []byte
}

func (app WeStack) SwaggerPaths() *map[string]wst.M {
	return &app._swaggerPaths
}

func (app *WeStack) FindModel(modelName string) (*model.Model, error) {
	result := (*app.ModelRegistry)[modelName]
	if result == nil {
		return nil, errors.New(fmt.Sprintf("Model %v not found", modelName))
	}
	return result, nil
}

func (app WeStack) FindDatasource(dsName string) (*datasource.Datasource, error) {
	result := (*app.Datasources)[dsName]

	if result == nil {
		return nil, errors.New(fmt.Sprintf("Datasource %v not found", dsName))
	}

	return result, nil
}

func (app *WeStack) loadModels() {

	fileInfos, err := ioutil.ReadDir("./common/models")
	if err != nil {
		panic("Error while loading models: " + err.Error())
	}

	var globalModelConfig *map[string]*model.SimplifiedConfig
	if err := wst.LoadFile("./server/model-config.json", &globalModelConfig); err != nil {
		panic("Missing or invalid ./server/model-config.json: " + err.Error())
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
		if config.Relations == nil {
			config.Relations = &map[string]*model.Relation{}
		}

		configFromGlobal := (*globalModelConfig)[config.Name]

		if configFromGlobal == nil {
			panic("ERROR: Missing model " + config.Name + " in model-config.json")
		}

		dataSource := (*app.Datasources)[configFromGlobal.Datasource]

		if dataSource == nil {
			panic(fmt.Sprintf("ERROR: Missing or invalid datasource file for %v", dataSource))
		}

		loadedModel := model.New(config, app.ModelRegistry)
		app.setupModel(loadedModel, dataSource)
	}

	for _, loadedModel := range *app.ModelRegistry {
		fixRelations(loadedModel)
	}
}

func (app *WeStack) setupModel(loadedModel *model.Model, dataSource *datasource.Datasource) {

	loadedModel.App = app.AsInterface()
	loadedModel.Datasource = dataSource

	config := loadedModel.Config

	loadedModel.Initialize()

	if config.Base == "Role" {
		app.RoleModel = loadedModel

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
				"user": {
					Type:  "belongsTo",
					Model: "user",
					//PrimaryKey: "",
					//ForeignKey: "",
				},
			},
			Casbin: model.CasbinConfig{
				Policies: []string{
					"$owner,*,__get__role,allow",
				},
			},
		}, app.ModelRegistry)
		roleMappingModel.App = app.AsInterface()
		roleMappingModel.Datasource = dataSource

		app.RoleMappingModel = roleMappingModel
		app.setupModel(roleMappingModel, dataSource)
	}

	if config.Base == "User" {

		loadedModel.On("login", func(ctx *model.EventContext) error {
			data := ctx.Data
			if (*data)["email"] == nil || strings.TrimSpace((*data)["email"].(string)) == "" {
				return ctx.NewError(fiber.ErrBadRequest, "USERNAME_EMAIL_REQUIRED", fiber.Map{"message": "username or email is required"})
			}

			if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
				return ctx.NewError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"})
			}

			email := (*data)["email"].(string)
			users, err := loadedModel.FindMany(&wst.Filter{
				Where: &wst.Where{
					"email": email,
				},
			}, ctx)
			if len(users) == 0 {
				return ctx.NewError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"})
			}
			firstUser := users[0]
			ctx.Instance = &firstUser

			firstUserData := firstUser.ToJSON()
			savedPassword := firstUserData["password"]
			err = bcrypt.CompareHashAndPassword([]byte(savedPassword.(string)), []byte((*data)["password"].(string)))
			if err != nil {
				return ctx.NewError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"})
			}

			userIdHex := firstUser.Id.(primitive.ObjectID).Hex()

			roleNames := []string{"USER"}
			if app.RoleMappingModel != nil {
				ctx.Bearer = &model.BearerToken{
					User: &model.BearerUser{
						System: true,
					},
					Roles: []model.BearerRole{},
				}
				roleContext := &model.EventContext{
					BaseContext:            ctx,
					DisableTypeConversions: true,
				}
				roleEntries, err := app.RoleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
					"principalType": "USER",
					"$or": []wst.M{
						{
							"principalId": userIdHex,
						},
						{
							"principalId": firstUser.Id,
						},
					},
				}, Include: &wst.Include{{Relation: "role"}}}, roleContext)
				if err != nil {
					return err
				}
				for _, roleEntry := range roleEntries {
					role := roleEntry.GetOne("role")
					roleNames = append(roleNames, role.ToJSON()["name"].(string))
				}
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"userId":  userIdHex,
				"created": time.Now().UnixMilli(),
				"ttl":     604800 * 2 * 1000,
				"roles":   roleNames,
			})

			tokenString, err := token.SignedString(loadedModel.App.JwtSecretKey)

			ctx.StatusCode = fiber.StatusOK
			ctx.Result = fiber.Map{"id": tokenString, "userId": userIdHex}
			return nil
		})

	}

	var plural string
	if config.Plural != "" {
		plural = config.Plural
	} else {
		plural = wst.DashedCase(regexp.MustCompile("y$").ReplaceAllString(config.Name, "ie")) + "s"
	}
	config.Plural = plural

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
		"(" +
		"	((p.sub == '$owner' && isOwner(r.sub, r.obj, p.obj)) || g(r.sub, p.sub)) && keyMatch(r.obj, p.obj) && (g(r.act, p.act) || keyMatch(r.act, p.act))" +
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
	casbModel.AddDef("e", "e", replaceVarNames(policyEffect))
	casbModel.AddDef("m", "m", replaceVarNames(matchersDefinition))

	if len(loadedModel.Config.Casbin.Policies) > 0 {
		for _, p := range loadedModel.Config.Casbin.Policies {
			casbModel.AddPolicy("p", "p", []string{replaceVarNames(p)})
		}
	} else {
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,read,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,write,allow")})
	}

	if config.Base == "User" {
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,create,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$everyone,*,login,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$owner,*,*,allow")})
		casbModel.AddPolicy("p", "p", []string{replaceVarNames("$authenticated,*,findSelf,allow")})
	}

	loadedModel.CasbinModel = &casbModel
	loadedModel.CasbinAdapter = &adapter

	err = adapter.SavePolicy(casbModel)
	if err != nil {
		panic(err)
	}
	if loadedModel.Config.Public {

		modelRouter := app.Server.Group(app.RestApiRoot+"/"+plural, func(ctx *fiber.Ctx) error {
			return ctx.Next()
		})
		loadedModel.Router = &modelRouter

		loadedModel.BaseUrl = app.RestApiRoot + "/" + plural

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

		loadedModel.Observe("before save", func(ctx *model.EventContext) error {
			data := ctx.Data

			if (*data)["modified"] == nil {
				timeNow := time.Now()
				(*data)["modified"] = timeNow
			}

			if ctx.IsNewInstance {
				if (*data)["created"] == nil {
					timeNow := time.Now()
					(*data)["created"] = timeNow
				}

				if config.Base == "User" {
					if (*data)["email"] == nil || strings.TrimSpace((*data)["email"].(string)) == "" {
						// TODO: Validate email
						return ctx.NewError(fiber.ErrBadRequest, "EMAIL_PRESENCE", fiber.Map{"message": "Invalid email"})
					}
					filter := wst.Filter{Where: &wst.Where{"email": (*data)["email"]}}
					existent, err2 := loadedModel.FindOne(&filter, ctx)
					if err2 != nil {
						return err2
					}
					if existent != nil {
						return ctx.NewError(fiber.ErrConflict, "EMAIL_UNIQUENESS", fiber.Map{"message": "User exists"})
					}

					if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
						return ctx.NewError(fiber.ErrBadRequest, "PASSWORD_BLANK", fiber.Map{"message": "Invalid password"})
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

			} else {
				if config.Base == "User" {
					if (*data)["password"] != nil && (*data)["password"] != "" {
						log.Println("Update User")
						hashed, err := bcrypt.GenerateFromPassword([]byte((*data)["password"].(string)), 10)
						if err != nil {
							return err
						}
						(*data)["password"] = string(hashed)
					}
				}
			}
			return nil
		})

		loadedModel.On("create", func(ctx *model.EventContext) error {
			created, err := loadedModel.Create(*ctx.Data, ctx)
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
				return ctx.NewError(fiber.ErrBadRequest, "BAD_REQUEST", fiber.Map{"message": fmt.Sprintf("Deleted %v instances for %v", deletedCount, ctx.ModelID)})
			}
			ctx.StatusCode = fiber.StatusNoContent
			ctx.Result = ""
			return nil
		}
		loadedModel.On("instance_delete", deleteByIdHandler)

	}
}

func fixRelations(loadedModel *model.Model) {
	for relationName, relation := range *loadedModel.Config.Relations {
		relatedModelName := relation.Model
		relatedLoadedModel := (*loadedModel.GetModelRegistry())[relatedModelName]

		if relatedLoadedModel == nil {
			log.Println()
			log.Printf("WARNING: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
			log.Println()
			continue
		}

		if relation.PrimaryKey == nil {
			sId := "_id"
			relation.PrimaryKey = &sId
		}

		if relation.ForeignKey == nil {
			switch relation.Type {
			case "belongsTo":
				foreignKey := strings.ToLower(relatedModelName[:1]) + relatedModelName[1:] + "Id"
				relation.ForeignKey = &foreignKey
				//(*loadedModel.Config.Relations)[relationName] = relation
				break
			case "hasOne", "hasMany":
				foreignKey := strings.ToLower(loadedModel.Name[:1]) + loadedModel.Name[1:] + "Id"
				relation.ForeignKey = &foreignKey
				//(*loadedModel.Config.Relations)[relationName] = relation
				break
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
	if loadedModel.DisabledHandlers[event] != true {
		err := loadedModel.GetHandler(event)(eventContext)
		if err != nil {
			return err
		}
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

	dsViper := viper.New()

	fileToLoad := ""

	if env, present := os.LookupEnv("GO_ENV"); present {
		fileToLoad = "datasources." + env
		dsViper.SetConfigName(fileToLoad) // name of config file (without extension)
	} else {
		dsViper.SetConfigName("datasources") // name of config file (without extension)
	}
	dsViper.SetConfigType("json") // REQUIRED if the config file does not have the extension in the name

	dsViper.AddConfigPath("./server") // call multiple times to add many search paths
	dsViper.AddConfigPath(".")        // optionally look for config in the working directory

	err := dsViper.ReadInConfig() // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Println(fmt.Sprintf("WARNING: %v.json not found, fallback to datasources.json", fileToLoad))
			dsViper.SetConfigName("datasources") // name of config file (without extension)
			err := dsViper.ReadInConfig()        // Find and read the config file
			if err != nil {
				panic(fmt.Errorf("fatal error config file: %w", err))
			}
			break
		default:
			panic(fmt.Errorf("fatal error config file: %w", err))
		}
	}

	settings := dsViper.AllSettings()
	ctx := context.Background()
	for key := range settings {
		dsName := dsViper.GetString(key + ".name")
		if dsName == "" {
			dsName = key
		}
		connector := dsViper.GetString(key + ".connector")
		if connector == "mongodb" /* || connector == "memory"*/ || connector == "redis" {
			ds := datasource.New(key, dsViper, ctx)
			err := ds.Initialize()
			if err != nil {
				panic(err)
			}
			(*app.Datasources)[dsName] = ds
			if app.Debug {
				log.Println("Connected to database", dsViper.GetString(key+".database"))
			}
		} else {
			panic("ERROR: connector " + connector + " not supported")
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

		hostname := ctx.Hostname()

		matchedProtocol := "https"

		if strings.Contains(hostname, "localhost") || regexp.MustCompile("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}").MatchString(hostname) {
			matchedProtocol = "http"
		}

		return ctx.JSON(fiber.Map{
			//"schemes": []string{"http"},
			"openapi": "3.0.1",
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
				"version": "3.0",
			},
			"components": fiber.Map{
				"securitySchemes": fiber.Map{
					"bearerAuth": fiber.Map{
						"type":         "http",
						"scheme":       "bearer",
						"bearerFormat": "JWT",
					},
				},
			},
			//"security": fiber.Map{
			//	"bearerAuth": fiber.Map{
			//		"type": "http",
			//		"scheme": "bearer",
			//		"bearerFormat": "JWT",
			//	},
			//},
			"servers": []fiber.Map{
				{
					"url": fmt.Sprintf("%v://%v", matchedProtocol, hostname),
				},
				{
					"url": fmt.Sprintf("http://127.0.0.1:%v", app.Port),
				},
			},
			//"basePath": "/",
			"paths": app.SwaggerPaths(),
		})
	})

	//app.Server.Get("/swagger/*", swagger.New(swagger.Config{
	//	//DeepLinking:       false,
	//	//DocExpansion:      "",
	//	//OAuth:             nil,
	//	//OAuth2RedirectUrl: "",
	//	//URL:               "",
	//})) // default
	app.Server.Get("/swagger/*", func(ctx *fiber.Ctx) error {
		return ctx.Type("html", "utf-8").Send(lib.SwaggerUIStatic)
	}) // default

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
		})
	}
}

func (app *WeStack) AsInterface() *wst.IApp {
	return &wst.IApp{
		Debug:        app.Debug,
		JwtSecretKey: app.JwtSecretKey,
		FindModel: func(modelName string) (interface{}, error) {
			return app.FindModel(modelName)
		},
		FindDatasource: func(datasource string) (interface{}, error) {
			return app.FindDatasource(datasource)
		},
		SwaggerPaths: func() *map[string]wst.M {
			return app.SwaggerPaths()
		},
	}
}

func (app *WeStack) loadModelsFixedRoutes() {
	for _, entry := range *app.ModelRegistry {
		loadedModel := entry

		e, err := casbin.NewEnforcer(*loadedModel.CasbinModel, *loadedModel.CasbinAdapter, true)
		if err != nil {
			panic(err)
		}

		loadedModel.Enforcer = e

		e.EnableAutoSave(true)
		e.AddFunction("isOwner", func(arguments ...interface{}) (interface{}, error) {

			subId := arguments[0]
			objId := arguments[1]
			policyObj := arguments[2]

			if loadedModel.App.Debug {
				log.Println(fmt.Sprintf("isOwner() of %v ?", policyObj))
			}

			switch objId.(type) {
			case primitive.ObjectID:
				objId = objId.(primitive.ObjectID).Hex()
				break
			case string:
				break
			default:
				objId = fmt.Sprintf("%v", objId)
				break
			}
			objId = strings.TrimSpace(objId.(string))

			if objId == "" || objId == "*" {
				return false, nil
			}

			switch subId.(type) {
			case primitive.ObjectID:
				subId = subId.(primitive.ObjectID).Hex()
				break
			case string:
				break
			default:
				subId = fmt.Sprintf("%v", subId)
				break
			}
			subId = strings.TrimSpace(subId.(string))

			if subId == "" || subId == "*" {
				return false, nil
			}

			roleKey := fmt.Sprintf("%v_OWNERS", objId)
			usersForRole, err := loadedModel.Enforcer.GetUsersForRole(roleKey)
			if err != nil {
				return false, err
			}

			if usersForRole == nil || len(usersForRole) == 0 {
				objUserId := ""
				if loadedModel.Config.Base == "User" {
					objUserId = model.GetIDAsString(objId)
					_, err := loadedModel.Enforcer.AddRoleForUser(objUserId, roleKey)
					if err != nil {
						return nil, err
					}

				} else {
					for key, r := range *loadedModel.Config.Relations {

						if *r.ForeignKey == "userId" {

							thisInstance, err := loadedModel.FindById(objId, &wst.Filter{
								Include: &wst.Include{{Relation: key}},
							}, &model.EventContext{
								Bearer: &model.BearerToken{
									User: &model.BearerUser{System: true},
								},
							})
							if err != nil {
								return false, err
							}

							user := thisInstance.GetOne(key)

							if user != nil {
								objUserId = model.GetIDAsString(user.Id)

								_, err := loadedModel.Enforcer.AddRoleForUser(objUserId, roleKey)
								if err != nil {
									return nil, err
								}
								err = loadedModel.Enforcer.SavePolicy()
								if err != nil {
									return nil, err
								}

							}

							break

						}

					}
				}

				usersForRole = append(usersForRole, objUserId)
			}

			for _, userInRole := range usersForRole {
				if subId == userInRole {
					return true, nil
				}
			}

			return false, nil

		})

		_, err = e.AddRoleForUser("findMany", replaceVarNames("read"))
		if err != nil {
			panic(err)
		}
		_, err = e.AddRoleForUser("findById", replaceVarNames("read"))
		if err != nil {
			panic(err)
		}

		_, err = e.AddRoleForUser("create", replaceVarNames("write"))
		if err != nil {
			panic(err)
		}
		_, err = e.AddRoleForUser("instance_updateAttributes", replaceVarNames("write"))
		if err != nil {
			panic(err)
		}
		_, err = e.AddRoleForUser("instance_delete", replaceVarNames("write"))
		if err != nil {
			panic(err)
		}

		_, err = e.AddRoleForUser("read", replaceVarNames("*"))
		if err != nil {
			panic(err)
		}
		_, err = e.AddRoleForUser("write", replaceVarNames("*"))
		if err != nil {
			panic(err)
		}

		err = e.SavePolicy()
		if err != nil {
			panic(err)
		}

		err = e.LoadPolicy()
		if err != nil {
			panic(err)
		}

		if app.Debug {
			loadedModel.CasbinModel.PrintModel()
		}

		text := loadedModel.CasbinModel.ToText()
		err = os.WriteFile(fmt.Sprintf("common/models/%v.casbin.dump.conf", loadedModel.Name), []byte(text), os.ModePerm)
		if err != nil {
			panic(err)
		}
		if !loadedModel.Config.Public {
			if app.Debug {
				log.Println("WARNING: Model", loadedModel.Name, "is not public")
			}
			continue
		}

		if app.Debug {
			log.Println("Mount GET " + loadedModel.BaseUrl)
		}
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
			return handleEvent(eventContext, loadedModel, "findMany")
		}, model.RemoteMethodOptions{
			Name: "findMany",
			Http: model.RemoteMethodOptionsHttp{
				Args: model.RemoteMethodOptionsHttpArgs{
					{
						Name:        "filter",
						Type:        "string",
						Description: "",
						In:          "query",
						Required:    false,
					},
				},
				Path: "/",
				Verb: "get",
			},
		})

		if app.Debug {
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
			Http: model.RemoteMethodOptionsHttp{
				Args: model.RemoteMethodOptionsHttpArgs{
					{
						Name:        "body",
						Type:        "object",
						Description: "",
						In:          "body",
						Required:    true,
					},
				},
				Path: "/",
				Verb: "post",
			},
		})

		if loadedModel.Config.Base == "User" {

			loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
				return handleEvent(eventContext, loadedModel, "login")
			}, model.RemoteMethodOptions{
				Name:        "login",
				Description: "Logins a user",
				Http: model.RemoteMethodOptionsHttp{
					Args: model.RemoteMethodOptionsHttpArgs{
						{
							Name:        "data",
							Type:        "object",
							Description: "",
							In:          "body",
							Required:    false,
						},
					},
					Path: "/login",
					Verb: "post",
				},
			},
			)

			loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

				err, token := eventContext.GetBearer(loadedModel)
				if err != nil {
					return err
				}
				eventContext.ModelID = token.User.Id
				return loadedModel.HandleRemoteMethod("findById", eventContext)

			}, model.RemoteMethodOptions{
				Name:        "findSelf",
				Description: "Find user with their bearer",
				Http: model.RemoteMethodOptionsHttp{
					Args: model.RemoteMethodOptionsHttpArgs{
						{
							Name:        "filter",
							Type:        "string",
							Description: "",
							In:          "query",
							Required:    false,
						},
					},
					Path: "/me",
					Verb: "get",
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

		if app.Debug {
			log.Println("Mount GET " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

			id := eventContext.Ctx.Params("id")
			if eventContext.ModelID == nil {
				eventContext.ModelID = id
			} else if asSt, asStOk := eventContext.ModelID.(string); asStOk && len(strings.TrimSpace(asSt)) == 0 {
				eventContext.ModelID = id
			}

			return handleEvent(eventContext, loadedModel, "findById")

		}, model.RemoteMethodOptions{
			Name: "findById",
			Http: model.RemoteMethodOptionsHttp{
				Args: model.RemoteMethodOptionsHttpArgs{
					{
						Name:        "filter",
						Type:        "string",
						Description: "",
						In:          "query",
						Required:    false,
					},
				},
				Path: "/:id",
				Verb: "get",
			},
		})

		if app.Debug {
			log.Println("Mount PATCH " + loadedModel.BaseUrl + "/:id")
		}
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
			id, err := primitive.ObjectIDFromHex(eventContext.Ctx.Params("id"))
			if err != nil {
				return err
			}
			//var data *wst.M
			//err = json.Unmarshal(eventContext.Ctx.Body(), &data)
			//if err != nil {
			//	return err
			//}
			eventContext.ModelID = &id
			//eventContext.Data = data
			return handleEvent(eventContext, loadedModel, "instance_updateAttributes")
		}, model.RemoteMethodOptions{
			Name: "instance_updateAttributes",
			Http: model.RemoteMethodOptionsHttp{
				Args: model.RemoteMethodOptionsHttpArgs{
					{
						Name:        "data",
						Type:        "object",
						Description: "",
						In:          "body",
						Required:    true,
					},
				},
				Path: "/:id",
				Verb: "patch",
			},
		})

		if app.Debug {
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

type Options struct {
	debug        bool
	RestApiRoot  string
	Port         int32
	jwtSecretKey []byte
}

func New(options Options) *WeStack {
	server := fiber.New()

	modelRegistry := make(map[string]*model.Model)
	datasources := make(map[string]*datasource.Datasource)

	jwtSecretKey := ""
	if s, present := os.LookupEnv("JWT_SECRET"); present {
		jwtSecretKey = s
	}
	_debug := false
	if envDebug, _ := os.LookupEnv("DEBUG"); envDebug == "true" {
		_debug = true
	}

	app := WeStack{
		ModelRegistry: &modelRegistry,
		Server:        server,
		Datasources:   &datasources,
		Debug:         _debug,
		RestApiRoot:   options.RestApiRoot,
		Port:          options.Port,
		JwtSecretKey:  []byte(jwtSecretKey),

		init: time.Now(),
	}

	server.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Println(e)
			debug.PrintStack()
		},
	}))

	return &app
}
