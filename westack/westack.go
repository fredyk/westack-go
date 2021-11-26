package westack

import (
	"encoding/json"
	"fmt"
	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"runtime/debug"

	// docs are generated by Swag CLI, you have to import them.
	// replace with your own docs folder, usually "github.com/username/reponame/docs"
	//_ "github.com/fredyk/westack-go/docs"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
)

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WeStack struct {
	ModelRegistry *map[string]*model.Model
	Datasources   *map[string]*datasource.Datasource
	Server        *fiber.App

	_swaggerPaths map[string]map[string]interface{}
}

func (app WeStack) SwaggerPaths() *map[string]map[string]interface{} {
	return &app._swaggerPaths
}

func (app *WeStack) FindModel(modelName string) *model.Model {
	return (*app.ModelRegistry)[modelName]
}

func (app *WeStack) loadModels() {

	fileInfos, err := ioutil.ReadDir("./common/models")
	if err != nil {
		log.Println("Error while loading models: " + err.Error())
		return
	}

	var globalModelConfig *map[string]*model.Config
	common.LoadFile("./model-config.json", &globalModelConfig)

	app._swaggerPaths = map[string]map[string]interface{}{}
	for _, fileInfo := range fileInfos {
		var config model.Config
		common.LoadFile("./common/models/"+fileInfo.Name(), &config)

		configFromGlobal := (*globalModelConfig)[config.Name]

		if configFromGlobal == nil {
			log.Fatal("ERROR: Missing model ", config.Name, " in model-config.json")
		}

		//noinspection GoUnusedVariable
		dataSource := (*app.Datasources)[configFromGlobal.Datasource]
		loadedModel := model.New(config)
		loadedModel.App = app.AsInterface()
		loadedModel.Datasource = dataSource

		(*app.ModelRegistry)[config.Name] = loadedModel

		if loadedModel.Config.Public {
			var plural string
			if config.Plural != "" {
				plural = config.Plural
			} else {
				plural = common.DashedCase(config.Name) + "s"
			}
			config.Plural = plural
			// TODO: Dynamic rest base
			restApiRoot := "/api/v1"
			modelRouter := app.Server.Group(restApiRoot+"/"+plural, func(ctx *fiber.Ctx) error {
				//log.Println("Resolve " + loadedModel.Name + " " + ctx.Method() + " " + ctx.Path())
				return ctx.Next()
			})
			loadedModel.Router = &modelRouter

			loadedModel.BaseUrl = restApiRoot + "/" + plural
			loadedModel.On("create", func(ctx *model.EventContext) error {

				var data map[string]interface{}
				err := json.Unmarshal(ctx.Ctx.Body(), &data)
				if err != nil {
					return err
				}
				if config.Base == "User" {
					log.Println("Create User")
					hashed, err := bcrypt.GenerateFromPassword([]byte(data["password"].(string)), 10)
					if err != nil {
						return err
					}
					data["password"] = string(hashed)
				}
				created, err := loadedModel.Create(data)
				if err != nil {
					return err
				}
				ctx.Result = created.ToJSON()
				return nil
			})

			if config.Base == "User" {

				loadedModel.On("login", func(ctx *model.EventContext) error {
					var loginBody *LoginBody
					var loginAsMap *map[string]interface{}
					err := json.Unmarshal(ctx.Ctx.Body(), &loginBody)
					err = json.Unmarshal(ctx.Ctx.Body(), &loginAsMap)
					if err != nil {
						ctx.Result = fiber.Map{"error": err}
						return err
					}
					ctx.Data = loginAsMap
					email := loginBody.Email
					users, err := loadedModel.FindMany(&map[string]interface{}{
						"where": map[string]interface{}{
							"email": email,
						},
					})
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

					ctx.Result = fiber.Map{"id": "<token>", "userId": firstUser.Id.(primitive.ObjectID).Hex()}
					return nil
				})

			}

		}
	}
}

func handleEvent(ctx *fiber.Ctx, loadedModel *model.Model, event string) error {
	eventContext := model.EventContext{
		Ctx: ctx,
	}
	err := loadedModel.GetHandler(event)(&eventContext)
	if err != nil {
		return loadedModel.SendError(ctx, err)
	}
	return ctx.JSON(eventContext.Result)
}

func (app *WeStack) loadDataSources() {
	var allDatasources *map[string]*model.DataSourceConfig
	common.LoadFile("./datasources.json", &allDatasources)

	for dsName, dsConfig := range *allDatasources {
		if dsConfig.Connector == "mongodb" {
			ds := datasource.New(map[string]interface{}{
				"name":      dsConfig.Name,
				"connector": dsConfig.Connector,
				"database":  dsConfig.Database,
				"url":       fmt.Sprintf("mongodb://%v:%v/%v", dsConfig.Host, dsConfig.Port, dsConfig.Database),
			})
			err := ds.Initialize()
			if err != nil {
				log.Println(err)
			}
			(*app.Datasources)[dsName] = ds
		} else {
			log.Println("ERROR: connector", dsConfig.Connector, "not supported")
		}
	}
}

func (app *WeStack) Boot(customRoutesCallback func(app *WeStack)) {

	app.loadDataSources()

	app.loadModels()
	app.loadModelsRoutes()

	if customRoutesCallback != nil {
		(customRoutesCallback)(app)
	}

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
			"host":     "127.0.0.1:8023",
			"basePath": "/",
			"paths":    app.SwaggerPaths(),
		})
	})

	app.Server.Get("/swagger/*", swagger.Handler) // default
	//app.Server.Get("/swagger*", swagger.New(swagger.Config{ // custom
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
		(*loadedModel.Router).Use(func(ctx *fiber.Ctx) error {
			log.Println("WARNING: Unresolved method in " + loadedModel.Name + ": " + ctx.Method() + " " + ctx.Path())
			return ctx.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Shared class %#v has no method handling %v %v", loadedModel.Name, ctx.Method(), ctx.Path())}})
			//return ctx.Next()
		})
	}
}

func (app *WeStack) AsInterface() *common.IApp {
	return &common.IApp{
		SwaggerPaths: func() *map[string]map[string]interface{} {
			return app.SwaggerPaths()
		},
	}
}

func (app *WeStack) loadModelsRoutes() {
	for _, entry := range *app.ModelRegistry {
		loadedModel := entry

		log.Println("Mount GET " + loadedModel.BaseUrl)
		loadedModel.RemoteMethod(loadedModel.FindManyRoute, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/",
				Verb: "get",
			},
		})

		log.Println("Mount GET " + loadedModel.BaseUrl + "/:id")
		loadedModel.RemoteMethod(loadedModel.FindByIdRoute, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/:id",
				Verb: "get",
			},
		})

		log.Println("Mount POST " + loadedModel.BaseUrl)
		loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
			return handleEvent(ctx, loadedModel, "create")
		}, model.RemoteMethodOptions{
			Http: model.RemoteMethodOptionsHttp{
				Path: "/",
				Verb: "post",
			},
		})
		if loadedModel.Config.Base == "User" {

			loadedModel.RemoteMethod(func(ctx *fiber.Ctx) error {
				return handleEvent(ctx, loadedModel, "login")
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

func New() WeStack {
	server := fiber.New()

	modelRegistry := make(map[string]*model.Model)
	datasources := make(map[string]*datasource.Datasource)

	app := WeStack{
		ModelRegistry: &modelRegistry,
		Server:        server,
		Datasources:   &datasources,
	}

	// Default middleware config
	server.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(e interface{}) {
			log.Println(e)
			debug.PrintStack()
		},
	}))

	return app
}
