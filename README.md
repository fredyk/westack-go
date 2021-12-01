# westack-go

### Introduction
This is an experimental migration of Loopback 3 to Go

### Basic example

```go
package main

import (
	"github.com/fredyk/westack-go/westack"
	"github.com/gofiber/fiber/v2"
	"log"
)

func main() {

	app := westack.New(westack.WeStackOptions{
		Debug:       false,
		RestApiRoot: "/api/v1",
		Port:        8023,
	})

	app.Boot(func(app * westack.WeStack) {

		// Setup your custom routes here
		app.Server.Get("/status", func(c * fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "OK"})
		})

	})

	log.Fatal(app.Listen(fmt.Sprintf(":%v", app.Port)))

}

```