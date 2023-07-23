package westack

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fredyk/westack-go/westack/lib/swaggerhelper"
	swaggerhelper2 "github.com/fredyk/westack-go/westack/lib/swaggerhelperinterface"

	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
)

func (app *WeStack) loadModels() error {

	// List directory common/models without using ioutil.ReadDir
	// https://stackoverflow.com/questions/5884154/read-all-files-in-a-directory-in-go
	//fileInfos, err := ioutil.ReadDir("./common/models")
	fileInfos, err := fs.ReadDir(os.DirFS("./common/models"), ".")

	if err != nil {
		panic("Error while loading models: " + err.Error())
	}

	var globalModelConfig *map[string]*model.SimplifiedConfig
	if err := wst.LoadFile("./server/model-config.json", &globalModelConfig); err != nil {
		panic("Missing or invalid ./server/model-config.json: " + err.Error())
	}

	app.swaggerHelper = swaggerhelper.NewSwaggerHelper()
	err = app.swaggerHelper.CreateOpenAPI()
	if err != nil {
		return err
	}
	var someUserModel *model.Model
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
			fmt.Printf("Error while loading model %v: %v\n", fileInfo.Name(), err)
			panic(err)
		}
		if config.Relations == nil {
			config.Relations = &map[string]*model.Relation{}
		}

		configFromGlobal := (*globalModelConfig)[config.Name]

		if configFromGlobal == nil {
			panic("ERROR: Missing model " + config.Name + " in model-config.json")
		}

		dataSource := (*app.datasources)[configFromGlobal.Datasource]

		if dataSource == nil {
			panic(fmt.Sprintf("ERROR: Missing or invalid datasource entry %v for model %v declared at model-config.json", configFromGlobal.Datasource, config.Name))
		}

		loadedModel := model.New(config, app.modelRegistry)
		app.setupModel(loadedModel, dataSource)
		if loadedModel.Config.Base == "User" {
			someUserModel = loadedModel
		}
	}

	if app.roleMappingModel != nil {
		(*app.roleMappingModel.Config.Relations)["user"].Model = someUserModel.Name
		app.setupModel(app.roleMappingModel, app.roleMappingModel.Datasource)
	}

	for _, loadedModel := range *app.modelRegistry {
		err := fixRelations(loadedModel)
		if err != nil {
			return err
		}
	}
	return nil
}
func (app *WeStack) loadDataSources() {

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
		if connector == "mongodb" || connector == "memorykv" {
			ds := datasource.New(key, dsViper, ctx)

			if app.dataSourceOptions != nil {
				ds.Options = (*app.dataSourceOptions)[dsName]
			}

			err := ds.Initialize()
			if err != nil {
				panic(err)
			}
			(*app.datasources)[dsName] = ds
			if app.debug {
				log.Println("Connected to database", dsViper.GetString(key+".database"))
			}
		} else {
			panic("ERROR: connector " + connector + " not supported")
		}
	}
}

