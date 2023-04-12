package westack

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	wst "github.com/fredyk/westack-go/westack/common"
)

func gRPCCallWithQueryParams[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		//fmt.Printf("%s %T \n", serviceUrl, clientMethod)
		var rawParamsQuery InputT
		if err := ctx.QueryParser(&rawParamsQuery); err != nil {
			fmt.Printf("GRPCCallWithQueryParams Query Parse Error: %s\n", err)
			return SendError(ctx, err)
		}
		client, err := obtainConnectedClient(serviceUrl, clientConstructor)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Connect Error: %s\n", err)
			return SendError(ctx, err)
		}

		res, err := clientMethod(client, ctx.Context(), &rawParamsQuery)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Call Error: %v --> %s\n", ctx.Route().Name, err)
			return SendError(ctx, err)
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		toSend, err := m.MarshalToString(res)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Marshal Error: %s\n", err)
			return SendError(ctx, err)
		}
		ctx.Response().Header.SetContentType("application/json")
		return ctx.SendString(toSend)
	}
}

var cachedConnectionsByURL = make(map[string]map[string]interface{})
var cachedConnectionsByURLMutex = &sync.RWMutex{}

func obtainConnectedClient[ClientT interface{}](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT) (ClientT, error) {
	var client ClientT
	cachedConnectionsByURLMutex.Lock()
	defer cachedConnectionsByURLMutex.Unlock()
	if _, ok := cachedConnectionsByURL[serviceUrl]; !ok {
		cachedConnectionsByURL[serviceUrl] = make(map[string]interface{})
	}
	clientConstructorName := fmt.Sprintf("%T", clientConstructor)
	if client1, ok := cachedConnectionsByURL[serviceUrl][clientConstructorName]; ok {
		return client1.(ClientT), nil
	}

	conn, err := connectGRPCService(serviceUrl)
	if err != nil {
		fmt.Printf("GRPCCallWithQueryParams Connect Error: %s\n", err)
		return client, err
	}
	// Disconnect and remove from cache after 5 minutes
	go func(conn *grpc.ClientConn, serviceUrl string, clientConstructorName string) {
		<-time.After(5 * time.Minute)
		cachedConnectionsByURLMutex.Lock()
		delete(cachedConnectionsByURL[serviceUrl], clientConstructorName)
		cachedConnectionsByURLMutex.Unlock()
		// Wait another 5 minutes before disconnecting
		<-time.After(5 * time.Minute)
		err := disconnect(conn)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Disconnect Error: %s\n", err)
		}
	}(conn, serviceUrl, clientConstructorName)
	client = clientConstructor(conn)
	cachedConnectionsByURL[serviceUrl][clientConstructorName] = client
	return client, err
}

func gRPCCallWithBody[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		//fmt.Printf("%s %T \n", serviceUrl, clientMethod)
		var rawParamsInput InputT
		if err := ctx.BodyParser(&rawParamsInput); err != nil {
			fmt.Printf("GRPCCallWithBody Body Parse Error: %s\n", err)
			return SendError(ctx, err)
		}
		client, err := obtainConnectedClient(serviceUrl, clientConstructor)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Connect Error: %s\n", err)
			return SendError(ctx, err)
		}

		res, err := clientMethod(client, ctx.Context(), &rawParamsInput)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Call Error: %v --> %s\n", ctx.Route().Name, err)
			return SendError(ctx, err)
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		toSend, err := m.MarshalToString(res)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Marshal Error: %s\n", err)
			return SendError(ctx, err)
		}
		ctx.Response().Header.SetContentType("application/json")
		return ctx.SendString(toSend)
	}
}

func connectGRPCService(url string) (*grpc.ClientConn, error) {
	fmt.Printf("[DEBUG] wst-grpc: Connecting to %s\n", url)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clientConn, err := grpc.DialContext(ctx, url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithBlock())
	return clientConn, err
}

func disconnect(conn *grpc.ClientConn) error {
	return conn.Close()
}

func SendError(ctx *fiber.Ctx, err error) error {
	newErr := wst.CreateError(fiber.ErrInternalServerError, "ERR_INTERNAL", fiber.Map{"message": err.Error()}, "Error")
	ctx.Response().Header.SetStatusCode(newErr.FiberError.Code)
	ctx.Response().Header.SetStatusMessage([]byte(newErr.FiberError.Message))
	return ctx.JSON(fiber.Map{
		"error": fiber.Map{
			"statusCode": newErr.FiberError.Code,
			"name":       newErr.Name,
			"code":       newErr.Code,
			"error":      newErr.FiberError.Error(),
			"message":    (newErr.Details)["message"],
			"details":    newErr.Details,
		},
	})
}

func replaceVarNames(definition string) string {
	return regexp.MustCompile("\\$(\\w+)").ReplaceAllStringFunc(definition, func(match string) string {
		return "_" + strings.ToUpper(match[1:]) + "_"
	})
}
