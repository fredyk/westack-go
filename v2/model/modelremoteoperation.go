package model

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/lib/swaggerhelper"
	"github.com/gofiber/fiber/v2"
)

func BindRemoteOperationWithContext[T any, R any](loadedModel *StatefulModel, handler func(req *RemoteOperationReq[T]) (R, error), options RemoteOperationOptions) fiber.Router {

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
	verb := "post"

	fmt.Printf("[INFO] Binding remote operation %s at %s %s%s\n", options.Name, strings.ToUpper(verb), loadedModel.BaseUrl, path)

	_, err := loadedModel.Enforcer.AddRoleForUser(options.Name, "*")
	if err != nil {
		loadedModel.App.Logger().Fatalf("Error adding role '%v' for user '%v': %v", options.Name, "*", err)
	}

	toInvoke := (*loadedModel.Router).Post
	operation := ""

	fullPath := loadedModel.BaseUrl + "/" + path
	fullPath = regexp.MustCompile("//+").ReplaceAllString(fullPath, "/")
	fullPath = regexp.MustCompile(`:(\w+)`).ReplaceAllString(fullPath, "{$1}")

	if description == "" {
		description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
	}

	pathParams := regexp.MustCompile(`:(\w+)`).FindAllString(path, -1)

	inputSchemaName := swaggerhelper.RegisterGenericComponent[T](loadedModel.App.SwaggerHelper())
	resultSchemaName := swaggerhelper.RegisterGenericComponent[R](loadedModel.App.SwaggerHelper())

	inputSchema := wst.M{
		"$ref": fmt.Sprintf("#/components/schemas/%v", inputSchemaName),
	}
	resultSchema := wst.M{
		"$ref": fmt.Sprintf("#/components/schemas/%v", resultSchemaName),
	}

	pathDef := createOpenAPIPathDef(loadedModel, description, pathParams)
	assignOpenAPIRequestBody(pathDef, inputSchema)
	assignOpenAPIResponse(pathDef, resultSchema)

	loadedModel.App.SwaggerHelper().AddPathSpec(fullPath, verb, pathDef)
	// clean up memory
	pathDef = nil
	runtime.GC()

	remoteMethodOptions := RemoteMethodOptions{
		Name: options.Name,
	}

	handlerWrapper := func(ctx *EventContext) error {

		for _, rl := range options.RateLimits {
			if !rl.Allow(ctx) {
				return ctx.Ctx.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
			}
		}

		req := &RemoteOperationReq[T]{
			Ctx: ctx,
		}
		err := ctx.Ctx.BodyParser(&req.Input)
		if err != nil {
			return ctx.Ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		result, err := handler(req)
		if err != nil {
			return ctx.Ctx.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		ctx.Result = result

		return nil
	}

	loadedModel.remoteMethodsMap[options.Name] = createRemoteMethodOperationItem(handlerWrapper, remoteMethodOptions)

	(*loadedModel.Router).Options(path, func(ctx *fiber.Ctx) error {
		ctx.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		ctx.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		ctx.Set("Access-Control-Allow-Origin", "*")
		return ctx.Status(fiber.StatusNoContent).SendString("")
	})

	return toInvoke(path, createFiberHandler(remoteMethodOptions, loadedModel, verb, path)).Name(loadedModel.Name + "." + remoteMethodOptions.Name)

}

func BindRemoteOperationWithOptions[T any, R any](loadedModel *StatefulModel, handler func(req T) (R, error), options RemoteOperationOptions) fiber.Router {
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
	functionName := getFunctionName(handler)
	slug := wst.DashedCase(functionName)
	path := fmt.Sprintf("/hooks/%s", slug)
	return BindRemoteOperationWithOptions[T, R](loadedModel, handler, RemoteOperationOptions{
		Name:        functionName,
		Path:        path,
		Description: fmt.Sprintf("Invokes %s on %s", functionName, loadedModel.Name),
	})
}

func RemoteOptions() *RemoteOperationOptions {
	return &RemoteOperationOptions{}
}

func (options *RemoteOperationOptions) WithName(name string) *RemoteOperationOptions {
	options.Name = name
	return options
}

func (options *RemoteOperationOptions) WithPath(path string) *RemoteOperationOptions {
	options.Path = path
	return options
}

func (options *RemoteOperationOptions) WithRateLimits(rateLimits ...*RateLimit) *RemoteOperationOptions {
	options.RateLimits = rateLimits
	return options
}
