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

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/model"
)

func (app *WeStack) loadNotFoundRoutes() {
	for _, entry := range *app.modelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.debug {
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

func (app *WeStack) loadModelsFixedRoutes() {
	for _, entry := range *app.modelRegistry {
		loadedModel := entry

		e, err := casbin.NewEnforcer(*loadedModel.CasbinModel, *loadedModel.CasbinAdapter, app.debug)
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

						} else {
							//log.Printf("Invalid foreign key in relation %v.%v (%v.%v --> %v.%v)\n", loadedModel.Name, key, loadedModel.Name, r.ForeignKey, r.Model, r.PrimaryKey)
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

		if app.debug {
			loadedModel.CasbinModel.PrintModel()
		}

		if app.Viper.GetBool("casbin.dumpModels") {
			text := loadedModel.CasbinModel.ToText()
			err = os.WriteFile(fmt.Sprintf("common/models/%v.casbin.dump.conf", loadedModel.Name), []byte(text), os.ModePerm)
			if err != nil {
				panic(err)
			}
		}

		if !loadedModel.Config.Public {
			if app.debug {
				log.Println("WARNING: Model", loadedModel.Name, "is not public")
			}
			continue
		}

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

		if loadedModel.Config.Base == "User" {

			loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
				return handleEvent(eventContext, loadedModel, "login")
			}, model.RemoteMethodOptions{
				Name:        "login",
				Description: "Logins a user",
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

				err, token := eventContext.GetBearer(loadedModel)
				if err != nil {
					return err
				}
				eventContext.ModelID = token.User.Id
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
				fmt.Println("verify user ", eventContext.Bearer.User.Id)
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
				userId := eventContext.Bearer.User.Id
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
					redirectToUrl := eventContext.Ctx.Query("redirect_uri")
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

		}
	}
}

func (app *WeStack) loadModelsDynamicRoutes() {
	for _, entry := range *app.modelRegistry {
		loadedModel := entry
		if !loadedModel.Config.Public {
			if app.debug {
				log.Println("WARNING: Model", loadedModel.Name, "is not public")
			}
			continue
		}

		if app.debug {
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
	}
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
