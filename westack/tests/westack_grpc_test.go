package tests

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/fredyk/westack-go/westack"

	pb "github.com/fredyk/westack-go/westack/tests/proto"
)

func Test_GRPCCallWithQueryParams(t *testing.T) {
	// setup
	// start server
	server := westack.New(westack.Options{
		Port: 8020,
	})

	// start a mock grpc server
	go startMockGrpcServer()

	server.Boot(func(app *westack.WeStack) {
		app.Server.Get("/test-grpc-get", westack.GRPCCallWithQueryParams[pb.ReqGrpcTestMessage, pb.FooClient, *pb.ResGrpcTestMessage](
			"localhost:7777",
			pb.NewGrpcTestClient,
			pb.FooClient.TestFoo,
		)).Name("Test_TestGrpcGet")
	})

	go func() {
		err := server.Start()
		if err != nil {
			t.Errorf("GRPCCallWithQueryParams Error: %s", err)
		}
	}()

	time.Sleep(1 * time.Second)

	// start client
	client := http.Client{}

	// test
	res, err := client.Get("http://localhost:8020/test-grpc-get?foo=bar")
	if err != nil {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("GRPCCallWithQueryParams Error: %d", res.StatusCode)
	}

	// read response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

	// compare response
	var out pb.ResGrpcTestMessage
	err = json.Unmarshal(body, &out)
	if err != nil {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

	if out.Bar != "bar" {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

	// teardown
	err = server.Stop()
	if err != nil {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

}

func startMockGrpcServer() {
	// create a listener on TCP port 7777
	lis, err := net.Listen("tcp", ":7777")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create a server instance
	s := grpc.NewServer()

	// attach the Greeter service to the server
	pb.RegisterFooServer(s, &pb.FooServerImpl{})

	// start the server
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
