package tests

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"

	"github.com/fredyk/westack-go/westack"
	"github.com/fredyk/westack-go/westack/model"

	pb "github.com/fredyk/westack-go/westack/tests/proto"
)

var server *westack.WeStack
var userId primitive.ObjectID
var noteId primitive.ObjectID
var noteModel *model.Model
var systemContext *model.EventContext

func Test_GRPCCallWithQueryParamsOK(t *testing.T) {

	// start client
	client := http.Client{}

	// test for ok
	res, err := client.Get("http://localhost:8020/test-grpc-get?foo=1")
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

	if out.Bar != 1 {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

}

func Test_GRPCCallWithQueryParamsError(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Get("http://localhost:8020/test-grpc-get?foo=")
	if err != nil {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

	if res.StatusCode != 500 {
		t.Errorf("GRPCCallWithQueryParams Error: %d", res.StatusCode)
	}

}

func Test_GRPCCallWithQueryParams_WithBadQueryParams(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Get("http://localhost:8020/test-grpc-get?foo=abc")
	if err != nil {
		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
	}

	if res.StatusCode != 500 {
		t.Errorf("GRPCCallWithQueryParams Error: %d", res.StatusCode)
	}

}

// todo: fix this test
//func Test_GRPCCallWithQueryParams_WithInvalidConnection(t *testing.T) {
//
//	// start client
//	client := http.Client{
//		Timeout: time.Second * 5,
//	}
//
//	// test for error
//	res, err := client.Get("http://localhost:8020/test-grpc-get-invalid?foo=1")
//	if err != nil {
//		t.Errorf("GRPCCallWithQueryParams Error: %s", err)
//		return
//	}
//
//	if res.StatusCode != 500 {
//		t.Errorf("GRPCCallWithQueryParams Error: %d", res.StatusCode)
//	}
//
//}

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

func Test_ReqGrpcTestMessage(t *testing.T) {
	in := pb.ReqGrpcTestMessage{
		Foo: 1,
	}
	compactedPb := in.String()
	compactedJson := "foo:1 "
	assert.Equal(t, compactedJson, compactedPb)

	// just invoke the method to increase coverage
	in.ProtoMessage()

}

func Test_ResGrpcTestMessage(t *testing.T) {
	in := pb.ResGrpcTestMessage{
		Bar: 1,
	}
	compactedPb := in.String()
	compactedJson := "bar:1 "
	assert.Equal(t, compactedJson, compactedPb)

	// just invoke the method to increase coverage
	in.ProtoMessage()

}

// before all tests
func TestMain(m *testing.M) {

	var err error
	userId, err = primitive.ObjectIDFromHex("5f9f1b5b9b9b9b9b9b9b9b9c")
	if err != nil {
		log.Fatal(err)
	}
	noteId, err = primitive.ObjectIDFromHex("5f9f1b5b9b9b9b9b9b9b9b9b")
	if err != nil {
		log.Fatal(err)
	}
	systemContext = &model.EventContext{
		Bearer: &model.BearerToken{User: &model.BearerUser{System: true}},
	}

	// start server
	server = westack.New(westack.Options{
		Port: 8020,
	})

	// start a mock grpc server
	go startMockGrpcServer()

	server.Boot(func(app *westack.WeStack) {
		// for valid connections
		app.Server.Get("/test-grpc-get", westack.GRPCCallWithQueryParams[pb.ReqGrpcTestMessage, pb.FooClient, *pb.ResGrpcTestMessage](
			"localhost:7777",
			pb.NewGrpcTestClient,
			pb.FooClient.TestFoo,
		)).Name("Test_TestGrpcGet")
		//// for invalid connections
		//app.Server.Get("/test-grpc-get-invalid", westack.GRPCCallWithQueryParams[pb.ReqGrpcTestMessage, pb.FooClient, *pb.ResGrpcTestMessage](
		//	"localhost:8020",
		//	pb.NewGrpcTestClient,
		//	pb.FooClient.TestFoo,
		//)).Name("Test_TestGrpcGetInvalid")
	})

	go func() {
		err := server.Start()
		if err != nil {
			log.Fatalf("failed to start: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	noteModel, err = server.FindModel("Note")
	if err != nil {
		log.Fatal(err)
	}

	m.Run()

	// after all tests

	// teardown
	err = server.Stop()
	if err != nil {
		log.Fatalf("failed to stop: %v", err)
	}

}
