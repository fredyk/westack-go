package westack

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/lib"
	"github.com/fredyk/westack-go/westack/model"
)

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WeStack struct {
	Server *fiber.App

	port              int32
	datasources       *map[string]*datasource.Datasource
	modelRegistry     *map[string]*model.Model
	debug             bool
	restApiRoot       string
	roleMappingModel  *model.Model
	dataSourceOptions *map[string]*datasource.Options
	_swaggerPaths     map[string]wst.M
	init              time.Time
	jwtSecretKey      []byte
	viper             *viper.Viper
}

func (app *WeStack) FindModel(modelName string) (*model.Model, error) {
	result := (*app.modelRegistry)[modelName]
	if result == nil {
		return nil, errors.New(fmt.Sprintf("Model %v not found", modelName))
	}
	return result, nil
}

func (app *WeStack) FindDatasource(dsName string) (*datasource.Datasource, error) {
	result := (*app.datasources)[dsName]

	if result == nil {
		return nil, errors.New(fmt.Sprintf("Datasource %v not found", dsName))
	}

	return result, nil
}

func (app *WeStack) Boot(customRoutesCallbacks ...func(app *WeStack)) {

	app.loadDataSources()

	app.loadModels()

	app.Middleware(func(c *fiber.Ctx) error {
		method := c.Method()
		err := c.Next()
		if err != nil {
			log.Println("Error:", err)
			log.Printf("%v: %v\n", method, c.OriginalURL())
			switch err.(type) {
			case *fiber.Error:
				if err.(*fiber.Error).Code == fiber.StatusNotFound {
					return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": fiber.Map{"status": err.(*fiber.Error).Code, "message": fmt.Sprintf("Unknown method %v %v", method, c.Path())}})
				} else {
					return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": fiber.Map{"status": err.(*fiber.Error).Code, "message": err.(*fiber.Error).Message}})
				}
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"status": fiber.StatusInternalServerError, "message": err.Error()}})
			}
		}
		return nil
	})

	app.Middleware(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Println(e)
			debug.PrintStack()
		},
	}))

	app.loadModelsFixedRoutes()

	for _, cb := range customRoutesCallbacks {
		cb(app)
	}

	app.loadModelsDynamicRoutes()
	app.loadNotFoundRoutes()

	app.Server.Get("/swagger/doc.json", swaggerDocsHandler(app))

	app.Server.Get("/swagger/*", func(ctx *fiber.Ctx) error {
		return ctx.Type("html", "utf-8").Send(lib.SwaggerUIStatic)
	})

}

func (app *WeStack) Start() interface{} {
	log.Printf("DEBUG Server took %v ms to start\n", time.Now().UnixMilli()-app.init.UnixMilli())
	return app.Server.Listen(fmt.Sprintf("0.0.0.0:%v", app.port))
}

func (app *WeStack) Middleware(handler fiber.Handler) {
	app.Server.Use(handler)
}

func GRPCCallWithQueryParams[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return gRPCCallWithQueryParams(serviceUrl, clientConstructor, clientMethod)
}

func GRPCCallWithBody[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return gRPCCallWithBody(serviceUrl, clientConstructor, clientMethod)
}

type Options struct {
	RestApiRoot       string
	Port              int32
	JwtSecretKey      string
	DatasourceOptions *map[string]*datasource.Options

	debug bool
}

func New(options ...Options) *WeStack {
	server := fiber.New()

	modelRegistry := make(map[string]*model.Model)
	datasources := make(map[string]*datasource.Datasource)

	var finalOptions Options
	if len(options) > 0 {
		finalOptions = options[0]
	}
	if finalOptions.JwtSecretKey == "" {
		if s, present := os.LookupEnv("JWT_SECRET"); present {
			finalOptions.JwtSecretKey = s
		}
	}
	_debug := false
	if envDebug, _ := os.LookupEnv("DEBUG"); envDebug == "true" {
		_debug = true
	}

	appViper := viper.New()

	fileToLoad := ""

	if env, present := os.LookupEnv("GO_ENV"); present {
		fileToLoad = "config." + env
		appViper.SetConfigName(fileToLoad) // name of config file (without extension)
	} else {
		appViper.SetConfigName("config") // name of config file (without extension)
	}
	appViper.SetConfigType("json") // REQUIRED if the config file does not have the extension in the name

	appViper.AddConfigPath("./server") // call multiple times to add many search paths
	appViper.AddConfigPath(".")           // for unit tests     

	err := appViper.ReadInConfig() // Find and read the config file
	if err != nil {                // Handle errors reading the config file
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Println(fmt.Sprintf("WARNING: %v.json not found, fallback to config.json", fileToLoad))
			appViper.SetConfigName("config") // name of config file (without extension)
			err := appViper.ReadInConfig()   // Find and read the config file
			if err != nil {
				panic(fmt.Errorf("fatal error config file: %w", err))
			}
			break
		default:
			panic(fmt.Errorf("fatal error config file: %w", err))
		}
	}

	if finalOptions.RestApiRoot == "" {
		finalOptions.RestApiRoot = appViper.GetString("restApiRoot")
	}
	if finalOptions.Port == 0 {
		finalOptions.Port = appViper.GetInt32("port")
	}
	app := WeStack{
		Server: server,

		modelRegistry:     &modelRegistry,
		datasources:       &datasources,
		debug:             _debug,
		restApiRoot:       finalOptions.RestApiRoot,
		port:              finalOptions.Port,
		jwtSecretKey:      []byte(finalOptions.JwtSecretKey),
		dataSourceOptions: finalOptions.DatasourceOptions,
		init:              time.Now(),
		viper:             appViper,
	}

	return &app
}

func InitAndServe() {
	app := New()

	app.Boot()

	log.Fatal(app.Start())
}
