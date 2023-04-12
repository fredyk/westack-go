package swaggerhelper

import (
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"os"
	"runtime"
)

type SwaggerHelper interface {
	// GetOpenAPI returns the OpenAPI specification as a map, or an error if it fails
	GetOpenAPI() (map[string]interface{}, error)
	// CreateOpenAPI creates a new OpenAPI specification and saves it to disk, or returns an error if it fails
	CreateOpenAPI() error
	// AddPathSpec adds a path specification to the OpenAPI specification
	AddPathSpec(path string, verb string, verbSpec map[string]interface{})
	// Dump dumps the OpenAPI specification to disk, or returns an error if it fails
	Dump() error
}

type swaggerHelper struct {
	swaggerMap map[string]interface{}
}

func (sH *swaggerHelper) GetOpenAPI() (map[string]interface{}, error) {
	// Load data/swagger.json
	swagger, err := os.ReadFile("data/swagger.json")
	if err != nil {
		return nil, err
	}
	// Unmarshal it into a map
	var swaggerMap map[string]interface{}
	err = json.Unmarshal(swagger, &swaggerMap)
	if err != nil {
		return nil, err
	}
	return swaggerMap, nil
}

func (sH *swaggerHelper) CreateOpenAPI() error {
	sH.swaggerMap = map[string]interface{}{
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
		"servers": make([]map[string]interface{}, 0),
		//"basePath": "/",
		"paths": make(map[string]interface{}),
	}
	// Marshal
	swagger, err := json.Marshal(sH.swaggerMap)
	if err != nil {
		return err
	}
	// Create data directory if it doesn't exist
	_, err = os.Stat("data")
	if os.IsNotExist(err) {
		err = os.Mkdir("data", 0755)
		if err != nil {
			return err
		}
	}
	// Save
	err2 := os.WriteFile("data/swagger.json", swagger, 0644)
	return err2
}

func (sH *swaggerHelper) AddPathSpec(path string, verb string, verbSpec map[string]interface{}) {
	// Add verbSpec to [path][verb]
	if _, ok := sH.swaggerMap["paths"].(map[string]interface{})[path]; !ok {
		sH.swaggerMap["paths"].(map[string]interface{})[path] = make(map[string]interface{})
	}
	sH.swaggerMap["paths"].(map[string]interface{})[path].(map[string]interface{})[verb] = verbSpec
	return
}

func (sH *swaggerHelper) Dump() error {
	// Marshal
	swagger, err := json.Marshal(sH.swaggerMap)
	if err != nil {
		return err
	}
	// Save
	err2 := os.WriteFile("data/swagger.json", swagger, 0644)
	// Free up memory
	swagger = nil
	sH.free()
	return err2
}

func (sH *swaggerHelper) free() {
	sH.swaggerMap = nil
	// Invoke the GC to free up the memory immediately
	runtime.GC()
}

func NewSwaggerHelper() SwaggerHelper {
	return &swaggerHelper{}
}
