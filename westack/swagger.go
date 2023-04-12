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
