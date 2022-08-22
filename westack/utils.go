package westack

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.elastic.co/apm/module/apmgrpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	wst "github.com/fredyk/westack-go/westack/common"
)

func GRPCCallWithQueryParams[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc gogrpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...gogrpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
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
		defer func(conn *gogrpc.ClientConn) {
			err := disconnect(conn)
			if err != nil {
				fmt.Printf("GRPCCallWithQueryParams Disconnect Error: %s\n", err)
			}
		}(conn)
		client := clientConstructor(conn)

		res, err := clientMethod(client, ctx.Context(), &rawParamsQuery)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Call Error: %s\n", err)
			return SendError(ctx, err)
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		tosend, err := m.MarshalToString(res)
		if err != nil {
			fmt.Printf("GRPCCallWithQueryParams Marshal Error: %s\n", err)
			return SendError(ctx, err)
		}
		ctx.Response().Header.SetContentType("application/json")
		return ctx.SendString(tosend)
	}
}

func GRPCCallWithBody[InputT any, ClientT interface{}, OutputT proto.Message](serviceUrl string, clientConstructor func(cc gogrpc.ClientConnInterface) ClientT, clientMethod func(ClientT, context.Context, *InputT, ...gogrpc.CallOption) (OutputT, error)) func(ctx *fiber.Ctx) error {
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
		defer func(conn *gogrpc.ClientConn) {
			err := disconnect(conn)
			if err != nil {
				fmt.Printf("GRPCCallWithBody Disconnect Error: %s\n", err)
			}
		}(conn)
		client := clientConstructor(conn)

		res, err := clientMethod(client, ctx.Context(), &rawParamsInput)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Call Error: %s\n", err)
			return SendError(ctx, err)
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		tosend, err := m.MarshalToString(res)
		if err != nil {
			fmt.Printf("GRPCCallWithBody Marshal Error: %s\n", err)
			return SendError(ctx, err)
		}
		ctx.Response().Header.SetContentType("application/json")
		return ctx.SendString(tosend)
	}
}

func connectGRPCService(url string) (*gogrpc.ClientConn, error) {
	return gogrpc.Dial(url, gogrpc.WithTransportCredentials(insecure.NewCredentials()), gogrpc.WithBlock(), gogrpc.WithBlock(), gogrpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
		gogrpc.WithStreamInterceptor(apmgrpc.NewStreamClientInterceptor()))
}

func disconnect(conn *gogrpc.ClientConn) error {
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
