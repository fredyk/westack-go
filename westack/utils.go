package westack

import (
	"context"
	"fmt"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

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
		conn, err := connectGRPCService(serviceUrl)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Connect Error: %s\n", err)
			return SendError(ctx, err)
		}
		defer func(conn *grpc.ClientConn) {
			err := disconnect(conn)
			if err != nil {
				fmt.Printf("GRPCCallWithQueryParams Disconnect Error: %s\n", err)
			}
		}(conn)
		client := clientConstructor(conn)

		res, err := clientMethod(client, ctx.Context(), &rawParamsQuery)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Call Error: %v --> %s\n", ctx.Route().Name, err)
			return SendError(ctx, err)
		}
		m := jsonpb.MarshalOptions{EmitUnpopulated: true}
		toSend, err := m.Marshal(res)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Marshal Error: %s\n", err)
			return SendError(ctx, err)
		}
		ctx.Response().Header.SetContentType("application/json")
		return ctx.SendString(string(toSend))
	}
}

func gRPCCallWithBody[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc grpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...grpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		//fmt.Printf("%s %T \n", serviceUrl, clientMethod)
		var rawParamsInput InputT
		if err := ctx.BodyParser(&rawParamsInput); err != nil {
			fmt.Printf("GRPCCallWithBody Body Parse Error: %s\n", err)
			return SendError(ctx, err)
		}
		conn, err := connectGRPCService(serviceUrl)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Connect Error: %s\n", err)
			return SendError(ctx, err)
		}
		defer func(conn *grpc.ClientConn) {
			err := disconnect(conn)
			if err != nil {
				fmt.Printf("GRPCCallWithBody Disconnect Error: %s\n", err)
			}
		}(conn)
		client := clientConstructor(conn)

		res, err := clientMethod(client, ctx.Context(), &rawParamsInput)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Call Error: %v --> %s\n", ctx.Route().Name, err)
			return SendError(ctx, err)
		}
		m := jsonpb.MarshalOptions{EmitUnpopulated: true}
		toSend, err := m.Marshal(res)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Marshal Error: %s\n", err)
			return SendError(ctx, err)
		}
		ctx.Response().Header.SetContentType("application/json")
		return ctx.SendString(string(toSend))
	}
}

func connectGRPCService(url string) (*grpc.ClientConn, error) {
	return grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithBlock())
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
