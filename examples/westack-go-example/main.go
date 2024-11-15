package main

import (
	"github.com/fredyk/westack-go/v2"
	"github.com/gofiber/fiber/v2"
	"log"
)

func main() {

	// Instantiate a new WeStack app
	app := westack.New()

	// Boot the app with a custom boot function
	app.Boot(func(app *westack.WeStack) {

		// Add a custom route
		app.Server.Get("/status", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "ok"})
		})

	})

	// Start the app
	log.Fatal(app.Start())

}
