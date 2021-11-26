package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/gofiber/fiber/v2"
)

type Property struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Default  string `json:"default"`
}

type Relation struct {
	Type  string `json:"type"`
	Model string `json:"model"`
}

type ACL struct {
	AccessType    string `json:"accessType"`
	PrincipalType string `json:"principalType"`
}

type Config struct {
	Name       string              `json:"name"`
	Plural     string              `json:"plural"`
	Base       string              `json:"base"`
	Datasource string              `json:"dataSource"`
	Public     bool                `json:"public"`
	Properties map[string]Property `json:"properties"`
	Relations  map[string]Relation `json:"relations"`
	Acls       []ACL               `json:"acls"`
	Hidden     []string            `json:"hidden"`
}

type DataSourceConfig struct {
	Name      string `json:"name"`
	Connector string `json:"connector"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	User      string `json:"user"`
	Password  string `json:"password"`
}

type Model struct {
	Name          string `json:"name"`
	Config        Config
	Datasource    *datasource.Datasource
	Router        *fiber.Router
	App           *common.IApp
	eventHandlers map[string]func(eventContext *EventContext) error
	BaseUrl       string
}

func (loadedModel *Model) SendError(ctx *fiber.Ctx, err error) error {
	switch err.(type) {
	case *WeStackError:
		return ctx.Status((err).(*WeStackError).FiberError.Code).JSON(fiber.Map{"error": err.(*WeStackError).FiberError.Error(), "details": err.(*WeStackError).Details})
	}
	return err
}

func New(config Config) *Model {
	return &Model{
		Name:          config.Name,
		Config:        config,
		eventHandlers: map[string]func(eventContext *EventContext) error{},
	}
}

type ModelInstance struct {
	data  map[string]interface{}
	Model Model
	Id    interface{}
}

type RegistryEntry struct {
	Name  string
	Model *Model
}

func (modelInstance ModelInstance) ToJSON() map[string]interface{} {
	var result map[string]interface{}
	result = modelInstance.data
	return result
}

func (loadedModel Model) Build(data map[string]interface{}, fromDb bool) ModelInstance {

	if data["id"] == nil {
		data["id"] = data["_id"]
		if data["id"] != nil {
			delete(data, "_id")
		}
	}
	data = common.CopyMap(data)
	modelInstance := ModelInstance{
		Id:    data["id"],
		data:  data,
		Model: loadedModel,
	}
	eventContext := EventContext{
		Data:     &data,
		Instance: &modelInstance,
		Ctx:      nil,
	}
	loadedModel.GetHandler("__operation__loaded")(&eventContext)

	return modelInstance
}

func parseFilter(filter string) *map[string]interface{} {
	var filterMap *map[string]interface{}
	if filter != "" {
		_ = json.Unmarshal([]byte(filter), &filterMap)
	}
	return filterMap
}

func (loadedModel Model) FindMany(filterMap *map[string]interface{}) ([]ModelInstance, error) {

	var documents []map[string]interface{}
	err := loadedModel.Datasource.FindMany(loadedModel.Name, filterMap).All(context.Background(), &documents)
	if err != nil {
		return nil, err
	}

	var results = make([]ModelInstance, len(documents))

	for idx, document := range documents {
		results[idx] = loadedModel.Build(document, true)
	}

	return results, nil
}

type OperationError struct {
	Code    int
	Message string
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%v %v", e.Code, e.Message)
}

func NewError(code int, message string) *OperationError {
	res := &OperationError{
		code, message,
	}
	return res
}

func (loadedModel Model) FindById(id string, filterMap map[string]interface{}) (*ModelInstance, error) {
	var document map[string]interface{}
	cursor := loadedModel.Datasource.FindById(loadedModel.Name, id, &filterMap)
	if cursor != nil {
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		} else {
			result := loadedModel.Build(document, true)
			return &result, nil
		}
	} else {
		return nil, NewError(404, "Document "+id+" found")
	}
}

func (loadedModel Model) Create(data map[string]interface{}) (*ModelInstance, error) {

	eventContext := EventContext{
		Data:          &data,
		Ctx:           nil,
		IsNewInstance: true,
	}
	loadedModel.GetHandler("__operation__before_save")(&eventContext)
	var document map[string]interface{}
	cursor := loadedModel.Datasource.Create(loadedModel.Name, &data)
	if cursor != nil {
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		} else {
			result := loadedModel.Build(document, true)
			result.hideProperties()
			loadedModel.GetHandler("__operation__after_save")(&EventContext{
				Data:          &result.data,
				Instance:      &result,
				Ctx:           nil,
				IsNewInstance: true,
			})
			return &result, nil
		}
	} else {
		return nil, NewError(400, "Could not create document")
	}

}

func handleError(c *fiber.Ctx, err error) error {
	switch err.(type) {
	case *OperationError:
		return c.Status(err.(*OperationError).Code).JSON(fiber.Map{"error": fiber.Map{"status": err.(*OperationError).Code, "message": err.(*OperationError).Message}})
	default:
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"status": 500, "message": "Internal Server Error"}})
	}
}

func (modelInstance ModelInstance) hideProperties() {
	for _, propertyName := range modelInstance.Model.Config.Hidden {
		delete(modelInstance.data, propertyName)
	}
}

func (loadedModel Model) FindManyRoute(c *fiber.Ctx) error {
	filterSt := c.Query("filter")
	filterMap := parseFilter(filterSt)

	result, err := loadedModel.FindMany(filterMap)
	out := make([]map[string]interface{}, len(result))
	for idx, item := range result {
		item.hideProperties()
		out[idx] = item.ToJSON()
	}

	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(out)
}

func (loadedModel Model) FindByIdRoute(c *fiber.Ctx) error {
	id := c.Params("id")
	if regexp.MustCompile("^([0-9a-f]{24})$").MatchString(id) {
		filterSt := c.Query("filter")
		filterMap := parseFilter(filterSt)
		if filterMap == nil {
			filterMap = &map[string]interface{}{}
		}

		if filterSt == "" {

		}
		result, err := loadedModel.FindById(id, *filterMap)
		result.hideProperties()
		if err != nil {
			return handleError(c, err)
		}
		return c.JSON(result.ToJSON())
	} else {
		return c.Next()
	}
}

type RemoteMethodOptionsHttp struct {
	Path string
	Verb string
}

type RemoteMethodOptions struct {
	Description string
	Http        RemoteMethodOptionsHttp
}

func (loadedModel Model) RemoteMethod(handler func(c *fiber.Ctx) error, options RemoteMethodOptions) fiber.Router {
	var http = options.Http
	path := strings.ToLower(http.Path)
	verb := strings.ToLower(http.Verb)
	description := options.Description

	var toInvoke func(string, ...fiber.Handler) fiber.Router
	operation := ""

	switch verb {
	case "get":
		toInvoke = (*loadedModel.Router).Get
		operation = "Finds"
	case "options":
		toInvoke = (*loadedModel.Router).Options
		operation = "Gets options for"
	case "head":
		toInvoke = (*loadedModel.Router).Head
		operation = "Checks"
	case "post":
		toInvoke = (*loadedModel.Router).Post
		operation = "Creates"
	case "put":
		toInvoke = (*loadedModel.Router).Put
		operation = "Replaces"
	case "patch":
		toInvoke = (*loadedModel.Router).Patch
		operation = "Updates attributes in"
	case "delete":
		toInvoke = (*loadedModel.Router).Delete
		operation = "Deletes"
	}

	fullPath := loadedModel.BaseUrl + "/" + path
	fullPath = regexp.MustCompile("//+").ReplaceAllString(fullPath, "/")

	if (*loadedModel.App.SwaggerPaths())[fullPath] == nil {
		(*loadedModel.App.SwaggerPaths())[fullPath] = map[string]interface{}{}
	}

	if description == "" {
		description = fmt.Sprintf("%v %v.", operation, loadedModel.Config.Plural)
	}

	var parameters []map[string]interface{}
	if verb == "post" || verb == "put" || verb == "patch" {
		parameters = []map[string]interface{}{
			{
				"name":        "data",
				"in":          "body",
				"description": "data",
				"required":    "true",
				"schema": map[string]interface{}{
					"type": "object",
				},
			},
		}
	} else {
		parameters = []map[string]interface{}{}
	}
	(*loadedModel.App.SwaggerPaths())[fullPath][verb] = map[string]interface{}{
		"description": description,
		"consumes": []string{
			"*/*",
		},
		"produces": []string{
			"application/json",
		},
		"tags": []string{
			loadedModel.Config.Name,
		},
		"parameters": parameters,
		"summary":    description,
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "OK",
				"schema": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": true,
				},
			},
		},
	}

	return toInvoke(path, handler)
}

type EventContext struct {
	Data          *map[string]interface{}
	Instance      *ModelInstance
	Ctx           *fiber.Ctx
	IsNewInstance bool
	Result        interface{}
}

type WeStackError struct {
	FiberError *fiber.Error
	Details    fiber.Map
	Ctx        *EventContext
}

func (err *WeStackError) Error() string {
	return fmt.Sprintf("%v %v: %v", err.FiberError.Code, err.FiberError.Error(), err.Details)
}

func (ctx *EventContext) RestError(fiberError *fiber.Error, details fiber.Map) error {
	ctx.Result = details
	return &WeStackError{
		FiberError: fiberError,
		Details:    details,
		Ctx:        ctx,
	}
}

func wrapEventHandler(model Model, eventKey string, handler func(eventContext *EventContext) error) func(eventContext *EventContext) error {
	currentHandler := model.eventHandlers[eventKey]
	if currentHandler != nil {
		newHandler := handler
		handler = func(eventContext *EventContext) error {
			currentHandlerResult := currentHandler(eventContext)
			if currentHandlerResult != nil {
				return currentHandlerResult
			} else {
				return newHandler(eventContext)
			}
		}
	}
	return handler
}

func (loadedModel Model) On(event string, handler func(eventContext *EventContext) error) {
	loadedModel.eventHandlers[event] = wrapEventHandler(loadedModel, event, handler)
}

func (loadedModel Model) Observe(operation string, handler func(eventContext *EventContext) error) {
	eventKey := "__operation__" + strings.ReplaceAll(strings.TrimSpace(operation), " ", "_")
	loadedModel.On(eventKey, handler)
}

func (loadedModel Model) GetHandler(event string) func(eventContext *EventContext) error {
	res := loadedModel.eventHandlers[event]
	if res == nil {
		res = func(eventContext *EventContext) error {
			log.Println("no handler found for ", loadedModel.Name, ".", event)
			return nil
		}
	}
	return res
}
