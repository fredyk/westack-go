package main

import (
	"fmt"
	"log"

	"github.com/fredyk/westack-go/westack"
	"github.com/gofiber/fiber/v2"

	"github.com/fredyk/westack-go/examples/basic-auth-example/server"
)

func main() {

	app := westack.New(westack.Options{
		RestApiRoot: "/api/v1",
		Port:        8023,
	})

	app.Boot(server.ServerBoot)

	app.Server.Get("/*", func(c *fiber.Ctx) error {
		log.Println("GET: " + c.Path())
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Unknown method %v %v", c.Method(), c.Path())}})
	})
	app.Server.Post("/*", func(c *fiber.Ctx) error {
		log.Println("POST: " + c.Path())
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Unknown method %v %v", c.Method(), c.Path())}})
	})

	log.Fatal(app.Start())

}
