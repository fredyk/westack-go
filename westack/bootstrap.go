package westack

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/fredyk/westack-go/westack/lib/swaggerhelper"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
)

type UpserRequestBody struct {
	Roles []string `json:"roles"`
}

var ValidEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$`)

func isAllowedForProtectedFields(bearer *model.BearerToken) bool {
	var roles []model.BearerRole
	if bearer != nil {
		roles = bearer.Roles
		if len(roles) > 0 {
			for i := 0; i < len(roles); i++ {
				if roles[i].Name == "admin" || roles[i].Name == "__protectedFieldsPrivileged" {
					return true
				}
			}
		}
	}
	return false
}

func (app *WeStack) loadModels() error {

	// List directory common/models without using ioutil.ReadDir
	// https://stackoverflow.com/questions/5884154/read-all-files-in-a-directory-in-go
	//fileInfos, err := ioutil.ReadDir("./common/models")
	fileInfos, err := fs.ReadDir(os.DirFS("./common/models"), ".")

	if err != nil {
		return fmt.Errorf("error while loading models: " + err.Error())
	}

	var globalModelConfig *map[string]*model.SimplifiedConfig
	if err := wst.LoadFile("./server/model-config.json", &globalModelConfig); err != nil {
		return fmt.Errorf("missing or invalid ./server/model-config.json: " + err.Error())
	}

	app.swaggerHelper = swaggerhelper.NewSwaggerHelper()
	err = app.swaggerHelper.CreateOpenAPI()
	if err != nil {
		return err
	}
	var someUserModel *model.StatefulModel
	for _, fileInfo := range fileInfos {

		if fileInfo.IsDir() {
			continue
		}

		if strings.Split(fileInfo.Name(), ".")[1] != "json" {
			continue
		}
		var config *model.Config
		err := wst.LoadFile("./common/models/"+fileInfo.Name(), &config)
		if err != nil {
			return fmt.Errorf("error while loading model %v: %v", fileInfo.Name(), err)
		}
		if config.Relations == nil {
			config.Relations = &map[string]*model.Relation{}
		}

		configFromGlobal := (*globalModelConfig)[config.Name]

		if configFromGlobal == nil {
			return fmt.Errorf("missing model " + config.Name + " in model-config.json")
		}

		dataSource := (*app.datasources)[configFromGlobal.Datasource]

		if dataSource == nil {
			return fmt.Errorf("missing or invalid datasource file for %v", dataSource)
		}

		loadedModel := model.New(config, app.modelRegistry)
		err = app.setupModel(loadedModel.(*model.StatefulModel), dataSource)
		if err != nil {
			return err
		}
		if loadedModel.(*model.StatefulModel).Config.Base == "User" {
			someUserModel = loadedModel.(*model.StatefulModel)
		}
	}

	if app.roleMappingModel != nil {
		(*app.roleMappingModel.Config.Relations)["user"].Model = someUserModel.Name
		err := app.setupModel(app.roleMappingModel, app.roleMappingModel.Datasource)
		if err != nil {
			return err
		}
	}

	err2 := fixRelations(app)
	if err2 != nil {
		return err2
	}
	buildRelationsGraph(app)
	return nil
}

type UniqueNessRestriction struct {
	Code      string
	Message   string
	ErrorName string
}

func buildRelationsGraph(app *WeStack) {
	for _, thisModel := range *app.modelRegistry {
		for _, otherModel := range *app.modelRegistry {
			for _, relation := range *otherModel.Config.Relations {
				if relation.Model == thisModel.Name {
					if relation.Type == "hasOne" {
						// Possible inverse relation bulding:
						//if thisModel.Config.Relations == nil {
						//	thisModel.Config.Relations = &map[string]*model.Relation{}
						//}
						//(*thisModel.Config.Relations)[relationName] = &model.Relation{
						//	Type:  "belongsTo",
						//	Model: otherModel.Name,
						//}

						// Restrict hasOne relations to be only one-to-one
						if _, ok := app.restrictModelUniquenessByField[thisModel.Name]; !ok {
							app.restrictModelUniquenessByField[thisModel.Name] = make(map[string]UniqueNessRestriction)
						}
						app.restrictModelUniquenessByField[thisModel.Name][*relation.ForeignKey] = UniqueNessRestriction{
							Code:      "UNIQUENESS",
							Message:   fmt.Sprintf("The `%v` instance is not valid. Details: %v already exists.", thisModel.Name, *relation.ForeignKey),
							ErrorName: "ValidationError",
						}
					}
					break
				}
			}
		}
	}
}

func fixRelations(app *WeStack) error {
	for _, loadedModel := range *app.modelRegistry {
		err := fixModelRelations(loadedModel)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *WeStack) loadDataSources() error {

	dsViper := viper.New()
	app.DsViper = dsViper

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
			fmt.Printf("[WARNING] %v.json not found, fallback to datasources.json\n", fileToLoad)
			dsViper.SetConfigName("datasources") // name of config file (without extension)
			err := dsViper.ReadInConfig()        // Find and read the config file
			if err != nil {
				return fmt.Errorf("fatal error config file: %w", err)
			}
		default:
			return fmt.Errorf("fatal error config file: %w", err)
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
		if connector == "mongodb" || connector == "memorykv" {
			ds := datasource.New(app.asInterface(), key, dsViper, ctx)

			if app.dataSourceOptions != nil {
				ds.Options = (*app.dataSourceOptions)[dsName]
				if ds.Options == nil {
					ds.Options = &datasource.Options{}
				}
				ds.Options.RetryOnError = dsViper.GetBool(key + ".retryOnError")
			}

			err := ds.Initialize()
			if err != nil {
				return fmt.Errorf("could not initialize datasource %v: %v", dsName, err)
			}
			(*app.datasources)[dsName] = ds
			if app.debug {
				log.Println("Connected to database", dsViper.GetString(key+".database"))
			}
		} else {
			return fmt.Errorf("connector " + connector + " not supported")
		}
	}

	//dsViper.Set("<internal>", wst.M{
	//	"connector": "memorykv",
	//	"database":  "westack",
	//	"name":      "<internal>",
	//})
	dsViper.SetDefault("<internal>.connector", "memorykv")
	dsViper.SetDefault("<internal>.database", "westack")
	dsViper.SetDefault("<internal>.name", "<internal>")
	internalDs := datasource.New(app.asInterface(), "<internal>", dsViper, ctx)
	err = internalDs.Initialize()
	if err != nil {
		return fmt.Errorf("could not initialize datasource <internal>: %v", err)
	}
	(*app.datasources)["<internal>"] = internalDs

	return nil
}

func (app *WeStack) setupModel(loadedModel *model.StatefulModel, dataSource *datasource.Datasource) error {

	loadedModel.App = app.asInterface()
	loadedModel.Datasource = dataSource

	config := loadedModel.Config

	loadedModel.Initialize()

	if config.Base == "Role" {
		setupRoleModel(config, app, dataSource)
	}

	if config.Base == "User" {

		setupUserModel(loadedModel, app)

	}

	var plural string
	if config.Plural != "" {
		plural = config.Plural
	} else {
		plural = wst.DashedCase(regexp.MustCompile("y$").ReplaceAllString(config.Name, "ie")) + "s"
	}
	config.Plural = plural

	err := createCasbinModel(loadedModel, app, config)
	if err != nil {
		return err
	}

	if loadedModel.Config.Public {

		modelRouter := app.Server.Group(app.restApiRoot+"/"+plural, func(ctx *fiber.Ctx) error {
			return ctx.Next()
		})
		loadedModel.Router = &modelRouter

		loadedModel.BaseUrl = app.restApiRoot + "/" + plural

		loadedModel.On("findMany", func(ctx *model.EventContext) error {
			return handleFindMany(app, loadedModel, ctx)
		})
		loadedModel.On("count", func(ctx *model.EventContext) error {
			result, err := loadedModel.Count(ctx.Filter, ctx)
			if err != nil {
				return err
			}
			ctx.StatusCode = fiber.StatusOK
			ctx.Result = wst.M{"count": result}
			return nil
		})
		loadedModel.On("findById", func(ctx *model.EventContext) error {
			result, err := loadedModel.FindById(ctx.ModelID, ctx.Filter, ctx)
			if result != nil {
				result.(*model.StatefulInstance).HideProperties()
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
			intervalPattern := regexp.MustCompile(`^[-+]\d+s$`)

			if _, ok := (*data)["modified"]; !ok {
				timeNow := time.Now()
				(*data)["modified"] = timeNow
			}

			if ctx.IsNewInstance {
				if _, ok := (*data)["created"]; !ok {
					timeNow := time.Now()
					(*data)["created"] = timeNow
				}

				for propertyName, propertyConfig := range config.Properties {
					defaultValue := propertyConfig.Default
					if defaultValue != nil {
						if _, ok := (*data)[propertyName]; !ok {
							if defaultValue == "null" {
								defaultValue = nil
							}
							if propertyConfig.Type == "date" {
								if defaultValue == "$now" {
									(*data)[propertyName] = time.Now()
									continue
								}
								if match := intervalPattern.MatchString(defaultValue.(string)); match {
									secondsString := defaultValue.(string)[1 : len(defaultValue.(string))-1]
									seconds, err := strconv.Atoi(secondsString)
									if err != nil {
										return err
									}

									adjustment := 1
									if defaultValue.(string)[0] == '-' {
										adjustment = -1
									}

									defaultValue = time.Now().Add(time.Duration(adjustment*seconds) * time.Second)
								}
							}
							(*data)[propertyName] = defaultValue
						}
					}
				}

				if config.Base == "User" {
					username := (*data)["username"]
					email := (*data)["email"]
					if (username == nil || strings.TrimSpace(username.(string)) == "") && (email == nil || strings.TrimSpace(email.(string)) == "") {
						return wst.CreateError(fiber.ErrBadRequest, "EMAIL_PRESENCE", fiber.Map{"message": "Either username or email is required", "codes": wst.M{"email": []string{"presence"}}}, "ValidationError")
					}

					if email != nil && !ValidEmailRegex.MatchString(email.(string)) {
						return wst.CreateError(fiber.ErrBadRequest, "EMAIL_FORMAT", fiber.Map{"message": "Invalid email format", "codes": wst.M{"email": []string{"format"}}}, "ValidationError")
					}

					if username != nil && strings.TrimSpace(username.(string)) != "" {
						filter := wst.Filter{Where: &wst.Where{"username": username}}
						existent, err2 := loadedModel.FindOne(&filter, ctx)
						if err2 != nil {
							return err2
						}
						if existent != nil {
							return wst.CreateError(fiber.ErrConflict, "USERNAME_UNIQUENESS", fiber.Map{"message": fmt.Sprintf("The `user` instance is not valid. Details: `username` User already exists (value: \"%v\").", username), "codes": wst.M{"username": []string{"uniqueness"}}}, "ValidationError")
						}
					}

					if email != nil && strings.TrimSpace(email.(string)) != "" {
						filter := wst.Filter{Where: &wst.Where{"email": email}}
						existent, err2 := loadedModel.FindOne(&filter, ctx)
						if err2 != nil {
							return err2
						}
						if existent != nil {
							return wst.CreateError(fiber.ErrConflict, "EMAIL_UNIQUENESS", fiber.Map{"message": fmt.Sprintf("The `user` instance is not valid. Details: `email` Email already exists (value: \"%v\").", email), "codes": wst.M{"email": []string{"uniqueness"}}}, "ValidationError")
						}
					}

					if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
						return wst.CreateError(fiber.ErrBadRequest, "PASSWORD_BLANK", fiber.Map{"message": "Invalid password"}, "ValidationError")
					}
					hashed, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("%s%s", string(loadedModel.App.JwtSecretKey), (*data)["password"].(string))), 10)
					if err != nil {
						return err
					}
					(*data)["password"] = string(hashed)

					if app.debug {
						fmt.Printf("Create User: ('%v', '%v')\n", (*data)["username"], (*data)["email"])
					}
				}

				// Check inverse hasOne uniqueness
				if _, ok := app.restrictModelUniquenessByField[loadedModel.Name]; ok {
					for foreignKey, restriction := range app.restrictModelUniquenessByField[loadedModel.Name] {
						if (*data)[foreignKey] != nil {
							filter := wst.Filter{Where: &wst.Where{foreignKey: (*data)[foreignKey]}}
							existent, err2 := loadedModel.FindOne(&filter, &model.EventContext{Bearer: &model.BearerToken{User: &model.BearerUser{System: true}}})
							if err2 != nil {
								return err2
							}
							if existent != nil && existent.GetID() != nil {
								if app.debug {
									fmt.Printf("[ERROR] inverse hasOne restriction triggered for [%v %v=%v]\n", loadedModel.Name, foreignKey, (*data)[foreignKey])
								}
								return wst.CreateError(fiber.ErrConflict, restriction.Code, fiber.Map{"message": restriction.Message, "codes": wst.M{foreignKey: []string{strings.ToLower(restriction.Code)}}}, restriction.ErrorName)
							}
						}
					}
				}

			} else {
				if config.Base == "User" {
					if (*data)["password"] != nil && (*data)["password"] != "" {
						log.Println("Update User password")
						hashed, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("%s%s", string(loadedModel.App.JwtSecretKey), (*data)["password"].(string))), 10)
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

		protectedFieldsCount := len(loadedModel.Config.Protected)
		loadedModel.Observe("before build", func(eventContext *model.EventContext) error {
			if protectedFieldsCount <= 0 || eventContext.BaseContext.Bearer.User.System || skipOperationForBeforeBuild(eventContext.OperationName) {
				return nil
			}
			isDifferentUser := true
			if eventContext.BaseContext.Bearer != nil && eventContext.BaseContext.Bearer.User != nil {
				foundUserId := eventContext.ModelID.(primitive.ObjectID).Hex()
				requesterUserId := eventContext.BaseContext.Bearer.User.Id
				if v, ok := requesterUserId.(primitive.ObjectID); ok {
					requesterUserId = v.Hex()
				}
				isDifferentUser = foundUserId != requesterUserId.(string)
			}
			if isDifferentUser && !isAllowedForProtectedFields(eventContext.BaseContext.Bearer) {
				for _, hiddenProperty := range loadedModel.Config.Protected {
					delete(*eventContext.Data, hiddenProperty)
				}
			}
			return nil
		})

		deleteByIdHandler := func(ctx *model.EventContext) error {
			deleteResult, err := loadedModel.DeleteById(ctx.ModelID, ctx)
			if err == nil {
				deletedCount := deleteResult.DeletedCount
				if deletedCount != 1 {
					return wst.CreateError(fiber.ErrBadRequest, "BAD_REQUEST", fiber.Map{"message": fmt.Sprintf("Deleted %v instances for %v", deletedCount, ctx.ModelID)}, "Error")
				}
				ctx.StatusCode = fiber.StatusNoContent
				ctx.Result = ""
			}
			return err
		}
		loadedModel.On("instance_delete", deleteByIdHandler)

		if config.Base == "User" {
			upsertUserRolesHandler := func(ctx *model.EventContext) error {
				var body UpserRequestBody
				err := ctx.Ctx.BodyParser(&body)
				if err == nil {
					err = UpsertUserRoles(app, ctx.ModelID, body.Roles, ctx)
					if err == nil {
						ctx.StatusCode = fiber.StatusOK
						ctx.Result = wst.M{"result": "OK"}
					}
				}
				return err
			}
			loadedModel.On("user_upsertRoles", upsertUserRolesHandler)
		}

	}
	return nil
}

func handleFindMany(app *WeStack, loadedModel *model.StatefulModel, ctx *model.EventContext) error {
	if loadedModel.App.Debug {
		fmt.Println("[DEBUG] handleFindMany")
	}

	cursor := loadedModel.FindMany(ctx.Filter, ctx)
	if v, ok := cursor.(*model.ErrorCursor); ok {
		defer func(v *model.ErrorCursor) {
			err := v.Close()
			if err != nil {
				log.Println("Error while closing cursor at handleFindMany(): ", err)
			}
		}(v)
		var err error
		ctx.Result, err = v.Next()
		return err
	}
	chunkGenerator, err := traceChunkGenerator(app, loadedModel, ctx, cursor)
	if err != nil {
		return err
	}
	//chunkGenerator := model.NewCursorChunkGenerator(loadedModel, cursor)
	//switch cursor.(type) {
	//case *model.ErrorCursor:
	//	return cursor.(*model.ErrorCursor).Error()
	//}
	// Check if it is a *model.ChannelCursor, then check if it has an error
	if v, ok := cursor.(*model.ChannelCursor); ok && v.Err == nil {
		ctx.StatusCode = fiber.StatusOK
	} else if _, ok := cursor.(*model.FixedLengthCursor); ok {
		// No error
		ctx.StatusCode = fiber.StatusOK
	}
	ctx.Result = chunkGenerator
	return nil
}

// traceChunkGenerator is a helper function to trace the cursorChunkGenerator. For a given
// ctx.Filter:
// This is the flow:
//
//	if firstTime(ctx.Filter) || registeredError(ctx.Filter) {
//	  chunkGenerator = createFixedChunkGenerator(cursor)
//	  chunkGenerator.OnError(func (chunkGenerator, cursor, err) {
//	    registerError(ctx.Filter, err)
//	  })
//	  return chunkGenerator
//	} else {
//
//		 return createCursorChunkGenerator(cursor)
//	}
func traceChunkGenerator(app *WeStack, loadedModel *model.StatefulModel, ctx *model.EventContext, cursor model.Cursor) (model.ChunkGenerator, error) {
	internalDs, err := app.FindDatasource("<internal>")
	if err != nil {
		return nil, err
	}
	baseContext := model.FindBaseContext(ctx)
	filterSt := loadedModel.Name
	if ctx.Remote != nil {
		filterSt += ":" + ctx.Remote.Name
	} else {
		filterSt += ":" + string(ctx.OperationName)
	}
	if baseContext.Query != nil {
		bytes, _ := json.Marshal(baseContext.Query)
		filterSt += ":q:" + string(bytes)
	}
	if ctx.Filter != nil {
		bytes, _ := json.Marshal(ctx.Filter)
		filterSt += ":f:" + string(bytes)
	}
	if app.debug {
		fmt.Printf("[DEBUG] traceChunkGenerator: %v\n", filterSt)
	}
	lastErrorEntries, err := findLastErrorEntries(internalDs, filterSt)
	if err != nil {
		return nil, err
	}
	lastRequestEntriesCursor, err := internalDs.FindMany("chunkGeneratorTraceRequests", &wst.A{{"$match": wst.M{"_redId": filterSt}}})
	if err != nil {
		return nil, err
	}
	var lastRequestEntries model.InstanceA
	if err := lastRequestEntriesCursor.All(context.Background(), &lastRequestEntries); err != nil {
		return nil, err
	}
	_, err = internalDs.Create("chunkGeneratorTraceRequests", &wst.M{
		"_redId": filterSt,
		"_entries": wst.A{
			{"date": time.Now()},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(lastRequestEntries) == 0 || len(lastErrorEntries) > 0 {
		//docs, err := cursor.All()
		var docs model.InstanceA
		for {
			doc, err := cursor.Next()
			if err != nil {
				return nil, err
			}
			if doc == nil {
				break
			}
			docs = append(docs, doc)
		}
		if err != nil {
			_, err2 := internalDs.Create("chunkGeneratorTraceErrors", &wst.M{
				"_redId": filterSt,
				"_entries": wst.A{
					{"hadError": true},
				},
			})
			if err2 != nil {
				return nil, err2
			}
			return nil, err
		}
		chunkGenerator := model.NewInstanceAChunkGenerator(loadedModel, docs, "application/json")
		return chunkGenerator, nil
	} else {
		return model.NewCursorChunkGenerator(loadedModel, cursor), nil
	}
}

func findLastErrorEntries(internalDs *datasource.Datasource, filterSt string) (model.InstanceA, error) {
	lastErrorEntriesCursor, err := internalDs.FindMany("chunkGeneratorTraceErrors", &wst.A{{"$match": wst.M{"_redId": filterSt}}})
	if err != nil {
		return nil, err
	}
	var lastErrorEntries model.InstanceA
	if err := lastErrorEntriesCursor.All(context.Background(), &lastErrorEntries); err != nil {
		return nil, err
	}
	return lastErrorEntries, nil
}

func (app *WeStack) asInterface() *wst.IApp {
	return &wst.IApp{
		Debug:        app.debug,
		JwtSecretKey: app.jwtSecretKey,
		Viper:        app.Viper,
		Bson:         app.Bson,
		FindModel: func(modelName string) (interface{}, error) {
			return app.FindModel(modelName)
		},
		FindDatasource: func(datasource string) (interface{}, error) {
			return app.FindDatasource(datasource)
		},
		SwaggerHelper: func() wst.SwaggerHelper {
			return app.swaggerHelper
		},
		Logger: func() wst.ILogger {
			return app.logger
		},
	}
}

func fixModelRelations(loadedModel *model.StatefulModel) error {
	for relationName, relation := range *loadedModel.Config.Relations {

		if relation.Type == "" {
			return fmt.Errorf("relation %v.%v has no type", loadedModel.Name, relationName)
		}

		relatedModelName := relation.Model
		relatedLoadedModel := (*loadedModel.GetModelRegistry())[relatedModelName]

		if relatedLoadedModel == nil {
			return fmt.Errorf("related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
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
			case "hasOne", "hasMany":
				foreignKey := strings.ToLower(loadedModel.Name[:1]) + loadedModel.Name[1:] + "Id"
				relation.ForeignKey = &foreignKey
				//(*loadedModel.Config.Relations)[relationName] = relation
			}
		}
	}
	return nil
}

func skipOperationForBeforeBuild(operationName wst.OperationName) bool {
	return operationName == wst.OperationNameCreate || operationName == wst.OperationNameCount /* || operationName == wst.OperationNameFindMany*/
}
