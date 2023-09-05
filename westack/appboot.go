package westack

import (
	"fmt"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/memorykv"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/fredyk/westack-go/westack/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
)

func appBoot(customRoutesCallbacks []func(app *WeStack), app *WeStack) {
	app.loadDataSources()

	err := app.loadModels()
	if err != nil {
		app.logger.Fatalf("Error while loading models: %v", err)
	}

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

	app.Middleware(func(c *fiber.Ctx) error {
		method := c.Method()
		err := c.Next()
		if err != nil {
			log.Println("Error:", err)
			log.Printf("%v: %v\n", method, c.OriginalURL())
			switch err.(type) {
			case *fiber.Error:
				if err.(*fiber.Error).Code == fiber.StatusNotFound {
					return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": fiber.Map{"status": err.(*fiber.Error).Code, "message": fmt.Sprintf("Unknown method %v %v", method, c.Path())}})
				} else {
					return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": fiber.Map{"status": err.(*fiber.Error).Code, "message": err.(*fiber.Error).Message}})
				}
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"status": fiber.StatusInternalServerError, "message": err.Error()}})
			}
		}
		return nil
	})

	app.Middleware(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Println(e)
			debug.PrintStack()
		},
	}))

	if app.Options.EnableCompression {
		app.Middleware(compress.New(app.Options.CompressionConfig))
	}

	err = app.loadModelsFixedRoutes()
	if err != nil {
		app.logger.Fatalf("Error while loading models fixed routes: %v", err)
	}

	systemContext := &model.EventContext{
		Bearer: &model.BearerToken{User: &model.BearerUser{System: true}},
	}

	// Upsert the admin user
	_, err = UpsertUserWithRoles(app, UserWithRoles{
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

			for _, v := range response.Header["Content-Encoding"] {
				swaggerContentEncoding = v
			}

			var reader io.Reader
			//// decompress
			//switch swaggerContentEncoding {
			//case "gzip":
			//	reader, err = gzip.NewReader(response.Body)
			//	if err != nil {
			//		break
			//	}
			//case "br":
			//	reader = brotli.NewReader(response.Body)
			//case "deflate", "":
			reader = response.Body
			//}
			swaggerUIStatic, err = io.ReadAll(reader)
			if err != nil {
				return err
			}

			fmt.Printf("DEBUG: Fetched swagger ui static html (%v bytes)\n", len(swaggerUIStatic))
			fmt.Printf("DEBUG: Swagger Content-Encoding: %v\n", swaggerContentEncoding)
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
}
