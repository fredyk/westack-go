package westack

import (
	"context"
	"errors"
	"fmt"
	"github.com/fredyk/westack-go/westack/lib/swaggerhelperinterface"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"google.golang.org/grpc"

	"github.com/goccy/go-json"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
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
	logger            wst.ILogger
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

	appBoot(customRoutesCallbacks, app)

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

func (app *WeStack) Logger() wst.ILogger {
	return app.logger
}

func GRPCCallWithQueryParams[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error), timeoutSeconds ...float32) func(ctx *fiber.Ctx) error {
	return gRPCCallWithQueryParams(serviceUrl, clientConstructor, clientMethod, timeoutSeconds...)
}

func GRPCCallWithBody[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error), timeoutSeconds ...float32) func(ctx *fiber.Ctx) error {
	return gRPCCallWithBody(serviceUrl, clientConstructor, clientMethod, timeoutSeconds...)
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
	Logger        wst.ILogger
}

func New(options ...Options) *WeStack {

	var logger wst.ILogger

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

	if finalOptions.Logger != nil {
		logger = finalOptions.Logger
	} else {
		logger = log.New(os.Stdout, "[westack] ", 0)
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
		logger.Fatalf("WST_ADMIN_USERNAME environment variable is not set")
	}
	finalOptions.adminUsername = adminUsername

	adminPassword, present := os.LookupEnv("WST_ADMIN_PWD")
	if !present {
		logger.Fatalf("WST_ADMIN_PWD environment variable is not set")
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
		finalOptions.Port = appViper.GetInt("port")
	}
	if os.Getenv("PORT") != "" {
		portFromEnv, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			logger.Fatalf("Invalid PORT environment variable: %v", err)
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
		bsonRegistry = wst.CreateDefaultMongoRegistry()
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
		logger:            logger,
	}

	return &app
}

func InitAndServe(options Options) {
	app := New(options)

	app.Boot()

	// Catch SIGINT signal and Stop()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		err := app.Stop()
		if err != nil {
			app.logger.Fatal(err)
		}
		os.Exit(0)
	}()

	app.logger.Fatal(app.Start())
}
