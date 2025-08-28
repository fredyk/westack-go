package westack

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"google.golang.org/grpc"

	"github.com/goccy/go-json"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
	"github.com/fredyk/westack-go/v2/model"
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

	port                           int
	datasources                    *map[string]*datasource.Datasource
	modelRegistry                  *map[string]*model.StatefulModel
	restrictModelUniquenessByField map[string]map[string]UniqueNessRestriction
	debug                          bool
	restApiRoot                    string
	roleMappingModel               *model.StatefulModel
	accountCredentialsModel        *model.StatefulModel
	mfaModel                       *model.StatefulModel
	dataSourceOptions              *map[string]*datasource.Options
	init                           time.Time
	jwtSecretKey                   []byte
	swaggerHelper                  wst.SwaggerHelper
	logger                         wst.ILogger
	completedSetup                 bool
	registerControllers            func(r model.ControllerRegistry)
	
	// Security tracking
	loginAttempts                  map[string][]time.Time
	lockedAccounts                 map[string]time.Time
}

type BootOptions struct {
	RegisterControllers func(r model.ControllerRegistry)
}

func (app *WeStack) FindModel(modelName string) (*model.StatefulModel, error) {
	result := (*app.modelRegistry)[modelName]
	if result == nil {
		return nil, fmt.Errorf("model %v not found", modelName)
	}
	return result, nil
}

func (app *WeStack) FindDatasource(dsName string) (*datasource.Datasource, error) {
	result := (*app.datasources)[dsName]

	if result == nil {
		return nil, fmt.Errorf("datasource %v not found", dsName)
	}

	return result, nil
}

func (app *WeStack) FindModelsWithClass(modelClass string) (foundModels []*model.StatefulModel) {
	for _, foundModel := range *app.modelRegistry {
		if foundModel.Config.Base == modelClass {
			foundModels = append(foundModels, foundModel)
		}
	}
	return
}

func (app *WeStack) Boot(options BootOptions, customRoutesCallbacks ...func(app *WeStack)) {

	if options.RegisterControllers == nil {
		app.logger.Fatal(`RegisterControllers is required. Check your boot function contains following:
		app.Boot(BootOptions{
			RegisterControllers: models.RegisterControllers,
		}, customRoutesCallbacks...)`)
	}
	app.registerControllers = options.RegisterControllers
	appBoot(customRoutesCallbacks, app)

}

func (app *WeStack) Start() error {
	log.Printf("[INFO] Swagger available at http://localhost:%v/swagger\n", app.port)
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
	return app.Server.Shutdown()
}

func (app *WeStack) Logger() wst.ILogger {
	return app.logger
}

// Security methods for login protection
func (app *WeStack) checkLoginRateLimit(identifier string) error {
	if app.loginAttempts == nil {
		app.loginAttempts = make(map[string][]time.Time)
	}
	
	now := time.Now()
	attempts := app.loginAttempts[identifier]
	
	// Remove attempts older than 15 minutes
	var recentAttempts []time.Time
	for _, attempt := range attempts {
		if now.Sub(attempt) < 15*time.Minute {
			recentAttempts = append(recentAttempts, attempt)
		}
	}
	
	// Check if too many attempts
	if len(recentAttempts) >= 5 {
		return wst.CreateError(fiber.ErrTooManyRequests, "RATE_LIMITED", fiber.Map{
			"message": "Too many login attempts. Please try again later.",
		}, "RateLimitError")
	}
	
	// Add current attempt
	recentAttempts = append(recentAttempts, now)
	app.loginAttempts[identifier] = recentAttempts
	
	return nil
}

func (app *WeStack) checkAccountLockout(identifier string) error {
	if app.lockedAccounts == nil {
		app.lockedAccounts = make(map[string]time.Time)
	}
	
	lockTime, isLocked := app.lockedAccounts[identifier]
	if isLocked {
		// Check if lockout period has expired (30 minutes)
		if time.Since(lockTime) < 30*time.Minute {
			return wst.CreateError(fiber.ErrLocked, "ACCOUNT_LOCKED", fiber.Map{
				"message": "Account is temporarily locked due to too many failed login attempts.",
			}, "AccountLockError")
		}
		// Lockout expired, remove from locked accounts
		delete(app.lockedAccounts, identifier)
	}
	
	return nil
}

func (app *WeStack) recordFailedLogin(identifier string) {
	if app.loginAttempts == nil {
		app.loginAttempts = make(map[string][]time.Time)
	}
	if app.lockedAccounts == nil {
		app.lockedAccounts = make(map[string]time.Time)
	}
	
	attempts := app.loginAttempts[identifier]
	now := time.Now()
	
	// Count recent failed attempts (last 15 minutes)
	var recentFailures int
	for _, attempt := range attempts {
		if now.Sub(attempt) < 15*time.Minute {
			recentFailures++
		}
	}
	
	// Lock account after 5 failed attempts
	if recentFailures >= 5 {
		app.lockedAccounts[identifier] = now
	}
}

func (app *WeStack) clearFailedLogins(identifier string) {
	if app.loginAttempts != nil {
		delete(app.loginAttempts, identifier)
	}
	if app.lockedAccounts != nil {
		delete(app.lockedAccounts, identifier)
	}
}

