package westack

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/memorykv"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/v2/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func appBoot(customRoutesCallbacks []func(app *WeStack), app *WeStack) {

	err := createDataDirectory()
	if err != nil {
		app.logger.Fatalf("Error creating data directory: %v", err)
	}

	err = app.loadDataSources()
	if err != nil {
		app.logger.Fatalf("Error while loading datasources: %v", err)
	}

	err = app.loadModels()
	if err != nil {
		app.logger.Fatalf("Error while loading models: %v", err)
	}

	app.Middleware(func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			app.logger.Printf("Unhandled error: %v\n", err)
		}
		return err
	})

	app.Middleware(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Println(e)
			debug.PrintStack()
		},
	}))

	app.Middleware(createErrorHandler())

	pprofAuthUsername := os.Getenv("PPROF_AUTH_USERNAME")
	pprofAuthPassword := os.Getenv("PPROF_AUTH_PASSWORD")
	if pprofAuthUsername != "" && pprofAuthPassword != "" {
		app.Middleware(utils.PprofHandlers(utils.PprofMiddleOptions{
			Auth: utils.BasicAuthOptions{
				Username: pprofAuthUsername,
				Password: pprofAuthPassword,
			},
		}))
	}

	if app.Options.EnableCompression {
		app.Middleware(compress.New(app.Options.CompressionConfig))
	}

	err = app.loadModelsFixedRoutes()
	if err != nil {
		app.logger.Fatalf("Error while loading models fixed routes: %v", err)
	}

	systemContext := &model.EventContext{
		Bearer: &model.BearerToken{Account: &model.BearerAccount{System: true}},
	}

	// Upsert the admin user
	_, err = UpsertAccountWithRoles(app, AccountWithRoles{
		Username: app.Options.adminUsername,
		Password: app.Options.adminPwd,
		Roles:    []string{"admin"},
	}, systemContext)
	if err != nil {
		app.logger.Fatalf("Error while creating admin user: %v", err)
	}

	for _, cb := range customRoutesCallbacks {
		cb(app)
	}

	app.loadModelsDynamicRoutes()
	app.loadNotFoundRoutes()

	app.Server.Get("/system/memorykv/stats", func(c *fiber.Ctx) error {
		allStats := make(map[string]map[string]memorykv.MemoryKvStats)
		var totalSizeKiB float64
		for _, ds := range *app.datasources {
			if ds.SubViper.GetString("connector") == "memorykv" {
				kvDbStats := ds.Db.(memorykv.MemoryKvDb).Stats()
				allStats[ds.Name] = kvDbStats
				for _, kvStats := range kvDbStats {
					totalSizeKiB += float64(kvStats.TotalSize) / 1024.0
				}
			}
		}
		return c.JSON(fiber.Map{"stats": wst.M{
			"totalSizeKiB": totalSizeKiB,
			"datasources":  allStats,
		}})
	})

	app.Server.Get("/swagger/doc.json", swaggerDocsHandler(app))

	var swaggerUIStatic []byte
	var swaggerContentEncoding = "deflate"
	app.Server.Get("/swagger/*", func(ctx *fiber.Ctx) error {
		if swaggerUIStatic == nil {
			request, err := http.NewRequest("GET", "https://swagger-ui.fhcreations.com/", nil)
			if err != nil {
				return err
			}
			request.Header.Set("Accept", "text/html")
			request.Header.Set("Accept-Encoding", "gzip, deflate, br")
			//request.Header.Set("Accept-Language", "en-US,en;q=0.9")
			//request.Header.Set("Cache-Control", "no-cache")

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				return err
			}

			for _, v := range response.Header["Content-Encoding"] {
				swaggerContentEncoding = v
			}

			reader := response.Body
			swaggerUIStatic, err = io.ReadAll(reader)
			if err != nil {
				return err
			}

			fmt.Printf("[DEBUG] Fetched swagger ui static html (%v bytes)\n", len(swaggerUIStatic))
			fmt.Printf("[DEBUG] Swagger Content-Encoding: %v\n", swaggerContentEncoding)
		}

		ctx.Status(fiber.StatusOK).Set("Content-Type", "text/html; charset=utf-8")
		ctx.Set("Content-Encoding", swaggerContentEncoding)

		return ctx.Send(swaggerUIStatic)
	})

	// Free up memory
	err = app.swaggerHelper.Dump()
	if err != nil {
		fmt.Printf("Error while dumping swagger helper: %v\n", err)
	}

	app.completedSetup = true
}

func createDataDirectory() error {
	// Create data directory if it doesn't exist
	_, err := os.Stat("data")
	if os.IsNotExist(err) {
		err = os.Mkdir("data", 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func createErrorHandler() func(ctx *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		err := c.Next()
		if err != nil {
			log.Println("Error:", err)
			log.Printf("%v: %v\n", method, c.OriginalURL())
			if errors.Is(err, fiber.ErrUnauthorized) {
				err = wst.CreateError(fiber.ErrUnauthorized, "UNAUTHORIZED", fiber.Map{"message": "Unauthorized"}, "Error")
			}
			switch err := err.(type) {
			case *fiber.Error:
				if err.Code == fiber.StatusNotFound {
					return c.Status(err.Code).JSON(fiber.Map{"error": fiber.Map{"status": err.Code, "message": fmt.Sprintf("Unknown method %v %v", method, c.Path())}})
				} else {
					return c.Status(err.Code).JSON(fiber.Map{"error": fiber.Map{"status": err.Code, "message": err.Message}})
				}
			case *wst.WeStackError:
				errorName := err.Name
				if errorName == "" {
					errorName = "Error"
				}
				return c.Status(err.FiberError.Code).JSON(fiber.Map{
					"error": fiber.Map{
						"statusCode": err.FiberError.Code,
						"name":       errorName,
						"code":       err.Code,
						"error":      err.FiberError.Error(),
						"message":    (err.Details)["message"],
						"details":    err.Details,
					},
				})
			default:
				//return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"status": fiber.StatusInternalServerError, "message": err.Error()}})
				return SendInternalError(c, err)
			}
		}
		return nil
	}
}
