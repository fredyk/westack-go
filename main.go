package main

import (
	"fmt"
	"github.com/fredyk/westack-go/westack"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
)

func main() {
	debug := false
	if envDebug, _ := os.LookupEnv("DEBUG"); envDebug == "true" {
		debug = true
	}
	jwtSecretKey := ""
	if s, present := os.LookupEnv("JWT_SECRET"); present {
		jwtSecretKey = s
		if debug {
			log.Printf("<JWT_SECRET size=%v> found\n", len(jwtSecretKey))
		}
	}
	app := westack.New(westack.Options{
		Debug:        debug,
		RestApiRoot:  "/api/v1",
		Port:         8023,
		JwtSecretKey: []byte(jwtSecretKey),
	})

	app.Boot(func(app *westack.WeStack) {

	})

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
