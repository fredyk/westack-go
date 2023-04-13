package model

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"runtime"
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

	router := *loadedModel.Router
	switch verb {
	case "get":
		toInvoke = router.Get
		operation = "Finds"
	case "options":
		toInvoke = router.Options
		operation = "Gets options for"
	case "head":
		toInvoke = router.Head
		operation = "Checks"
	case "post":
		toInvoke = router.Post
		operation = "Creates"
	case "put":
		toInvoke = router.Put
		operation = "Replaces"
	case "patch":
		toInvoke = router.Patch
		operation = "Updates attributes in"
	case "delete":
		toInvoke = router.Delete
		operation = "Deletes"
	}

	fullPath := loadedModel.BaseUrl + "/" + path
	fullPath = regexp.MustCompile("//+").ReplaceAllString(fullPath, "/")
	fullPath = regexp.MustCompile(`:(\w+)`).ReplaceAllString(fullPath, "{$1}")

	if description == "" {
		description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
	}

	pathParams := regexp.MustCompile(`:(\w+)`).FindAllString(path, -1)

	pathDef := createOpenAPIPathDef(loadedModel, description, pathParams)

	if verb == "post" || verb == "put" || verb == "patch" {
		assignOpenAPIRequestBody(pathDef)
	} else {
		params := createOpenAPIAdditionalParams(options)
		if len(params) > 0 {
			pathDef["parameters"] = params
		}
	}

	loadedModel.App.SwaggerHelper().AddPathSpec(fullPath, verb, pathDef)
	// clean up memory
	pathDef = nil
	runtime.GC()

	loadedModel.remoteMethodsMap[options.Name] = createRemoteMethodOperationItem(handler, options)

	return toInvoke(path, createFiberHandler(options, loadedModel, verb, path)).Name(loadedModel.Name + "." + options.Name)
}

func createFiberHandler(options RemoteMethodOptions, loadedModel *Model, verb string, path string) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		eventContext := &EventContext{
			Ctx:    ctx,
			Remote: &options,
		}
		eventContext.Model = loadedModel
		err2 := loadedModel.HandleRemoteMethod(options.Name, eventContext)
		if err2 != nil {

			if err2 == fiber.ErrUnauthorized {
				err2 = wst.CreateError(fiber.ErrUnauthorized, "UNAUTHORIZED", fiber.Map{"message": "Unauthorized"}, "Error")
			}

			log.Printf("Error in remote method %v.%v (%v %v%v): %v\n", loadedModel.Name, options.Name, strings.ToUpper(verb), loadedModel.BaseUrl, path, err2.Error())
			return loadedModel.SendError(eventContext.Ctx, err2)
		}
		return nil
	}
}

func createRemoteMethodOperationItem(handler func(context *EventContext) error, options RemoteMethodOptions) *OperationItem {
	return &OperationItem{
		Handler: handler,
		Options: options,
	}
}

func createOpenAPIAdditionalParams(options RemoteMethodOptions) []wst.M {
	var params []wst.M
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
	return params
}

func assignOpenAPIRequestBody(pathDef wst.M) {
	pathDef["requestBody"] = wst.M{
		"description": "data",
		"required":    true,
		"content": wst.M{
			"application/json": wst.M{
				"schema": wst.M{
					"type": "object",
				},
			},
		},
	}
}

func createOpenAPIPathDef(loadedModel *Model, description string, rawPathParams []string) wst.M {
	pathDef := wst.M{
		"modelName": loadedModel.Name,
		"summary":   description,
	}
	if len(rawPathParams) > 0 {
		pathDef["rawPathParams"] = append([]string{}, rawPathParams...)
	}
	return pathDef
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
		//Err := json.Unmarshal(bytes, &data)
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
		eventContext.Handled = true
		if eventContext.StatusCode == 0 {
			eventContext.StatusCode = fiber.StatusOK
		} else if eventContext.StatusCode == fiber.StatusNoContent {
			return eventContext.Ctx.Status(fiber.StatusNoContent).SendString("")
		}
		switch eventContext.Result.(type) {
		case wst.M:
			if eventContext.Result.(wst.M)["<wst.NilMap>"] == 1 {
				eventContext.Ctx.Set("Content-Type", "application/json")
				return eventContext.Ctx.Status(eventContext.StatusCode).Send([]byte{'n', 'u', 'l', 'l'})
			}
			return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
		case *wst.M, wst.A, *wst.A, fiber.Map, *fiber.Map, map[string]interface{}, *map[string]interface{}, []interface{}, *[]interface{}, int, int32, int64, float32, float64, bool:
			return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
		case string:
			return eventContext.Ctx.Status(eventContext.StatusCode).SendString(eventContext.Result.(string))
		case []byte:
			return eventContext.Ctx.Status(eventContext.StatusCode).Send(eventContext.Result.([]byte))
		default:
			if resultAsGenerator, ok := eventContext.Result.(ChunkGenerator); ok {

				finalContentType := resultAsGenerator.ContentType()
				if finalContentType == "" {
					finalContentType = "application/octet-stream"
				}

				eventContext.Ctx.Set("Content-Type", finalContentType)
				eventContext.Ctx.Set("Transfer-Encoding", "chunked")
				eventContext.Ctx.Response().Header.Set("Transfer-Encoding", "chunked")

				return eventContext.Ctx.SendStream(resultAsGenerator.Reader(eventContext), -1)

			} else {
				fmt.Printf("Unknown type: %T after remote method %v\n", eventContext.Result, name)
				eventContext.Handled = false
			}
		}
	}
	resp := eventContext.Ctx.Response()
	if resp.StatusCode() == 0 {
		fmt.Printf("WARNING: No result found after remote method %v\n", name)
		return eventContext.Ctx.Status(fiber.StatusNoContent).SendString("")
	}
	return nil
}
