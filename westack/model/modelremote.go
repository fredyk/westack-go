package model

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
)

func (loadedModel *Model) SendError(ctx *fiber.Ctx, err error) error {
	switch err.(type) {
	case *wst.WeStackError:
		errorName := err.(*wst.WeStackError).Name
		if errorName == "" {
			errorName = "Error"
		}
		return ctx.Status((err).(*wst.WeStackError).FiberError.Code).JSON(fiber.Map{
			"error": fiber.Map{
				"statusCode": (err).(*wst.WeStackError).FiberError.Code,
				"name":       errorName,
				"code":       err.(*wst.WeStackError).Code,
				"error":      err.(*wst.WeStackError).FiberError.Error(),
				"message":    (err.(*wst.WeStackError).Details)["message"],
				"details":    err.(*wst.WeStackError).Details,
			},
		})
	default:
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"statusCode": 500,
				"name":       "InternalServerError",
				"code":       "INTERNAL_SERVER_ERROR",
				"error":      err.Error(),
				"message":    err.Error(),
			},
		})
	}
}

func (loadedModel *Model) RemoteMethod(handler func(context *EventContext) error, options RemoteMethodOptions) fiber.Router {
	if !loadedModel.Config.Public {
		panic(fmt.Sprintf("Trying to register a remote method in the private model: %v, you may set \"public\": true in the %v.json file", loadedModel.Name, loadedModel.Name))
	}
	options.Name = strings.TrimSpace(options.Name)
	if options.Name == "" {
		panic("Method name cannot be empty")
	}
	if loadedModel.remoteMethodsMap[options.Name] != nil {
		panic(fmt.Sprintf("Already registered a remote method with name '%v'", options.Name))
	}

	var http = options.Http
	path := http.Path
	verb := strings.ToLower(http.Verb)
	description := options.Description

	for _, arg := range options.Accepts {
		arg.Arg = strings.TrimSpace(arg.Arg)
		if arg.Arg == "" {
			panic(fmt.Sprintf("Argument name cannot be empty in the remote method '%v'", options.Name))
		}
		if arg.Http.Source != "query" && arg.Http.Source != "body" {
			panic(fmt.Sprintf("Argument '%v' in the remote method '%v' has an invalid 'in' value: '%v'", arg.Arg, options.Name, arg.Http.Source))
		}
	}

	_, err := loadedModel.Enforcer.AddRoleForUser(options.Name, "*")
	if err != nil {
		panic(err)
	}

	var toInvoke func(string, ...fiber.Handler) fiber.Router
	operation := ""

	switch verb {
	case "get":
		toInvoke = (*loadedModel.Router).Get
		operation = "Finds"
	case "options":
		toInvoke = (*loadedModel.Router).Options
		operation = "Gets options for"
	case "head":
		toInvoke = (*loadedModel.Router).Head
		operation = "Checks"
	case "post":
		toInvoke = (*loadedModel.Router).Post
		operation = "Creates"
	case "put":
		toInvoke = (*loadedModel.Router).Put
		operation = "Replaces"
	case "patch":
		toInvoke = (*loadedModel.Router).Patch
		operation = "Updates attributes in"
	case "delete":
		toInvoke = (*loadedModel.Router).Delete
		operation = "Deletes"
	}

	fullPath := loadedModel.BaseUrl + "/" + path
	fullPath = regexp.MustCompile("//+").ReplaceAllString(fullPath, "/")
	fullPath = regexp.MustCompile(`:(\w+)`).ReplaceAllString(fullPath, "{$1}")

	if (*loadedModel.App.SwaggerPaths())[fullPath] == nil {
		(*loadedModel.App.SwaggerPaths())[fullPath] = wst.M{}
	}

	if description == "" {
		description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
	}

	pathDef := wst.M{
		//"description": description,
		//"consumes": []string{
		//	"*/*",
		//},
		//"produces": []string{
		//	"application/json",
		//},
		"tags": []string{
			loadedModel.Name,
		},
		//"requestBody": requestBody,
		"summary": description,
		"security": []fiber.Map{
			{"bearerAuth": []string{}},
		},
		"responses": wst.M{
			"200": wst.M{
				"description": "OK",
				"content": wst.M{
					"application/json": wst.M{
						"schema": wst.M{
							"type": "object",
						},
					},
				},
				//"$ref": "#/components/schemas/" + loadedModel.Config.Name,
				//"schema": wst.M{
				//	"type":                 "object",
				//	"additionalProperties": true,
				//},
			},
		},
	}

	pathParams := regexp.MustCompile(`:(\w+)`).FindAllString(path, -1)

	params := make([]wst.M, len(pathParams))

	for idx, param := range pathParams {
		params[idx] = wst.M{
			"name":     strings.TrimPrefix(param, ":"),
			"in":       "path",
			"required": true,
			"schema": wst.M{
				"type": "string",
			},
		}
	}

	(*loadedModel.App.SwaggerPaths())[fullPath][verb] = pathDef

	if verb == "post" || verb == "put" || verb == "patch" {
		pathDef["requestBody"] = wst.M{
			"description": "data",
			"required":    true,
			//"name":        "data",
			//"in":          "body",
			//"schema": wst.M{
			//	"type": "object",
			//},
			"content": wst.M{
				"application/json": wst.M{
					"schema": wst.M{
						"type": "object",
					},
				},
			},
		}
	} else {

		for _, param := range options.Accepts {
			paramType := param.Type
			if paramType == "" {
				panic(fmt.Sprintf("Argument '%v' in the remote method '%v' has an invalid 'type' value: '%v'", param.Arg, options.Name, paramType))
			}
			paramDescription := param.Description
			if paramType == "date" {
				paramType = "string"
				paramDescription += " (format: ISO8601)"
			}
			params = append(params, wst.M{
				"name":        param.Arg,
				"in":          param.Http.Source,
				"description": paramDescription,
				"required":    param.Required,
				"schema": wst.M{
					"type": paramType,
				},
			})
		}

	}

	if len(params) > 0 {
		pathDef["parameters"] = params
	}

	(*loadedModel.App.SwaggerPaths())[fullPath][verb] = pathDef

	loadedModel.remoteMethodsMap[options.Name] = &OperationItem{
		Handler: handler,
		Options: options,
	}

	return toInvoke(path, func(ctx *fiber.Ctx) error {
		eventContext := &EventContext{
			Ctx:    ctx,
			Remote: &options,
		}
		err2 := loadedModel.HandleRemoteMethod(options.Name, eventContext)
		if err2 != nil {

			if err2 == fiber.ErrUnauthorized {
				err2 = wst.CreateError(fiber.ErrUnauthorized, "UNAUTHORIZED", fiber.Map{"message": "Unauthorized"}, "Error")
			}

			log.Printf("Error in remote method %v.%v (%v %v%v): %v\n", loadedModel.Name, options.Name, strings.ToUpper(verb), loadedModel.BaseUrl, path, err2.Error())
			return loadedModel.SendError(eventContext.Ctx, err2)
		}
		return nil
	})
}

