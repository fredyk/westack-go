package swaggerhelper

import (
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/lib/swaggerhelperinterface"
	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"os"
	"runtime"
)

type SwaggerMap interface {
	easyjson.Marshaler
	//map[string]interface{}
}

type swaggerHelper struct {
	swaggerMap SwaggerMap
}

func (sH *swaggerHelper) GetOpenAPI() (map[string]interface{}, error) {
	// Load data/swagger.json
	swagger, err := os.ReadFile("data/swagger.json")
	if err != nil {
		return nil, err
	}
	// Unmarshal it into a map
	var swaggerMap wst.M
	err = easyjson.Unmarshal(swagger, &swaggerMap)
	if err != nil {
		return nil, err
	}
	return swaggerMap, nil
}

func (sH *swaggerHelper) CreateOpenAPI() error {
	sH.swaggerMap = wst.M{
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
	//swagger, err := easyjson.Marshal(sH.swaggerMap)
	jw := jwriter.Writer{}
	sH.swaggerMap.MarshalEasyJSON(&jw)
	err := jw.Error
	if err != nil {
		return err
	}
	swagger, err := jw.BuildBytes()
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
	if _, ok := sH.swaggerMap.(wst.M)["paths"].(map[string]interface{})[path]; !ok {
		sH.swaggerMap.(wst.M)["paths"].(map[string]interface{})[path] = make(map[string]interface{})
	}
	sH.swaggerMap.(wst.M)["paths"].(map[string]interface{})[path].(map[string]interface{})[verb] = verbSpec
	return
}

func (sH *swaggerHelper) Dump() error {
	// Marshal
	jw := jwriter.Writer{}
	sH.swaggerMap.MarshalEasyJSON(&jw)
	err := jw.Error
	if err != nil {
		return err
	}
	swagger, err := jw.BuildBytes()
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

func NewSwaggerHelper() swaggerhelperinterface.SwaggerHelper {
	return &swaggerHelper{}
}
