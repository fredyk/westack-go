package westack

import (
	"context"
	"errors"
	"fmt"
	"github.com/fredyk/westack-go/westack/lib/swaggerhelperinterface"
	"github.com/fredyk/westack-go/westack/memorykv"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"

	"github.com/goccy/go-json"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/fredyk/westack-go/westack/utils"
)

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WeStack struct {
	Server  *fiber.App
	Viper   *viper.Viper
	DsViper *viper.Viper
	Options Options
	Bson    wst.BsonOptions

	port              int
	datasources       *map[string]*datasource.Datasource
	modelRegistry     *map[string]*model.Model
	debug             bool
	restApiRoot       string
	roleMappingModel  *model.Model
	dataSourceOptions *map[string]*datasource.Options
	init              time.Time
	jwtSecretKey      []byte
	swaggerHelper     swaggerhelperinterface.SwaggerHelper
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

func (app *WeStack) FindModelsWithClass(modelClass string) (foundModels []*model.Model) {
	for _, foundModel := range *app.modelRegistry {
		if foundModel.Config.Base == modelClass {
			foundModels = append(foundModels, foundModel)
		}
	}
	return
}

func (app *WeStack) Boot(customRoutesCallbacks ...func(app *WeStack)) {

	err := app.loadDataSources()
	if err != nil {
		log.Fatalf("Error while loading datasources: %v", err)
	}

	err = app.loadModels()
	if err != nil {
		log.Fatalf("Error while loading models: %v", err)
	}

	pprofAuthUsername := os.Getenv("PPROF_AUTH_USERNAME")
	pprofAuthPassword := os.Getenv("PPROF_AUTH_PASSWORD")
	if pprofAuthUsername != "" && pprofAuthPassword != "" {
		app.Middleware(utils.PprofHandlers(utils.PprofMiddleOptions{
			Auth: utils.BasicAuthOptions{
				Username: pprofAuthUsername,
				Password: pprofAuthPassword,
			},
		}))
	}

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

	if app.Options.EnableCompression {
		app.Middleware(compress.New(app.Options.CompressionConfig))
	}

	err = app.loadModelsFixedRoutes()
	if err != nil {
		log.Fatalf("Error while loading models fixed routes: %v", err)
	}

	systemContext := &model.EventContext{
		Bearer: &model.BearerToken{User: &model.BearerUser{System: true}},
	}

	// Upsert the admin user
	_, err = UpsertUserWithRoles(app, UserWithRoles{
		Username: app.Options.adminUsername,
		Password: app.Options.adminPwd,
		Roles:    []string{"admin"},
	}, systemContext)
	if err != nil {
		log.Fatalf("Error while creating admin user: %v", err)
	}

	for _, cb := range customRoutesCallbacks {
		cb(app)
	}

	app.loadModelsDynamicRoutes()
	app.loadNotFoundRoutes()

	app.Server.Get("/system/memorykv/stats", func(c *fiber.Ctx) error {
		allStats := make(map[string]map[string]memorykv.MemoryKvStats)
		var totalSizeKiB float64
		for _, ds := range *app.datasources {
			if ds.SubViper.GetString("connector") == "memorykv" {
				kvDbStats := ds.Db.(memorykv.MemoryKvDb).Stats()
				allStats[ds.Name] = kvDbStats
				for _, kvStats := range kvDbStats {
					totalSizeKiB += float64(kvStats.TotalSize) / 1024.0
				}
			}
		}
		return c.JSON(fiber.Map{"stats": wst.M{
			"totalSizeKiB": totalSizeKiB,
			"datasources":  allStats,
		}})
	})

	app.Server.Get("/swagger/doc.json", swaggerDocsHandler(app))

	var swaggerUIStatic []byte
	var swaggerContentEncoding = "deflate"
	app.Server.Get("/swagger/*", func(ctx *fiber.Ctx) error {
		if swaggerUIStatic == nil {
			request, err := http.NewRequest("GET", "https://swagger-ui.fhcreations.com/", nil)
			if err != nil {
				return err
			}
			request.Header.Set("Accept", "text/html")
			request.Header.Set("Accept-Encoding", "gzip, deflate, br")
			//request.Header.Set("Accept-Language", "en-US,en;q=0.9")
			//request.Header.Set("Cache-Control", "no-cache")

			response, err := http.DefaultClient.Do(request)

			for _, v := range response.Header["Content-Encoding"] {
				swaggerContentEncoding = v
			}

			var reader io.Reader
			//// decompress
			//switch swaggerContentEncoding {
			//case "gzip":
			//	reader, err = gzip.NewReader(response.Body)
			//	if err != nil {
			//		break
			//	}
			//case "br":
			//	reader = brotli.NewReader(response.Body)
			//case "deflate", "":
			reader = response.Body
			//}
			swaggerUIStatic, err = io.ReadAll(reader)
			if err != nil {
				return err
			}

			fmt.Printf("DEBUG: Fetched swagger ui static html (%v bytes)\n", len(swaggerUIStatic))
			fmt.Printf("DEBUG: Swagger Content-Encoding: %v\n", swaggerContentEncoding)
		}

		ctx.Status(fiber.StatusOK).Set("Content-Type", "text/html; charset=utf-8")
		ctx.Set("Content-Encoding", swaggerContentEncoding)

		return ctx.Send(swaggerUIStatic)
	})

	// Free up memory
	err = app.swaggerHelper.Dump()
	if err != nil {
		fmt.Printf("Error while dumping swagger helper: %v\n", err)
	}

}

func (app *WeStack) Start() error {
	log.Printf("DEBUG Server took %v ms to start\n", time.Now().UnixMilli()-app.init.UnixMilli())
	return app.Server.Listen(fmt.Sprintf("0.0.0.0:%v", app.port))
}

func (app *WeStack) Middleware(handler fiber.Handler) {
	app.Server.Use(handler)
}

func (app *WeStack) Stop() error {
	log.Println("Stopping server")
	for _, ds := range *app.datasources {
		err := ds.Close()
		if err != nil {
			return err
		}
	}
	err := app.Server.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func GRPCCallWithQueryParams[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return gRPCCallWithQueryParams(serviceUrl, clientConstructor, clientMethod)
}

func GRPCCallWithBody[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return gRPCCallWithBody(serviceUrl, clientConstructor, clientMethod)
}

type Options struct {
	RestApiRoot       string
	Port              int
	JwtSecretKey      string
	DatasourceOptions *map[string]*datasource.Options
	EnableCompression bool
	CompressionConfig compress.Config

	debug         bool
	adminUsername string
	adminPwd      string
}

func New(options ...Options) *WeStack {
	server := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

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

	adminUsername, present := os.LookupEnv("WST_ADMIN_USERNAME")
	if !present {
		log.Fatalf("WST_ADMIN_USERNAME environment variable is not set")
	}
	finalOptions.adminUsername = adminUsername

	adminPassword, present := os.LookupEnv("WST_ADMIN_PWD")
	if !present {
		log.Fatalf("WST_ADMIN_PWD environment variable is not set")
	}
	finalOptions.adminPwd = adminPassword

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
	appViper.AddConfigPath(".")        // for unit tests

	err := appViper.ReadInConfig() // Find and read the config file
	if err != nil {                // Handle errors reading the config file
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Println(fmt.Sprintf("WARNING: %v.json not found, fallback to config.json", fileToLoad))
			appViper.SetConfigName("config") // name of config file (without extension)
			err := appViper.ReadInConfig()   // Find and read the config file
			if err != nil {
				log.Fatalf("fatal error config file: %w", err)
			}
			break
		default:
			log.Fatalf("fatal error config file: %w", err)
		}
	}

	if finalOptions.RestApiRoot == "" {
		finalOptions.RestApiRoot = appViper.GetString("restApiRoot")
	}
	if finalOptions.Port == 0 {
		finalOptions.Port = appViper.GetInt("port")
	}
	if os.Getenv("PORT") != "" {
		portFromEnv, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			log.Fatalf("Invalid PORT environment variable: %v", err)
		}
		if finalOptions.debug {
			log.Printf("DEBUG: PORT environment variable is set to %v", portFromEnv)
		}
		finalOptions.Port = portFromEnv
	}

	var bsonRegistry *bsoncodec.Registry
	if finalOptions.DatasourceOptions != nil {
		for _, v := range *finalOptions.DatasourceOptions {
			if v.MongoDB != nil && v.MongoDB.Registry != nil {
				bsonRegistry = v.MongoDB.Registry
				break
			}
		}
	}
	if bsonRegistry == nil {
		bsonRegistry = bson.NewRegistryBuilder().RegisterCodec(reflect.TypeOf(primitive.ObjectID{}), &bsoncodec.TimeCodec{}).Build()
	}
	app := WeStack{
		Server:  server,
		Viper:   appViper,
		Options: finalOptions,
		Bson: wst.BsonOptions{
			Registry: bsonRegistry,
		},

		modelRegistry:     &modelRegistry,
		datasources:       &datasources,
		debug:             _debug,
		restApiRoot:       finalOptions.RestApiRoot,
		port:              finalOptions.Port,
		jwtSecretKey:      []byte(finalOptions.JwtSecretKey),
		dataSourceOptions: finalOptions.DatasourceOptions,
		init:              time.Now(),
	}

	return &app
}

func InitAndServe() {
	app := New()

	app.Boot()

	log.Fatal(app.Start())
}
