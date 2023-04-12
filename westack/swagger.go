package westack

import (
	"fmt"
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
		return ctx.JSON(swaggerMap)
	}
}
