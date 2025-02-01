package model

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
)

func (loadedModel *StatefulModel) RemoteMethod(handler func(context *EventContext) error, options RemoteMethodOptions) fiber.Router {
	if !loadedModel.Config.Public {
		loadedModel.App.Logger().Fatalf("Trying to register a remote method in the private model: %v, you may set \"public\": true in the %v.json file", loadedModel.Name, loadedModel.Name)
	}
	options.Name = strings.TrimSpace(options.Name)
	if options.Name == "" {
		loadedModel.App.Logger().Fatalf("Method name cannot be empty at the remote method in the model: %v, options: %v", loadedModel.Name, options)
	}
	if loadedModel.remoteMethodsMap[options.Name] != nil {
		loadedModel.App.Logger().Fatalf("Already registered a remote method with name '%v'", options.Name)
	}

	var http = options.Http
	path := http.Path
	verb := strings.ToLower(http.Verb)
	description := options.Description

	for _, arg := range options.Accepts {
		arg.Arg = strings.TrimSpace(arg.Arg)
		if arg.Arg == "" {
			loadedModel.App.Logger().Fatalf("Argument name cannot be empty in the remote method '%v'", options.Name)
		}
		if arg.Http.Source != "query" && arg.Http.Source != "body" {
			loadedModel.App.Logger().Fatalf("Argument '%v' in the remote method '%v' has an invalid 'in' value: '%v'", arg.Arg, options.Name, arg.Http.Source)
		}
	}

	_, err := loadedModel.Enforcer.AddRoleForUser(options.Name, "*")
	if err != nil {
		loadedModel.App.Logger().Fatalf("Error adding role '%v' for user '%v': %v", options.Name, "*", err)
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

	if !loadedModel.earlyDisabledMethods[options.Name] {
		pathParams := regexp.MustCompile(`:(\w+)`).FindAllString(path, -1)

		pathDef := createOpenAPIPathDef(loadedModel, description, pathParams)

		var schemaName string
		components := loadedModel.App.SwaggerHelper().GetComponents()
		if components["schemas"] != nil {
			schemas := components["schemas"].(wst.M)
			// find a schema by \w+\.ModelName
			re := regexp.MustCompile(`^\w+\.` + loadedModel.Name + `$`)
			for k := range schemas {
				if re.MatchString(k) {
					schemaName = k
					break
				}
			}
		}
		if schemaName == "" {
			schemaName = "models." + loadedModel.Name
		}

		if verb == "post" || verb == "put" || verb == "patch" {
			if options.Name == string(wst.OperationNameCreate) ||
				options.Name == string(wst.OperationNameUpdateAttributes) {
				assignOpenAPIRequestBody(pathDef, wst.M{
					"$ref": fmt.Sprintf("#/components/schemas/%s", schemaName),
				}, fiber.MIMEApplicationJSON)
			} else {
				assignOpenAPIRequestBody(pathDef, wst.M{
					"type": "object",
				}, fiber.MIMEApplicationJSON)
			}
		} else {
			params := createOpenAPIAdditionalParams(loadedModel, options)
			if len(params) > 0 {
				pathDef["parameters"] = params
			}
		}

		loadedModel.App.SwaggerHelper().AddPathSpec(fullPath, verb, pathDef, options.Name, schemaName)
		// clean up memory
		pathDef = nil
		runtime.GC()
	}

	loadedModel.remoteMethodsMap[options.Name] = createRemoteMethodOperationItem(loadedModel, handler, options)

	(*loadedModel.Router).Options(path, func(ctx *fiber.Ctx) error {
		ctx.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		ctx.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		ctx.Set("Access-Control-Allow-Origin", "*")
		return ctx.Status(fiber.StatusNoContent).SendString("")
	})

	return toInvoke(path, createFiberHandler(options, loadedModel, verb, path)).Name(loadedModel.Name + "." + options.Name)
}

var activeRequestsPerModel = make(map[string]int)
var activeRequestsMutex sync.RWMutex

func createFiberHandler(options RemoteMethodOptions, loadedModel *StatefulModel, verb string, path string) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		// Limit to 2 concurrent requests per model, new requests will be queued
		activeRequestsMutex.Lock()
		if _, ok := activeRequestsPerModel[loadedModel.Name]; !ok {
			activeRequestsPerModel[loadedModel.Name] = 0
		}
		for activeRequestsPerModel[loadedModel.Name] >= 2 {
			activeRequestsMutex.Unlock()
			time.Sleep(16 * time.Millisecond)
			activeRequestsMutex.Lock()
		}
		activeRequestsPerModel[loadedModel.Name]++
		activeRequestsMutex.Unlock()

		defer func() {
			activeRequestsMutex.Lock()
			activeRequestsPerModel[loadedModel.Name]--
			activeRequestsMutex.Unlock()
		}()

		eventContext := &EventContext{
			Ctx:    ctx,
			Remote: &options,
		}
		eventContext.Model = loadedModel
		err2 := loadedModel.HandleRemoteMethod(options.Name, eventContext)
		if err2 != nil {
			log.Printf("Error in remote method %v.%v (%v %v%v): %v\n", loadedModel.Name, options.Name, strings.ToUpper(verb), loadedModel.BaseUrl, path, err2.Error())
		}
		return err2
	}
}