func (app *WeStack) setupModel(loadedModel *model.Model, dataSource *datasource.Datasource) {

	loadedModel.App = app.asInterface()
	loadedModel.Datasource = dataSource

	config := loadedModel.Config

	loadedModel.Initialize()

	if config.Base == "Role" {
		roleMappingModel := model.New(&model.Config{
			Name:   "RoleMapping",
			Plural: "role-mappings",
			Base:   "PersistedModel",
			//Datasource: config.Datasource,
			Public:     true,
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
					"roleManager,*,read,allow",
					"roleManager,*,write,allow",
				},
			},
		}, app.modelRegistry)
		roleMappingModel.App = app.asInterface()
		roleMappingModel.Datasource = dataSource

		app.roleMappingModel = roleMappingModel
	}

	if config.Base == "User" {

		loadedModel.On("login", func(ctx *model.EventContext) error {
			data := ctx.Data
			email := data.GetString("email")
			username := data.GetString("username")

			if email == "" && username == "" {
				return wst.CreateError(fiber.ErrBadRequest, "USERNAME_EMAIL_REQUIRED", fiber.Map{"message": "username or email is required"}, "ValidationError")
			}

			if (*data)["password"] == nil || strings.TrimSpace((*data)["password"].(string)) == "" {
				return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
			}

			var where wst.Where
			if email != "" {
				where = wst.Where{"email": email}
			} else {
				where = wst.Where{"username": username}
			}
			users, err := loadedModel.FindMany(&wst.Filter{
				Where: &where,
			}, ctx).All()
			if len(users) == 0 {
				return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
			}
			firstUser := users[0]
			ctx.Instance = &firstUser

			firstUserData := firstUser.ToJSON()
			savedPassword := firstUserData["password"]
			err = bcrypt.CompareHashAndPassword([]byte(savedPassword.(string)), []byte((*data)["password"].(string)))
			if err != nil {
				return wst.CreateError(fiber.ErrUnauthorized, "LOGIN_FAILED", fiber.Map{"message": "login failed"}, "Error")
			}

			userIdHex := firstUser.Id.(primitive.ObjectID).Hex()

			roleNames := []string{"USER"}
			if app.roleMappingModel != nil {
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
				roleEntries, err := app.roleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
					"principalType": "USER",
					"$or": []wst.M{
						{
							"principalId": userIdHex,
						},
						{
							"principalId": firstUser.Id,
						},
					},
				}, Include: &wst.Include{{Relation: "role"}}}, roleContext).All()
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

	basePoliciesDirectory := app.Viper.GetString("casbin.policies.outputDirectory")
	_, err := os.Stat(basePoliciesDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(basePoliciesDirectory, os.ModePerm)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	f, err := os.OpenFile(fmt.Sprintf("%v/%v.policies.csv", basePoliciesDirectory, loadedModel.Name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}

	adapter := fileadapter.NewAdapter(fmt.Sprintf("%v/%v.policies.csv", basePoliciesDirectory, loadedModel.Name))

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

		modelRouter := app.Server.Group(app.restApiRoot+"/"+plural, func(ctx *fiber.Ctx) error {
			return ctx.Next()
		})
		loadedModel.Router = &modelRouter

		loadedModel.BaseUrl = app.restApiRoot + "/" + plural

		loadedModel.On("findMany", func(ctx *model.EventContext) error {
			return handleFindMany(loadedModel, ctx)
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

				for propertyName, propertyConfig := range config.Properties {
					if propertyConfig.Default != nil {
						if (*data)[propertyName] == nil {
							(*data)[propertyName] = propertyConfig.Default
						}
					}
				}

				if config.Base == "User" {
					username := (*data)["username"]
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

					// TODO: Jhon Validate Email
					if username == nil || strings.TrimSpace(username.(string)) == "" {
						email := (*data)["email"]
						if email == nil || strings.TrimSpace(email.(string)) == "" {
							// TODO: Validate email
							return wst.CreateError(fiber.ErrBadRequest, "EMAIL_PRESENCE", fiber.Map{"message": "Invalid email", "codes": wst.M{"email": []string{"presence"}}}, "ValidationError")
						}
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
					hashed, err := bcrypt.GenerateFromPassword([]byte((*data)["password"].(string)), 10)
					if err != nil {
						return err
					}
					(*data)["password"] = string(hashed)

					if app.debug {
						fmt.Printf("Create User: ('%v', '%v')\n", (*data)["username"], (*data)["email"])
					}
				}

			} else {
				if config.Base == "User" {
					if (*data)["password"] != nil && (*data)["password"] != "" {
						log.Println("Update User password")
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
				return wst.CreateError(fiber.ErrBadRequest, "BAD_REQUEST", fiber.Map{"message": fmt.Sprintf("Deleted %v instances for %v", deletedCount, ctx.ModelID)}, "Error")
			}
			ctx.StatusCode = fiber.StatusNoContent
			ctx.Result = ""
			return nil
		}
		loadedModel.On("instance_delete", deleteByIdHandler)

	}
}

func handleFindMany(loadedModel *model.Model, ctx *model.EventContext) error {
	if loadedModel.App.Debug {
		fmt.Println("DEBUG: handleFindMany")
	}

	cursor := loadedModel.FindMany(ctx.Filter, ctx)

	//chunkGenerator := model.NewInstanceAChunkGenerator(loadedModel, cursor, "application/json")
	chunkGenerator := model.NewCursorChunkGenerator(loadedModel, cursor)
	switch cursor.(type) {
	case *model.ErrorCursor:
		return cursor.(*model.ErrorCursor).Error()
	}
	if cursor.(*model.ChannelCursor).Err == nil {
		ctx.StatusCode = fiber.StatusOK
	}
	ctx.Result = chunkGenerator
	return nil
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
		SwaggerHelper: func() swaggerhelper2.SwaggerHelper {
			return app.swaggerHelper
		},
	}
}

func fixRelations(loadedModel *model.Model) error {
	for relationName, relation := range *loadedModel.Config.Relations {

		if relation.Type == "" {
			return fmt.Errorf("relation %v.%v has no type", loadedModel.Name, relationName)
		}

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
	return nil
}
