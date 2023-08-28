package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"google.golang.org/grpc"

	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"

	pb "github.com/fredyk/westack-go/westack/tests/proto"
)

type InstanceFromTests struct {
	id    primitive.ObjectID
	model *model.Model
}

var server *westack.WeStack
var userId primitive.ObjectID
var noteId primitive.ObjectID
var noteModel *model.Model
var userModel *model.Model
var customerModel *model.Model
var orderModel *model.Model
var storeModel *model.Model
var footerModel *model.Model
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

func Test_GRPCCallWithBodyParamsOK(t *testing.T) {

	t.Parallel()

	// start client
	client := http.Client{}

	// test for ok
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":1}`)))
	assert.Nil(t, err)
	assert.Equal(t, 200, res.StatusCode)

	// read response
	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	// compare response
	var out pb.ResGrpcTestMessage
	err = json.Unmarshal(body, &out)
	assert.Nil(t, err)
	assert.Equal(t, int32(1), out.Bar)

}

func Test_GRPCCallWithBodyParamsError(t *testing.T) {

	t.Parallel()

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":"abc"}`)))
	assert.Nil(t, err)
	assert.Equal(t, 500, res.StatusCode)

}

func Test_GRPCCallWithBodyParams_WithBadBody(t *testing.T) {

	t.Parallel()

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":abc}`)))
	assert.Nil(t, err)
	assert.Equal(t, 500, res.StatusCode)

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
		DatasourceOptions: &map[string]*datasource.Options{
			"db": {
				MongoDB: &datasource.MongoDBDatasourceOptions{
					Registry: FakeMongoDbRegistry(),
					Monitor:  FakeMongoDbMonitor(),
					Timeout:  3,
				},
				RetryOnError: true,
			},
		},
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
		app.Server.Post("/test-grpc-post", westack.GRPCCallWithBody[pb.ReqGrpcTestMessage, pb.FooClient, *pb.ResGrpcTestMessage](
			"localhost:7777",
			pb.NewGrpcTestClient,
			pb.FooClient.TestFoo,
		)).Name("Test_TestGrpcPost")
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

	exitCode := m.Run()

	// after all tests
	err = revertAllTests()
	if err != nil {
		log.Fatalf("failed to revert all tests: %v", err)
	}

	// teardown
	err = server.Stop()
	if err != nil {
		log.Fatalf("failed to stop: %v", err)
	}

	fmt.Printf("exit code: %d\n", exitCode)
	os.Exit(exitCode)

}

func FakeMongoDbMonitor() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, cmd *event.CommandStartedEvent) {
		},
		Succeeded: func(ctx context.Context, cmd *event.CommandSucceededEvent) {
		},
		Failed: func(ctx context.Context, cmd *event.CommandFailedEvent) {
		},
	}
}

func FakeMongoDbRegistry() *bsoncodec.Registry {
	// create a new registry
	registryBuilder := bson.NewRegistryBuilder().
		//RegisterTypeMapEntry(bson.TypeEmbeddedDocument, reflect.TypeOf(bson.M{})).
		RegisterTypeMapEntry(bson.TypeEmbeddedDocument, reflect.TypeOf(wst.M{})).
		//RegisterTypeMapEntry(bson.TypeArray, reflect.TypeOf([]bson.M{}))
		RegisterTypeMapEntry(bson.TypeArray, reflect.TypeOf(wst.A{}))

	// register the custom types
	registryBuilder.RegisterTypeEncoder(reflect.TypeOf(time.Time{}), bsoncodec.ValueEncoderFunc(func(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
		return vw.WriteDateTime(val.Interface().(time.Time).UnixNano() / int64(time.Millisecond))
	}))

	return registryBuilder.Build()
}

func revertAllTests() error {
	sharedDeleteManyWhere := &wst.Where{
		"_id": wst.M{
			"$ne": nil,
		},
	}
	for _, toDeleteMap := range []*model.Model{
		noteModel,
		userModel,
		customerModel,
		orderModel,
		storeModel,
		footerModel,
	} {
		deleteManyResult, err := toDeleteMap.DeleteMany(sharedDeleteManyWhere, systemContext)
		if err != nil {
			return err
		}
		fmt.Printf("Deleted %d instances from model %s\n", deleteManyResult.DeletedCount, toDeleteMap.Name)
	}
	return nil
}
