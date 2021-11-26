package main

import (
	"fmt"
	"github.com/fredyk/westack-go/westack"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/gofiber/fiber/v2"
	"log"
)

func main() {

	app := westack.New()

	app.Boot(setupRoutes)

	// For the ones pending to be done
	app.Server.Get("/*", func(c *fiber.Ctx) error {
		log.Println("GET: " + c.Path())
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Unknown method %v %v", c.Method(), c.Path())}})
	})
	app.Server.Post("/*", func(c *fiber.Ctx) error {
		log.Println("POST: " + c.Path())
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Unknown method %v %v", c.Method(), c.Path())}})
	})

	log.Fatal(app.Server.Listen(":8023"))

}

func setupRoutes(app *westack.WeStack) {

	app.FindModel("role").Observe("loaded", func(eventContext *model.EventContext) error {
		log.Println("loaded role ", eventContext.Data)
		return nil
	})

	userModel := app.FindModel("user")

	userModel.Observe("before save", func(eventContext *model.EventContext) error {
		log.Println("Before saving ", eventContext.Data, eventContext.IsNewInstance)
		return nil
	})

	userModel.Observe("after save", func(eventContext *model.EventContext) error {
		log.Println("After saving ", eventContext.Instance, eventContext.IsNewInstance)
		return nil
	})

	userModel.On("login", func(ctx *model.EventContext) error {
		log.Println("login instance ", ctx.Instance)
		log.Println("login data ", ctx.Data)
		ctx.Result = fiber.Map{"status": "Override login", "initial": ctx.Result}
		return nil
	})

	userModel.RemoteMethod(func(c *fiber.Ctx) error {
		return userModel.SendError(c, (&model.EventContext{}).RestError(fiber.ErrTeapot, fiber.Map{"error": "I used to be a cup"}))
	}, model.RemoteMethodOptions{
		Description: "Example error",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/example-error",
			Verb: "get",
		},
	})

}
