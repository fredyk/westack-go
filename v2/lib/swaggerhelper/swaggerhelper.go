package swaggerhelper

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
)

type SwaggerMap interface {
	easyjson.Marshaler
	//map[string]interface{}
}

type swaggerHelper struct {
	swaggerMap SwaggerMap
	app        *wst.IApp
}

func (sH *swaggerHelper) GetOpenAPI() (wst.M, error) {
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
	sH.swaggerMap = &wst.M{
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
		"components": &wst.M{
			"securitySchemes": fiber.Map{
				"bearerAuth": fiber.Map{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": wst.M{
				//"ExampleSchema": wst.M{
				//	"type": "object",
				//	"properties": wst.M{
				//		"foo": wst.M{
				//			"type": "string",
				//		},
				//		"bar": wst.M{
				//			"type": "integer",
				//		},
				//	},
				//},
			},
		},
		//"security": fiber.Map{
		//	"bearerAuth": fiber.Map{
		//		"type": "http",
		//		"scheme": "bearer",
		//		"bearerFormat": "JWT",
		//	},
		//},
		"servers": make([]wst.M, 0),
		//"basePath": "/",
		"paths": make(wst.M),
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
	// Save
	err2 := os.WriteFile("data/swagger.json", swagger, 0600)
	return err2
}

func (sH *swaggerHelper) RemovePathSpec(path string, verb string) {
	if sH.app.CompletedSetup() {
		os.Stderr.WriteString("Cannot remove path spec after setup is completed\n")
		os.Stderr.WriteString("Maybe you are trying to register a remote operation after the app.Boot() was called?\n")
		log.Fatal("Exiting")
	}
	// Remove [path][verb]
	if _, ok := (*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path]; ok {
		if _, ok := (*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path].(wst.M)[verb]; ok {
			delete((*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path].(wst.M), verb)

			if len((*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path].(wst.M)) == 0 {
				delete((*sH.swaggerMap.(*wst.M))["paths"].(wst.M), path)
			}
		} else {
			os.Stderr.WriteString("Spec not found for: " + path + " " + verb + "\n")
		}
	} else {
		os.Stderr.WriteString("Path not found: " + path + "\n")
	}
}

func (sH *swaggerHelper) AddPathSpec(path string, verb string, verbSpec wst.M, operationName string, modelName string) {
	if sH.app.CompletedSetup() {
		os.Stderr.WriteString("Cannot add path spec after setup is completed\n")
		os.Stderr.WriteString("Maybe you are trying to register a remote operation after the app.Boot() was called?\n")
		log.Fatal("Exiting")
	}
	// Add verbSpec to [path][verb]
	if _, ok := (*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path]; !ok {
		(*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path] = make(wst.M)
	}
	verbSpec["x-operationName"] = operationName
	verbSpec["x-modelName"] = modelName
	(*sH.swaggerMap.(*wst.M))["paths"].(wst.M)[path].(wst.M)[verb] = verbSpec
	return
}

func (sH *swaggerHelper) GetComponents() wst.M {
	return *(*sH.swaggerMap.(*wst.M))["components"].(*wst.M)
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
	err2 := os.WriteFile("data/swagger.json", swagger, 0600)
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

func NewSwaggerHelper(app *wst.IApp) wst.SwaggerHelper {
	return &swaggerHelper{
		app: app,
	}
}

type OpenApiModelDef struct {
	Name       string
	Properties wst.M
}

func RegisterGenericComponentForSample(sH wst.SwaggerHelper, sample any) string {
	// important to set it to nil first to avoid infinite recursion
	components := (*sH.(*swaggerHelper).swaggerMap.(*wst.M))["components"].(*wst.M)
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	itemSchemaName := t.String()
	schemaName := itemSchemaName

	if t.Name() == "" {

		// check if it is a slice
		if t.Kind() == reflect.Slice {
			t = t.Elem()
			itemSchemaName = t.String()

			if t.Kind() == reflect.Ptr {
				t = t.Elem()
				itemSchemaName = t.String()
			}
			if t.Name() == "" {
				// anonymous struct
				itemSchemaName = "object"
			}
			if t.Kind() == reflect.Struct {
				itemSchemaName = RegisterGenericComponentForSample(sH, reflect.New(t).Interface())
			}

			schemaName = fmt.Sprintf("%sList", itemSchemaName)
			if _, ok := (*components)["schemas"].(wst.M)[schemaName]; ok {
				return schemaName
			}
			(*components)["schemas"].(wst.M)[schemaName] = nil
			(*components)["schemas"].(wst.M)[schemaName] = wst.M{
				"type": "array",
				"items": wst.M{
					"$ref": "#/components/schemas/" + itemSchemaName,
				},
			}
			return schemaName
		} else {
			fmt.Printf("Anonymous struct %v %T\n", t, sample)
			// anonymous struct
			return "object"
		}
	}

	if _, ok := (*components)["schemas"].(wst.M)[schemaName]; ok {
		return schemaName
	}
	(*components)["schemas"].(wst.M)[schemaName] = nil
	(*components)["schemas"].(wst.M)[schemaName] = wst.M{
		"type":       "object",
		"properties": analyzeWithReflection(schemaName, t, components),
	}
	return schemaName
}

func RegisterGenericComponent[T any](sH wst.SwaggerHelper) string {
	sample := new(T)
	return RegisterGenericComponentForSample(sH, sample)
}

func getStructTag(f reflect.StructField, tagName string) string {
	return strings.Split(f.Tag.Get(tagName), ",")[0]
}

func analyzeWithReflection(rootTypeName string, t reflect.Type, components *wst.M) wst.M {
	schema := wst.M{}
	//valueOf := reflect.ValueOf(sample)
	var fields int
	if t.Kind() == reflect.Map {
		return wst.M{
			"type":       "object",
			"properties": wst.M{},
		}
	} else if t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		fields = t.Elem().NumField()
	} else {
		fields = t.NumField()
	}
	if rootTypeName == "" {
		rootTypeName = t.Name()
	}
	for i := 0; i < fields; i++ {
		var field reflect.StructField
		if t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
			field = t.Elem().Field(i)
		} else {
			field = t.Field(i)
		}
		// only if exported
		if field.PkgPath != "" {
			continue
		}
		tagged := getStructTag(field, "json")
		if tagged == "" {
			tagged = field.Name
		} else if tagged == "-" {
			continue
		}
		if field.Type.Kind() == reflect.Pointer {
			field.Type = field.Type.Elem()
		}
		switch field.Type.Name() {
		case "ObjectID":
			schema[tagged] = wst.M{
				"type": "string",
			}
			continue
		case "Time":
			schema[tagged] = wst.M{
				"type": "string",
			}
			continue
		}
		switch field.Type.Kind() {
		case reflect.String:
			schema[tagged] = wst.M{
				"type": "string",
			}
		//case int, int32, int64, float32, float64:
		case reflect.Int, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64, reflect.Uintptr:
			schema[tagged] = wst.M{
				"type": "number",
			}
		//case bool:
		case reflect.Bool:
			schema[tagged] = wst.M{
				"type": "boolean",
			}
		//case wst.M:
		case reflect.Map:
			schema[tagged] = analyzeWithReflection(rootTypeName, field.Type, components)
		//case []interface{}:
		case reflect.Slice:
			fieldElem := field.Type.Elem()
			itemType := fieldElem.Kind()
			fieldObjectTypeName := fieldElem.String()
			if fieldElem.Name() == "" {
				// anonymous struct
				fieldObjectTypeName = rootTypeName + field.Name
			}
			if itemType == reflect.Struct || itemType == reflect.Slice || itemType == reflect.Map || itemType == reflect.Interface {
				if _, ok := (*components)["schemas"].(wst.M)[fieldObjectTypeName]; !ok {
					(*components)["schemas"].(wst.M)[fieldObjectTypeName] = nil
					(*components)["schemas"].(wst.M)[fieldObjectTypeName] = wst.M{
						"type":       "object",
						"properties": analyzeWithReflection(fieldObjectTypeName, fieldElem, components),
					}
				}
				if len((*components)["schemas"].(wst.M)[fieldObjectTypeName].(wst.M)["properties"].(wst.M)) == 0 {
					// empty struct
					schema[tagged] = wst.M{
						"type": "array",
						"items": wst.M{
							"type": "object",
						},
					}
					// delete the empty struct
					delete((*components)["schemas"].(wst.M), fieldObjectTypeName)
				} else {
					schema[tagged] = wst.M{
						"type": "array",
						"items": wst.M{
							"$ref": "#/components/schemas/" + fieldObjectTypeName,
						},
					}
				}
			} else {
				var primitiveType string
				if itemType == reflect.String {
					primitiveType = "string"
				} else if itemType == reflect.Int || itemType == reflect.Int32 || itemType == reflect.Int64 || itemType == reflect.Float32 || itemType == reflect.Float64 {
					primitiveType = "number"
				} else if itemType == reflect.Bool {
					primitiveType = "boolean"
				} else {
					panic("Unknown primitive type " + itemType.String())
				}
				schema[tagged] = wst.M{
					"type": "array",
					"items": wst.M{
						"type": primitiveType,
					},
				}
			}
		case reflect.Struct:
			fieldObjectTypeName := field.Type.String()
			if field.Type.Name() == "" {
				// anonymous struct
				fieldObjectTypeName = rootTypeName + field.Name
			}
			if fieldObjectTypeName == "time.Time" {
				schema[tagged] = wst.M{
					"type": "string",
				}
			} else {

				// multipart.FileHeader maps to "type": "file"
				if fieldObjectTypeName == "multipart.FileHeader" {
					schema[tagged] = wst.M{
						"type": "file",
					}
					continue
				}

				if _, ok := (*components)["schemas"].(wst.M)[fieldObjectTypeName]; !ok {
					(*components)["schemas"].(wst.M)[fieldObjectTypeName] = nil
					(*components)["schemas"].(wst.M)[fieldObjectTypeName] = wst.M{
						"type":       "object",
						"properties": analyzeWithReflection(fieldObjectTypeName, field.Type, components),
					}
				}
				if len((*components)["schemas"].(wst.M)[fieldObjectTypeName].(wst.M)["properties"].(wst.M)) == 0 {
					// empty struct
					schema[tagged] = wst.M{
						"type": "object",
					}
					// delete the empty struct
					delete((*components)["schemas"].(wst.M), fieldObjectTypeName)
				} else {
					schema[tagged] = wst.M{
						"$ref": "#/components/schemas/" + fieldObjectTypeName,
					}
				}
			}
		case reflect.Interface:
			log.Fatalf("Interfaces are not meant to be used in JSON serialization at %s->%s", rootTypeName, field.Name)
		default:
			panic("Unknown type " + field.Type.Kind().String())
		}
	}
	return schema
}
