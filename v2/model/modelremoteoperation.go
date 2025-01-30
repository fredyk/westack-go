package model

import (
	"fmt"
	"mime/multipart"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/lib/swaggerhelper"
	"github.com/gofiber/fiber/v2"
)

func BindRemoteOperationWithContext[T any, R any](loadedModel *StatefulModel, handler func(req *RemoteOperationReq[T]) (R, error), options *RemoteOperationOptions) fiber.Router {

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

	path := options.Path
	description := options.Description
	verb := options.Verb

	isVerbWithBody := verb == "post" || verb == "put" || verb == "patch"

	fmt.Printf("[INFO] Binding remote operation %s at %s %s%s\n", options.Name, strings.ToUpper(verb), loadedModel.BaseUrl, path)

	_, err := loadedModel.Enforcer.AddRoleForUser(options.Name, "*")
	if err != nil {
		loadedModel.App.Logger().Fatalf("Error adding role '%v' for user '%v': %v", options.Name, "*", err)
	}

	var toInvoke func(string, ...fiber.Handler) fiber.Router

	router := *loadedModel.Router
	switch verb {
	case "get":
		toInvoke = router.Get
	case "options":
		toInvoke = router.Options
	case "head":
		toInvoke = router.Head
	case "post":
		toInvoke = router.Post
	case "put":
		toInvoke = router.Put
	case "patch":
		toInvoke = router.Patch
	case "delete":
		toInvoke = router.Delete
	}
	operation := ""

	fullPath := loadedModel.BaseUrl + "/" + path
	fullPath = regexp.MustCompile("//+").ReplaceAllString(fullPath, "/")
	fullPath = regexp.MustCompile(`:(\w+)`).ReplaceAllString(fullPath, "{$1}")

	if !loadedModel.earlyDisabledMethods[options.Name] {
		if description == "" {
			description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
		}

		pathParams := regexp.MustCompile(`:(\w+)`).FindAllString(path, -1)

		inputSchemaName := swaggerhelper.RegisterGenericComponent[T](loadedModel.App.SwaggerHelper())
		resultSchemaName := swaggerhelper.RegisterGenericComponent[R](loadedModel.App.SwaggerHelper())

		var inputSchema wst.M
		var resultSchema wst.M
		if inputSchemaName == "object" {
			inputSchema = wst.M{
				"type": "object",
			}
		} else {
			inputSchema = wst.M{
				"$ref": fmt.Sprintf("#/components/schemas/%v", inputSchemaName),
			}
		}
		if resultSchemaName == "object" {
			resultSchema = wst.M{
				"type": "object",
			}
		} else {
			resultSchema = wst.M{
				"$ref": fmt.Sprintf("#/components/schemas/%v", resultSchemaName),
			}
		}

		pathDef := createOpenAPIPathDef(loadedModel, description, pathParams)

		if isVerbWithBody {
			assignOpenAPIRequestBody(pathDef, inputSchema, options.ContentType)
		} else {
			assignOpenAPIRequestQueryParams(pathDef, inputSchema, loadedModel.App.SwaggerHelper().GetComponents())
		}
		assignOpenAPIResponse(pathDef, resultSchema)

		loadedModel.App.SwaggerHelper().AddPathSpec(fullPath, verb, pathDef, options.Name, loadedModel.Name)
		// clean up memory
		pathDef = nil
		runtime.GC()
	}

	remoteMethodOptions := RemoteMethodOptions{
		Name: options.Name,
	}

	sortRateLimitsByTimePeriod(options.RateLimits)

	handlerWrapper := func(ctx *EventContext) error {

		for _, rl := range options.RateLimits {
			if !rl.Allow(ctx) {
				return ctx.Ctx.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
			}
		}

		req := &RemoteOperationReq[T]{
			Ctx: ctx,
		}

		if options.StrictContentType && wst.CleanContentType(string(ctx.Ctx.Request().Header.ContentType())) != options.ContentType {
			return ctx.Ctx.Status(fiber.StatusBadRequest).SendString(
				fmt.Sprintf("invalid content type %v, expected %v", ctx.Ctx.Get("Content-Type"),
					options.ContentType))
		}

		pointerToInput := &req.Input

		if verb == "get" {
			err := ctx.Ctx.QueryParser(pointerToInput)
			if err != nil {
				return ctx.Ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
			}
		} else {
			err := ctx.Ctx.BodyParser(pointerToInput)
			if err != nil {
				return ctx.Ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
			}
		}

		if options.ContentType == fiber.MIMEMultipartForm {
			// parse form data files
			structValueAddr := reflect.ValueOf(pointerToInput)
			structValue := structValueAddr.Elem()
			fieldCount := structValue.NumField()
			for i := 0; i < fieldCount; i++ {
				field := structValue.Field(i)
				if field.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)).Elem() {
					structField := reflect.TypeOf(pointerToInput).Elem().Field(i)
					tagged := structField.Tag.Get("json")
					if tagged == "" {
						tagged = structField.Name
					} else {
						tagged = strings.Split(tagged, ",")[0]
					}
					file, err := ctx.Ctx.FormFile(tagged)
					if err != nil {
						return ctx.Ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
					}
					field.Set(reflect.ValueOf(file).Elem())
				}
			}
		}

		result, err := handler(req)
		if err != nil {
			return ctx.Ctx.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		ctx.Result = result

		if ctx.Ctx.Response().StatusCode() != 0 {
			ctx.StatusCode = ctx.Ctx.Response().StatusCode()
		}

		return nil
	}

	remoteMethodOptions.Http.Verb = verb
	loadedModel.remoteMethodsMap[options.Name] = createRemoteMethodOperationItem(loadedModel, handlerWrapper, remoteMethodOptions)

	(*loadedModel.Router).Options(path, func(ctx *fiber.Ctx) error {
		ctx.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		ctx.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		ctx.Set("Access-Control-Allow-Origin", "*")
		return ctx.Status(fiber.StatusNoContent).SendString("")
	})

	return toInvoke(path, createFiberHandler(remoteMethodOptions, loadedModel, verb, path)).Name(loadedModel.Name + "." + remoteMethodOptions.Name)

}

