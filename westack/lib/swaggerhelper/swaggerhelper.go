package swaggerhelper

import (
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"os"
)

type SwaggerHelper interface {
	// GetOpenAPI returns the OpenAPI specification as a map, or an error if it fails
	GetOpenAPI() (map[string]interface{}, error)
	// CreateOpenAPI creates a new OpenAPI specification and saves it to disk, or returns an error if it fails
	CreateOpenAPI() error
	// AddPathSpec adds a path specification to the OpenAPI specification and saves it
	AddPathSpec(path string, verb string, verbSpec map[string]interface{}) error
}

type swaggerHelper struct {
}

func (s *swaggerHelper) GetOpenAPI() (map[string]interface{}, error) {
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

func (s *swaggerHelper) CreateOpenAPI() error {
	openApi := map[string]interface{}{
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
		"servers": make([]fiber.Map, 0),
		//"basePath": "/",
		"paths": make(map[string]interface{}),
	}
	// Marshal
	swagger, err := json.Marshal(openApi)
	if err != nil {
		return err
	}
	// Save
	err2 := os.WriteFile("data/swagger.json", swagger, 0644)
	return err2
}

func (s *swaggerHelper) AddPathSpec(path string, verb string, verbSpec map[string]interface{}) error {
	// Load data/swagger.json
	swagger, err := os.ReadFile("data/swagger.json")
	if err != nil {
		return err
	}
	// Unmarshal
	var swaggerMap map[string]interface{}
	err2 := json.Unmarshal(swagger, &swaggerMap)
	if err2 != nil {
		return err2
	}
	// Add verbSpec to [path][verb]
	if _, ok := swaggerMap["paths"]; !ok {
		swaggerMap["paths"] = make(map[string]interface{})
	}
	paths := swaggerMap["paths"].(map[string]interface{})
	if _, ok := paths[path]; !ok {
		paths[path] = make(map[string]interface{})
	}
	pathMap := paths[path].(map[string]interface{})
	pathMap[verb] = verbSpec
	// Marshal
	swagger, err3 := json.Marshal(swaggerMap)
	if err3 != nil {
		return err3
	}
	// Save
	err4 := os.WriteFile("data/swagger.json", swagger, 0644)
	return err4
}

func NewSwaggerHelper() SwaggerHelper {
	return &swaggerHelper{}
}