func createRemoteMethodOperationItem(loadedModel *StatefulModel, handler func(context *EventContext) error, options RemoteMethodOptions) *OperationItem {
	operationItem := &OperationItem{
		Handler: handler,
		Options: options,
	}
	if loadedModel.earlyDisabledMethods[options.Name] {
		operationItem.disabled = true
	}
	return operationItem
}

func createOpenAPIAdditionalParams(loadedModel *StatefulModel, options RemoteMethodOptions) []wst.M {
	var params []wst.M
	for _, param := range options.Accepts {
		paramType := param.Type
		if paramType == "" {
			loadedModel.App.Logger().Fatalf("Argument '%v' in the remote method '%v' has an invalid 'type' value: '%v'", param.Arg, options.Name, paramType)
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

func assignOpenAPIRequestBody(pathDef wst.M, schema wst.M, contentType string) {
	pathDef["requestBody"] = wst.M{
		"description": "data",
		"required":    true,
		"content": wst.M{
			contentType: wst.M{
				"schema": schema,
			},
		},
	}
}

func assignOpenAPIRequestQueryParams(pathDef wst.M, inputSchema wst.M, components wst.M) {
	var component wst.M
	if inputSchema["$ref"] != nil {
		schemaName := strings.Split(inputSchema["$ref"].(string), "/")[3]
		component = components["schemas"].(wst.M)[schemaName].(wst.M)
	} else {
		component = inputSchema
	}
	if component["properties"] != nil {
		params := make([]wst.M, 0)
		for key, value := range component["properties"].(wst.M) {
			prop := value.(wst.M)
			paramType := prop["type"].(string)
			if paramType == "date" {
				paramType = "string"
			}
			params = append(params, wst.M{
				"name":        key,
				"in":          "query",
				"description": key,
				"required":    prop["required"],
				"schema": wst.M{
					"type": paramType,
				},
			})
		}
		pathDef["parameters"] = params
	}
}

func assignOpenAPIResponse(pathDef wst.M, schema wst.M) {
	pathDef["overrideResponses"] = wst.M{
		"200": wst.M{
			"description": "OK",
			"content": wst.M{
				"application/json": wst.M{
					"schema": schema,
				},
			},
		},
	}
}

func createOpenAPIPathDef(loadedModel *StatefulModel, description string, rawPathParams []string) wst.M {
	pathDef := wst.M{
		"modelName": loadedModel.Name,
		"summary":   description,
	}
	if len(rawPathParams) > 0 {
		pathDef["rawPathParams"] = append([]string{}, rawPathParams...)
	}
	return pathDef
}

func (loadedModel *StatefulModel) HandleRemoteMethod(name string, eventContext *EventContext) error {

	operationItem := loadedModel.remoteMethodsMap[name]

	if operationItem == nil {
		return errors.New(fmt.Sprintf("Method '%v' not found", name))
	}

	if operationItem.disabled {
		return fiber.ErrNotFound
	}

	c := eventContext.Ctx
	options := operationItem.Options
	handler := operationItem.Handler

	token, err := eventContext.GetBearer(loadedModel)
	if err != nil {
		return err
	}

	action := options.Name

	if loadedModel.App.Debug {
		log.Println(fmt.Sprintf("[DEBUG] Check auth for %v.%v (%v %v)", loadedModel.Name, options.Name, c.Method(), c.Path()))
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

	for k, v := range c.Queries() {
		(*eventContext.Query)[k] = v
	}

	shouldHaveBody := strings.ToLower(options.Http.Verb) == "post" || strings.ToLower(options.Http.Verb) == "put" || strings.ToLower(options.Http.Verb) == "patch"
	if shouldHaveBody {
		// if application/json
		if wst.CleanContentType(c.Get("Content-Type")) == "application/json" {
			var data wst.M
			err := eventContext.Ctx.BodyParser(&data)
			if err != nil {
				return wst.CreateError(fiber.ErrBadRequest, "INVALID_BODY", fiber.Map{"message": err.Error()}, "ValidationError")
			}
			eventContext.Data = &data
		} else if /*application/x-www-form-urlencoded*/ wst.CleanContentType(c.Get("Content-Type")) == "application/x-www-form-urlencoded" {
			rawBodyBytes := c.BodyRaw()
			rawBody := string(rawBodyBytes)
			parts := strings.Split(rawBody, "&")
			for _, part := range parts {
				kv := strings.Split(part, "=")
				(*eventContext.Data)[kv[0]] = kv[1]
				for i := 2; i < len(kv); i++ {
					(*eventContext.Data)[kv[0]] = (*eventContext.Data)[kv[0]].(string) + "=" + kv[i]
				}
			}
		} else if /*form-data*/ strings.Contains(wst.CleanContentType(c.Get("Content-Type")), "multipart/form-data") {
			form, err := c.MultipartForm()
			if err != nil {
				return wst.CreateError(fiber.ErrBadRequest, "INVALID_BODY", fiber.Map{"message": err.Error()}, "ValidationError")
			}
			for k, v := range form.Value {
				(*eventContext.Data)[k] = v
			}
		} else {
			if c.Get("Content-Length", "0") == "0" || (c.Get("Content-Length") == "" && c.Get("Transfer-Encoding") == "") {
				// no content
			} else {
				return wst.CreateError(fiber.ErrUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", fiber.Map{"message": "Unsupported media type"}, "ValidationError")
			}
		}
	} else {
		if c.Get("Content-Length", "0") != "0" {
			return wst.CreateError(fiber.ErrBadRequest, "INVALID_BODY", fiber.Map{"message": "Body not expected"}, "ValidationError")
		}
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
				param = 0.0
				if regexp.MustCompile(`^-\d+(\.\d+)?$`).MatchString(paramSt) {
					param, err = strconv.ParseFloat(paramSt, 64)
				}
				if err != nil {
					return wst.CreateError(fiber.ErrBadRequest, "INVALID_NUMBER", fiber.Map{"message": err.Error()}, "ValidationError")
				}
				break
			}
			(*eventContext.Query)[key] = param

			if paramDef.Arg == "filter" {
				filterSt := (*eventContext.Query)[key].(string)
				filterMap := ParseFilter(filterSt)
				if filterSt != "" && filterMap == nil {
					return wst.CreateError(fiber.ErrBadRequest, "INVALID_FILTER", fiber.Map{"message": "Invalid filter"}, "ValidationError")
				}

				eventContext.Filter = filterMap
				continue
			}

			foundSomeQuery = true

		}
	}
	replaced, err := datasource.ReplaceObjectIds(eventContext.Data)
	if err != nil {
		return err
	}
	eventContext.Data = replaced.(*wst.M)
	if foundSomeQuery {
		replaced, err = datasource.ReplaceObjectIds(eventContext.Query)
		if err != nil {
			return err
		}
		for k, v := range *replaced.(*wst.M) {
			(*eventContext.Query)[k] = v
		}
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
		if v, ok := eventContext.Result.(*wst.M); ok {
			if v != nil {
				eventContext.Result = *v
			}
		}
		switch eventContext.Result.(type) {
		case wst.M:
			if eventContext.Result.(wst.M)["<wst.NilMap>"] == 1 {
				eventContext.Ctx.Set("Content-Type", "application/json")
				return eventContext.Ctx.Status(eventContext.StatusCode).Send([]byte{'n', 'u', 'l', 'l'})
			}
			if eventContext.Ephemeral != nil {
				for k, v := range *eventContext.Ephemeral {
					(eventContext.Result.(wst.M))[k] = v
				}
			}
			return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
		case wst.A, *wst.A, fiber.Map, *fiber.Map, map[string]interface{}, *map[string]interface{}, []interface{}, *[]interface{}, int, int32, int64, float32, float64, bool:
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

				if eventContext.Result != nil {

					// check struct
					if reflect.TypeOf(eventContext.Result).Kind() == reflect.Struct {
						return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
					}

					// check slice of even struct or pointer to struct
					if reflect.TypeOf(eventContext.Result).Kind() == reflect.Slice {
						if reflect.TypeOf(eventContext.Result).Elem().Kind() == reflect.Struct {
							return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
						}
						if reflect.TypeOf(eventContext.Result).Elem().Kind() == reflect.Ptr {
							if reflect.TypeOf(reflect.New(reflect.TypeOf(eventContext.Result).Elem().Elem()).Interface()).Kind() == reflect.Struct {
								return eventContext.Ctx.Status(eventContext.StatusCode).JSON(eventContext.Result)
							}
						}
					}
				}

				fmt.Printf("Unknown type: %T after remote method %v\n", eventContext.Result, name)
				eventContext.Handled = false
			}
		}
	}
	resp := eventContext.Ctx.Response()
	if resp.StatusCode() == 0 {
		fmt.Printf("[WARNING] No result found after remote method %v\n", name)
		return eventContext.Ctx.Status(fiber.StatusNoContent).SendString("")
	}
	return nil
}

func (loadedModel *StatefulModel) DisableRemoteOperation(operationName wst.OperationName) {
	if existing, ok := loadedModel.remoteMethodsMap[string(operationName)]; ok {
		verb := existing.Options.Http.Verb
		path := existing.Options.Http.Path

		existing.disabled = true

		// unregister from swagger
		app := loadedModel.App

		path = strings.TrimPrefix(path, "/")

		app.SwaggerHelper().RemovePathSpec(loadedModel.BaseUrl+"/"+path, strings.ToLower(verb))
	} else {
		loadedModel.earlyDisabledMethods[string(operationName)] = true
	}
}