func sortRateLimitsByTimePeriod(rateLimits []*RateLimit) {
	// sort rate limits by largest time period first
	slices.SortFunc(rateLimits, func(a *RateLimit, b *RateLimit) int {
		if a.TimePeriod < b.TimePeriod {
			return 1
		}
		if a.TimePeriod > b.TimePeriod {
			return -1
		}
		return 0
	})
}

func BindRemoteOperationWithOptions[T any, R any](loadedModel *StatefulModel, handler func(req T) (R, error), options *RemoteOperationOptions) fiber.Router {
	if options.Name == "" {
		options.Name = getFunctionName(handler)
	}
	if options.Path == "" {
		options.Path = fmt.Sprintf("/hooks/%s", wst.DashedCase(options.Name))
	}
	if options.Description == "" {
		options.Description = fmt.Sprintf("Invokes %s on %s", options.Name, loadedModel.Name)
	}
	if options.ContentType == "" {
		options.ContentType = fiber.MIMEApplicationJSON
	}
	if options.Verb == "" {
		options.Verb = "post"
	}
	return BindRemoteOperationWithContext[T, R](loadedModel, func(req *RemoteOperationReq[T]) (R, error) {
		return handler(req.Input)
	}, options)
}

func getFunctionName(fn interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	splt := strings.Split(name, ".")
	return splt[len(splt)-1]
}

func BindRemoteOperation[T any, R any](loadedModel *StatefulModel, handler func(req T) (R, error)) fiber.Router {
	return BindRemoteOperationWithOptions[T, R](loadedModel, handler, &RemoteOperationOptions{})
}

func RemoteOptions() *RemoteOperationOptions {
	return &RemoteOperationOptions{
		ContentType: fiber.MIMEApplicationJSON,
	}
}

func (options *RemoteOperationOptions) WithName(name string) *RemoteOperationOptions {
	options.Name = name
	return options
}

func (options *RemoteOperationOptions) WithPath(path string) *RemoteOperationOptions {
	options.Path = path
	return options
}

func (options *RemoteOperationOptions) WithVerb(verb string) *RemoteOperationOptions {
	options.Verb = verb
	return options
}

func (options *RemoteOperationOptions) WithContentType(contentType string) *RemoteOperationOptions {
	options.ContentType = contentType
	options.StrictContentType = true
	return options
}

func (options *RemoteOperationOptions) WithRateLimits(rateLimits ...*RateLimit) *RemoteOperationOptions {
	options.RateLimits = rateLimits
	return options
}
