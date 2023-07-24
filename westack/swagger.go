package westack

import (
	"fmt"
	"github.com/goccy/go-json"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v2"

	wst "github.com/fredyk/westack-go/westack/common"
)

func swaggerDocsHandler(app *WeStack) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {

		// Get X-Forwarded-For header or remote IP
		var remoteForwardedForIp string
		var forwarded bool
		if ctx.Get("X-Forwarded-For") != "" {
			remoteForwardedForIp = ctx.Get("X-Forwarded-For")
			forwarded = true
		} else {
			remoteForwardedForIp = ctx.IP()
		}
		fmt.Printf("Request /swagger/doc.json from %s (forwarded = %t)\n", remoteForwardedForIp, forwarded)

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
		for _, pathItem := range swaggerMap["paths"].(map[string]interface{}) {
			// Iterate over methods
			for _, operation := range pathItem.(map[string]interface{}) {
				operation.(map[string]interface{})["security"] = []fiber.Map{
					{"bearerAuth": []string{}},
				}
				if v, ok := operation.(map[string]interface{})["modelName"]; ok {
					delete(operation.(map[string]interface{}), "modelName")
					operation.(map[string]interface{})["tags"] = []string{v.(string)}
				}
				if v, ok := operation.(map[string]interface{})["rawPathParams"]; ok {
					delete(operation.(map[string]interface{}), "rawPathParams")
					if _, ok := operation.(map[string]interface{})["parameters"]; !ok {
						operation.(map[string]interface{})["parameters"] = make([]interface{}, 0)
					}
					for _, rawPathParam := range v.([]interface{}) {
						operation.(map[string]interface{})["parameters"] = append(operation.(map[string]interface{})["parameters"].([]interface{}), wst.M{
							"name":     strings.TrimPrefix(rawPathParam.(string), ":"),
							"in":       "path",
							"required": true,
							"schema": wst.M{
								"type": "string",
							},
						})
					}
				}
				operation.(map[string]interface{})["responses"] = wst.M{
					"200": wst.M{
						"description": "OK",
						"content": wst.M{
							"application/json": wst.M{
								"schema": wst.M{
									"type": "object",
								},
							},
						},
					},
				}
			}
		}
		bytes, err := marshallSwaggerMap(ctx, err, swaggerMap)
		// Free memory
		swaggerMap = nil
		// Invoke GC
		runtime.GC()
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(wst.M{
				"error": err.Error(),
			})
		}
		return ctx.Status(fiber.StatusOK).Send(bytes)
	}
}

//go:noinline
func marshallSwaggerMap(ctx *fiber.Ctx, err error, swaggerMap map[string]interface{}) ([]byte, error) {
	// marshall
	bytes, err := json.Marshal(swaggerMap)
	return bytes, err
}