func (app *WeStack) logSecurityEvent(eventType, identifier string, success bool, metadata map[string]interface{}) {
	logData := map[string]interface{}{
		"timestamp":  time.Now().UTC(),
		"event_type": eventType,
		"identifier": identifier,
		"success":    success,
		"metadata":   metadata,
	}
	
	if app.debug {
		app.logger.Printf("[SECURITY] %s - %s: %v", eventType, identifier, logData)
	}
	
	// In production, this should write to a security audit log
	// For now, we'll use the standard logger
	if !success {
		app.logger.Printf("[SECURITY ALERT] %s failed for %s: %v", eventType, identifier, metadata)
	}
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
	BodyLimit         int
	ReadBufferSize    int
	WriteBufferSize   int

	debug             bool
	adminUsername     string
	adminPwd          string
	Logger            wst.ILogger
	DisablePortEnvVar bool
}

func New(options ...Options) *WeStack {

	var logger wst.ILogger

	var bodyLimit int = 4 * 1024 * 1024
	var readBufferSize int = 4 * 1024
	var writeBufferSize int = 4 * 1024

	var finalOptions Options
	if len(options) > 0 {
		finalOptions = options[0]
		if finalOptions.BodyLimit > 0 {
			bodyLimit = finalOptions.BodyLimit
		}
		if finalOptions.ReadBufferSize > 0 {
			readBufferSize = finalOptions.ReadBufferSize
		}
		if finalOptions.WriteBufferSize > 0 {
			writeBufferSize = finalOptions.WriteBufferSize
		}
	}

	server := fiber.New(fiber.Config{
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		DisableStartupMessage: true,
		BodyLimit:             bodyLimit,
		ReadBufferSize:        readBufferSize,
		WriteBufferSize:       writeBufferSize,
	})

	modelRegistry := make(map[string]*model.StatefulModel)
	datasources := make(map[string]*datasource.Datasource)

	if finalOptions.Logger != nil {
		logger = finalOptions.Logger
	} else {
		logger = log.New(os.Stdout, "[westack] ", 0)
	}

	if finalOptions.JwtSecretKey == "" {
		if s, present := os.LookupEnv("JWT_SECRET"); present && wst.IsSecurePassword(s) {
			if len(strings.TrimSpace(s)) >= 13 {
				finalOptions.JwtSecretKey = s
			} else {
				logger.Fatalf("Invalid JWT secret. It must be at least 13 characters long")
			}
		} else {
			logger.Fatalf("Missing JWT secret. Either set JWT_SECRET environment variable or provide it in the options as JwtSecretKey.\nPassword length must be at least 8 characters and contain at least one uppercase letter, one lowercase letter, one number and one special character")
		}
	}
	if envDebug, _ := os.LookupEnv("DEBUG"); envDebug == "true" {
		finalOptions.debug = true
	}

	adminUsername, present := os.LookupEnv("WST_ADMIN_USERNAME")
	if !present {
		logger.Fatalf("WST_ADMIN_USERNAME environment variable is not set")
	}
	finalOptions.adminUsername = adminUsername

	adminPassword, present := os.LookupEnv("WST_ADMIN_PWD")
	if !present || !wst.IsSecurePassword(adminPassword) {
		logger.Fatalf("WST_ADMIN_PWD environment variable is not set. Password length must be at least 8 characters and contain at least one uppercase letter, one lowercase letter, one number and one special character")
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
		switch {
		case errors.As(err, &viper.ConfigFileNotFoundError{}):
			fmt.Printf("[WARNING] %v.json not found, fallback to config.json\n", fileToLoad)
			appViper.SetConfigName("config") // name of config file (without extension)
			err := appViper.ReadInConfig()   // Find and read the config file
			if err != nil {
				log.Fatalf("fatal error config file: %v", err)
			}
		default:
			log.Fatalf("fatal error config file: %v", err)
		}
	}

	appViper.SetEnvPrefix("wst")
	replacer := strings.NewReplacer(".", "_")
	appViper.SetEnvKeyReplacer(replacer)
	appViper.AutomaticEnv()

	replaceEnvVars(appViper)

	if finalOptions.RestApiRoot == "" {
		finalOptions.RestApiRoot = appViper.GetString("restApiRoot")
	}
	if finalOptions.Port == 0 {
		finalOptions.Port = appViper.GetInt("port")
	}
	if os.Getenv("PORT") != "" && !finalOptions.DisablePortEnvVar {
		portFromEnv, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			logger.Fatalf("Invalid PORT environment variable: %v", err)
		}
		if finalOptions.debug {
			log.Printf("[DEBUG] PORT environment variable is set to %v", portFromEnv)
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

		modelRegistry:                  &modelRegistry,
		restrictModelUniquenessByField: make(map[string]map[string]UniqueNessRestriction),
		datasources:                    &datasources,
		debug:                          finalOptions.debug,
		restApiRoot:                    finalOptions.RestApiRoot,
		port:                           finalOptions.Port,
		jwtSecretKey:                   []byte(finalOptions.JwtSecretKey),
		dataSourceOptions:              finalOptions.DatasourceOptions,
		init:                           time.Now(),
		logger:                         logger,
		
		// Initialize security tracking maps
		loginAttempts:                  make(map[string][]time.Time),
		lockedAccounts:                 make(map[string]time.Time),
	}

	return &app
}

func InitAndServe(options Options, onBoot ...func(app *WeStack)) {
	app := New(options)

	app.Boot(BootOptions{RegisterControllers: func(r model.ControllerRegistry) {

	}}, onBoot...)

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