func (loadedModel *Model) HandleRemoteMethod(name string, eventContext *EventContext) error {

	operationItem := loadedModel.remoteMethodsMap[name]

	if operationItem == nil {
		return errors.New(fmt.Sprintf("Method '%v' not found", name))
	}

	c := eventContext.Ctx
	options := operationItem.Options
	handler := operationItem.Handler

	err, token := eventContext.GetBearer(loadedModel)
	if err != nil {
		return err
	}

	action := options.Name

	if loadedModel.App.Debug {
		log.Println(fmt.Sprintf("DEBUG: Check auth for %v.%v (%v %v)", loadedModel.Name, options.Name, c.Method(), c.Path()))
	}

	objId := "*"
	if eventContext.ModelID != nil {
		objId = GetIDAsString(eventContext.ModelID)
	} else {
		objId = c.Params("id")
		if objId == "" {
			objId = "*"
		}
	}

	err, allowed := loadedModel.EnforceEx(token, objId, action, eventContext)
	if err != nil {
		return err
	}
	if !allowed {
		return fiber.ErrUnauthorized
	}

	eventContext.Bearer = token

	eventContext.Data = &wst.M{}
	eventContext.Query = &wst.M{}

	if strings.ToLower(options.Http.Verb) == "post" || strings.ToLower(options.Http.Verb) == "put" || strings.ToLower(options.Http.Verb) == "patch" {
		var data wst.M
		//bytes := eventContext.Ctx.Body()
		//if len(bytes) > 0 {
		//err := json.Unmarshal(bytes, &data)
		err := eventContext.Ctx.BodyParser(&data)
		if err != nil {
			return wst.CreateError(fiber.ErrBadRequest, "INVALID_BODY", fiber.Map{"message": err.Error()}, "ValidationError")
		}
		eventContext.Data = &data
		//} else {
		//	// Empty body is allowed
		//}
	}

	foundSomeQuery := false
	for _, paramDef := range options.Accepts {
		key := paramDef.Arg
		if paramDef.Http.Source == "body" {

			// Already parsed. Only used for OpenAPI Description

		} else if paramDef.Http.Source == "query" {

			var param interface{}
			paramSt := c.Query(key, "")
			switch paramDef.Type {
			case "string":
				param = paramSt
				break
			case "date":
				param, err = wst.ParseDate(paramSt)
				if err != nil {
					return wst.CreateError(fiber.ErrBadRequest, "INVALID_DATE", fiber.Map{"message": err.Error()}, "ValidationError")
				}
				break
			case "number":
				param, err = strconv.ParseFloat(paramSt, 64)
				if err != nil {
					return wst.CreateError(fiber.ErrBadRequest, "INVALID_NUMBER", fiber.Map{"message": err.Error()}, "ValidationError")
				}
				break
			}
			(*eventContext.Query)[key] = param

			if paramDef.Arg == "filter" {
				filterSt := (*eventContext.Query)[key].(string)
				filterMap := ParseFilter(filterSt)

				eventContext.Filter = filterMap
				continue
			}

			foundSomeQuery = true

		}
	}
	eventContext.Data = datasource.ReplaceObjectIds(eventContext.Data).(*wst.M)
	if foundSomeQuery {
		eventContext.Query = datasource.ReplaceObjectIds(eventContext.Query).(*wst.M)
	}

	err = handler(eventContext)
	if err != nil {
		return err
	}
	if eventContext.Result != nil || eventContext.StatusCode != 0 {
		if eventContext.StatusCode == 0 {
			eventContext.StatusCode = fiber.StatusOK
		} else if eventContext.StatusCode == fiber.StatusNoContent {
			return eventContext.Ctx.Status(fiber.StatusNoContent).SendString("")
		}
		switch eventContext.Result.(type) {
		case wst.M, *wst.M, map[string]interface{}, *map[string]interface{}, wst.A, *wst.A, []interface{}, *[]interface{}, int, int32, int64, float32, float64, bool:
			return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
		case string:
			return eventContext.Ctx.Status(eventContext.StatusCode).SendString(eventContext.Result.(string))
		case []byte:
			return eventContext.Ctx.Status(eventContext.StatusCode).Send(eventContext.Result.([]byte))
		default:
			fmt.Printf("Unknown type: %T after remote method %v\n", eventContext.Result, name)
		}
	}
	resp := eventContext.Ctx.Response()
	if resp.StatusCode() == 0 {
		fmt.Printf("WARNING: No result found after remote method %v\n", name)
		return eventContext.Ctx.Status(fiber.StatusNoContent).Type("application/json").Send([]byte("null"))
	}
	return nil
}
