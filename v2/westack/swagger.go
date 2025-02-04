package westack

import (
	"fmt"
	"github.com/goccy/go-json"
	"reflect"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v2"

	wst "github.com/fredyk/westack-go/v2/common"
)

func swaggerDocsHandler(app *WeStack) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {

		hostname := ctx.Hostname()

		matchedProtocol := "https"

		if strings.Contains(hostname, "localhost") || wst.RegexpIpStart.MatchString(hostname) {
			matchedProtocol = "http"
		}

		swaggerMap, err := app.swaggerHelper.GetOpenAPI()
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(wst.M{
				"error": err.Error(),
			})
		}
		swaggerMap["servers"] = []wst.M{
			{
				"url": fmt.Sprintf("%v://%v", matchedProtocol, hostname),
			},
			{
				"url": fmt.Sprintf("http://127.0.0.1:%d", app.port),
			},
		}
		// Iterate over paths
		for _, pathItem := range *swaggerMap.GetM("paths") {
			// Iterate over methods
			for _, operation := range pathItem.(wst.M) {
				operation.(wst.M)["security"] = wst.A{
					{"bearerAuth": []string{}},
				}
				if v, ok := operation.(wst.M)["modelName"]; ok {
					delete(operation.(wst.M), "modelName")
					operation.(wst.M)["tags"] = []string{v.(string)}
				}
				if v, ok := operation.(wst.M)["rawPathParams"]; ok {
					delete(operation.(wst.M), "rawPathParams")
					if _, ok := operation.(wst.M)["parameters"]; !ok {
						operation.(wst.M)["parameters"] = make(wst.A, 0)
					}
					for _, rawPathParam := range v.(wst.A) {
						operation.(wst.M)["parameters"] = append(operation.(wst.M)["parameters"].(wst.A), wst.M{
							"name":     strings.TrimPrefix(rawPathParam["<value>"].(string), ":"),
							"in":       "path",
							"required": true,
							"schema": wst.M{
								"type": "string",
							},
						})
					}
				}
				resultSchema := wst.M{
					"type": "object",
				}
				if operationName, ok := operation.(wst.M)["x-operationName"]; ok {
					switch {
					case operationName == string(wst.OperationNameFindById) ||
						operationName == string(wst.OperationNameUpdateAttributes) ||
						operationName == string(wst.OperationNameCreate):
						resultSchema = wst.M{
							"$ref": fmt.Sprintf("#/components/schemas/%v", operation.(wst.M)["x-modelName"]),
						}
					case operationName == string(wst.OperationNameFindMany):
						resultSchema = wst.M{
							"type": "array",
							"items": wst.M{
								"$ref": fmt.Sprintf("#/components/schemas/%v", operation.(wst.M)["x-modelName"]),
							},
						}
					case operationName == string(wst.OperationNameDeleteById):
						resultSchema = wst.M{
							"$ref": fmt.Sprintf("#/components/schemas/%v",
								reflect.TypeFor[wst.DeleteResult]().String()),
						}
					case operationName == string(wst.OperationNameCount):
						resultSchema = wst.M{
							"$ref": fmt.Sprintf("#/components/schemas/%v",
								reflect.TypeFor[wst.CountResult]().String()),
						}
					case operationName == string(wst.OperationNameLogin):
						resultSchema = wst.M{
							"$ref": fmt.Sprintf("#/components/schemas/%v",
								reflect.TypeFor[wst.LoginResult]().String()),
						}
					}
				}
				operation.(wst.M)["responses"] = wst.M{
					"200": wst.M{
						"description": "OK",
						"content": wst.M{
							"application/json": wst.M{
								"schema": resultSchema,
							},
						},
					},
				}
				if overrideResponses, ok := operation.(wst.M)["overrideResponses"]; ok {
					delete(operation.(wst.M), "overrideResponses")
					for k, v := range overrideResponses.(wst.M) {
						operation.(wst.M)["responses"].(wst.M)[k] = v
					}
				}
			}
		}
		// TODO: How to test this error?
		//bytes, err := marshallSwaggerMap(ctx, err, swaggerMap)
		//if err != nil {
		//	return ctx.Status(fiber.StatusInternalServerError).JSON(wst.M{
		//		"error": err.Error(),
		//	})
		//}
		bytes, _ := marshallSwaggerMap(swaggerMap)
		// Assume no error
		// Free memory
		swaggerMap = nil
		// Invoke GC
		runtime.GC()
		return ctx.Status(fiber.StatusOK).Send(bytes)
	}
}

//go:noinline
func marshallSwaggerMap(swaggerMap map[string]interface{}) ([]byte, error) {
	return json.Marshal(swaggerMap)
}
