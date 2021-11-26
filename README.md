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

	app := westack.New()

	app.Boot(func(app * westack.WeStack) {

		// Setup your custom routes here
		app.Server.Get("/status", func(c * fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "OK"})
		})

	})

	app.Server.Listen(":8023")

}

```