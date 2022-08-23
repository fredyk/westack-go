package westack

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	wst "github.com/fredyk/westack-go/westack/common"
)

func (app *WeStack) SwaggerPaths() *map[string]wst.M {
	return &app._swaggerPaths
}
func swaggerDocsHandler(app *WeStack) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {

		hostname := ctx.Hostname()

		matchedProtocol := "https"

		if strings.Contains(hostname, "localhost") || wst.RegexpIpStart.MatchString(hostname) {
			matchedProtocol = "http"
		}

		return ctx.JSON(fiber.Map{
			//"schemes": []string{"http"},
			"openapi": "3.0.1",
			"info": fiber.Map{
				"description":    "This is your go-based API Server.",
				"title":          "Swagger API",
				"termsOfService": "https://swagger.io/terms/",
				"contact": fiber.Map{
					"name":  "API Support",
					"url":   "https://www.swagger.io/support",
					"email": "support@swagger.io",
				},
				"license": fiber.Map{
					"name": "Apache 2.0",
					"url":  "https://www.apache.org/licenses/LICENSE-2.0.html",
				},
				"version": "3.0",
			},
			"components": fiber.Map{
				"securitySchemes": fiber.Map{
					"bearerAuth": fiber.Map{
						"type":         "http",
						"scheme":       "bearer",
						"bearerFormat": "JWT",
					},
				},
			},
			//"security": fiber.Map{
			//	"bearerAuth": fiber.Map{
			//		"type": "http",
			//		"scheme": "bearer",
			//		"bearerFormat": "JWT",
			//	},
			//},
			"servers": []fiber.Map{
				{
					"url": fmt.Sprintf("%v://%v", matchedProtocol, hostname),
				},
				{
					"url": fmt.Sprintf("http://127.0.0.1:%v", app.Port),
				},
			},
			//"basePath": "/",
			"paths": app.SwaggerPaths(),
		})
	}
}
