package main

import (
	"fmt"
	"github.com/fredyk/westack-go/westack"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
)

func main() {

	jwtSecretKey := ""
	if s, present := os.LookupEnv("JWT_SECRET"); present {
		jwtSecretKey = s
	}
	debug := true
	if envDebug, _ := os.LookupEnv("DEBUG"); envDebug == "true" {
		debug = true
	} else if env, present := os.LookupEnv("GO_ENV"); present && env == "PRODUCTION" {
		debug = false
	}
	app := westack.New(westack.Options{
		Debug:        false,
		RestApiRoot:  "/api/v1",
		Port:         8023,
		JwtSecretKey: []byte(jwtSecretKey),
	})

	app.Boot(ServerBoot)

	app.Server.Get("/*", func(c *fiber.Ctx) error {
		log.Println("GET: " + c.Path())
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Unknown method %v %v", c.Method(), c.Path())}})
	})
	app.Server.Post("/*", func(c *fiber.Ctx) error {
		log.Println("POST: " + c.Path())
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"status": 404, "message": fmt.Sprintf("Unknown method %v %v", c.Method(), c.Path())}})
	})

	log.Fatal(app.Start(fmt.Sprintf(":%v", app.Port)))

}
