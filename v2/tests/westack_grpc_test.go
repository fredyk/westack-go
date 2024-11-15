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
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"google.golang.org/grpc"

	"github.com/fredyk/westack-go/v2"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
	"github.com/fredyk/westack-go/v2/model"

	pb "github.com/fredyk/westack-go/v2/tests/proto"
)

var server *westack.WeStack
var userId primitive.ObjectID
var noteId primitive.ObjectID
var noteModel *model.StatefulModel
var userModel *model.StatefulModel
var customerModel *model.StatefulModel
var orderModel *model.StatefulModel
var storeModel *model.StatefulModel
var footerModel *model.StatefulModel
var imageModel *model.StatefulModel
var appModel *model.StatefulModel
var systemContext *model.EventContext

func Test_GRPCCalls(t *testing.T) {
	t.Parallel()
	var wg1 sync.WaitGroup
	wg1.Add(2)
	t.Run("GRPCQueryWithTimeout", func(t *testing.T) {
		defer wg1.Done()
		testGRPCCallWithQueryParamsWithQueryParamsTimeout(t)
	})
	t.Run("GRPCBodyWithTimeout", func(t *testing.T) {
		defer wg1.Done()
		testGRPCCallWithQueryParamsWithBodyTimeout(t)
	})
	wg1.Wait()
	t.Run("GRPCCalls/TestGRPCCallWithQueryParamsOK", testGRPCCallWithQueryParamsOK)
	t.Run("GRPCCalls/TestGRPCCallWithQueryParamsError", testGRPCCallWithQueryParamsError)
	t.Run("GRPCCalls/TestGRPCCallWithQueryParamsWithBadQueryParams", testGRPCCallWithQueryParamsWithBadQueryParams)
	t.Run("GRPCCalls/TestGRPCCallWithBodyParamsOK", testGRPCCallWithBodyParamsOK)
	t.Run("GRPCCalls/TestGRPCCallWithBodyParamsError1", testGRPCCallWithBodyParamsError1)
	t.Run("GRPCCalls/TestGRPCCallWithBodyParamsError2", testGRPCCallWithBodyParamsError2)
	t.Run("GRPCCalls/TestGRPCCallWithBodyParamsWithBadBody", testGRPCCallWithBodyParamsWithBadBody)

}

func testGRPCCallWithQueryParamsWithQueryParamsTimeout(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Get("http://localhost:8020/test-grpc-get?foo=1&timeout=0.000001")
	assert.NoError(t, err)
	bytes, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	var parsedResp wst.M
	err = easyjson.Unmarshal(bytes, &parsedResp)
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusInternalServerError, parsedResp.GetInt("error.statusCode"))
	assert.Equal(t, "context deadline exceeded", parsedResp.GetString("error.message"))

}

func testGRPCCallWithQueryParamsWithBodyTimeout(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Post("http://localhost:8020/test-grpc-post?timeout=0.000001", "application/json", bufio.NewReader(strings.NewReader(`{"foo":1}`)))
	assert.NoError(t, err)
	bytes, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	var parsedResp wst.M
	err = easyjson.Unmarshal(bytes, &parsedResp)
	assert.NoError(t, err)
	assert.EqualValues(t, fiber.StatusInternalServerError, parsedResp.GetInt("error.statusCode"))
	assert.Equal(t, "context deadline exceeded", parsedResp.GetString("error.message"))

}

func testGRPCCallWithQueryParamsOK(t *testing.T) {

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

func testGRPCCallWithQueryParamsError(t *testing.T) {

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

func testGRPCCallWithQueryParamsWithBadQueryParams(t *testing.T) {

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

func testGRPCCallWithBodyParamsOK(t *testing.T) {

	// start client
	client := http.Client{}

	// test for ok
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":1}`)))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)

	// read response
	body, err := io.ReadAll(res.Body)
	assert.NoError(t, err)

	// compare response
	var out pb.ResGrpcTestMessage
	err = json.Unmarshal(body, &out)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), out.Bar)

}

func testGRPCCallWithBodyParamsError1(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":"abc"}`)))
	assert.NoError(t, err)
	assert.Equal(t, 500, res.StatusCode)

}

func testGRPCCallWithBodyParamsError2(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":2}`)))
	assert.NoError(t, err)
	assert.Equal(t, 500, res.StatusCode)

}

func testGRPCCallWithBodyParamsWithBadBody(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Post("http://localhost:8020/test-grpc-post", "application/json", bufio.NewReader(strings.NewReader(`{"foo":abc}`)))
	assert.NoError(t, err)
	assert.Equal(t, 500, res.StatusCode)

}

// todo: fix this test
//func Test_GRPCCallWithQueryParamsWithInvalidConnection(t *testing.T) {
//
//	// start client
//	client := http.Client{
//		Timeout: time.Second * 5,
//	}
//
//	// test for error
//	res, err := client.GetAt("http://localhost:8020/test-grpc-get-invalid?foo=1")
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

func startMockGrpcServer() {
	// create a listener on TCP port 7777
	lis, err := net.Listen("tcp", "localhost:7777")
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

func Test_SpecialFilterNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilterNow_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"title": randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$now")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilterToday(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilterToday_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 24 hours ago
		"created": time.Now().Add(-24 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$today")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilterYesterday(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilterYesterday_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 48 hours ago
		"created": time.Now().Add(-48 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$yesterday")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilterTomorrow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilterTomorrow_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"title": randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$tomorrow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter7DaysAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter7DaysAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 8 days ago
		"created": time.Now().Add(-8 * 24 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$7dago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter4WeeksAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter4WeeksAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 5 weeks ago
		"created": time.Now().Add(-5 * 7 * 24 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$4wago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter3MonthsAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter3MonthsAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 4 months ago
		"created": time.Now().Add(-4 * 30 * 24 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$3mago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter2YearsAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter2YearsAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 3 years ago
		"created": time.Now().Add(-3 * 365 * 24 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$2yago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter15SecondsAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter15SecondsAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 16 seconds ago
		"created": time.Now().Add(-16 * time.Second),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$15Sago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter10MinutesAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter10MinutesAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 11 minutes ago
		"created": time.Now().Add(-11 * time.Minute),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$10Mago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter5HoursAgo(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter5HoursAgo_%d", createRandomInt())
	note, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		// Created 6 hours ago
		"created": time.Now().Add(-6 * time.Hour),
		"title":   randomTitle,
	}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	notes, err := testSpecialDatePlaceholder(t, "$lte", "$5Hago")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), randomTitle)

}

func Test_SpecialFilter7DaysFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter7DaysFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 8 days from now
		"created": time.Now().Add(8 * 24 * time.Hour),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$7dfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))

}

func Test_SpecialFilter4WeeksFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter4WeeksFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 5 weeks from now
		"created": time.Now().Add(5 * 7 * 24 * time.Hour),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$4wfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))

}

func Test_SpecialFilter3MonthsFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter3MonthsFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 4 months from now
		"created": time.Now().Add(4 * 30 * 24 * time.Hour),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$3mfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))

}

func Test_SpecialFilter2YearsFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter2YearsFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 3 years from now
		"created": time.Now().Add(3 * 365 * 24 * time.Hour),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$2yfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))

}

func Test_SpecialFilter15SecondsFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter15SecondsFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 30 seconds from now
		"created": time.Now().Add(30 * time.Second),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$15Sfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))

}

func Test_SpecialFilter10MinutesFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter10MinutesFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 11 minutes from now
		"created": time.Now().Add(11 * time.Minute),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$10Mfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))

}

func Test_SpecialFilter5HoursFromNow(t *testing.T) {

	t.Parallel()

	// Create a note for checking the filter
	randomTitle := fmt.Sprintf("Test_SpecialFilter5HoursFromNow_%d", createRandomInt())
	note, err := noteModel.Create(wst.M{
		// Created 6 hours from now
		"created": time.Now().Add(6 * time.Hour),
		"title":   randomTitle,
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)
	assert.Contains(t, note.ToJSON(), "id")
	notes, err := testSpecialDatePlaceholder(t, "$gte", "$5Hfromnow")
	assert.NoError(t, err)
	assert.NotEmptyf(t, notes, "There should be at least one note")
	assert.Contains(t, reduceByKey(notes, "title"), note.GetString("title"))
}

func testSpecialDatePlaceholder(t *testing.T, specialDateKey string, specialDatePlaceholder string) (wst.A, error) {
	filter := fmt.Sprintf("{\"where\":{\"created\":{\"%s\":\"%s\"}}}", specialDateKey, specialDatePlaceholder)
	encodedFilter := encodeUriComponent(filter)
	// use fmt
	endpointWithFilter := fmt.Sprintf("/notes?filter=%s", encodedFilter)
	fmt.Printf("Original filter: %s\n", filter)
	fmt.Printf("Sending query: %s\n", endpointWithFilter)
	out, err := invokeApiJsonA(t, "GET", endpointWithFilter, nil, wst.M{
		"Authorization": "Bearer " + randomUserToken.GetString("id"),
	})
	assert.NoError(t, err)
	return out, err
}

func encodeUriComponent(st string) string {
	return url.QueryEscape(st)
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
		Bearer: &model.BearerToken{Account: &model.BearerAccount{System: true}},
	}

	// start server
	server = westack.New(westack.Options{
		Port:              8020,
		DisablePortEnvVar: true,
		DatasourceOptions: &map[string]*datasource.Options{
			"db": {
				MongoDB: &datasource.MongoDBDatasourceOptions{
					//Registry: FakeMongoDbRegistry(),
					Monitor: FakeMongoDbMonitor(),
					Timeout: 3,
				},
				RetryOnError: true,
			},
		},
	})

	// start a mock grpc server
	go startMockGrpcServer()

	server.Boot(func(app *westack.WeStack) {
		// for valid connections
		app.Server.Get("/test-grpc-get", func(ctx *fiber.Ctx) error {
			var timeout float32 = 15.0
			if v := ctx.QueryFloat("timeout"); v > 0 {
				timeout = float32(v)
			}
			return westack.GRPCCallWithQueryParams(
				"localhost:7777",
				pb.NewGrpcTestClient,
				pb.FooClient.TestFoo,
				timeout,
			)(ctx)
		}).Name("Test_TestGrpcGet")
		app.Server.Post("/test-grpc-post", func(ctx *fiber.Ctx) error {
			var timeout float32 = 15.0
			if v := ctx.QueryFloat("timeout"); v > 0 {
				timeout = float32(v)
			}
			return westack.GRPCCallWithBody(
				"localhost:7777",
				pb.NewGrpcTestClient,
				pb.FooClient.TestFoo,
				timeout,
			)(ctx)
		}).Name("Test_TestGrpcPost")
		//// for invalid connections
		//app.Server.Get("/test-grpc-get-invalid", westack.GRPCCallWithQueryParams[pb.ReqGrpcTestMessage, pb.FooClient, *pb.ResGrpcTestMessage](
		//	"localhost:8020",
		//	pb.NewGrpcTestClient,
		//	pb.FooClient.TestFoo,
		//)).Name("Test_TestGrpcGetInvalid")

	})

	userN := createRandomInt()
	plainUser := wst.M{
		"email":    fmt.Sprintf("user-%d@example.com", userN),
		"username": fmt.Sprintf("user_%d", userN),
		"password": "abcd1234.",
	}
	var t *testing.T = new(testing.T)
	// Instantiate a test here using m
	randomUser = createUser(t, plainUser)
	randomUserToken, err = loginUser(plainUser.GetString("username"), plainUser.GetString("password"), t)
	if err != nil {
		log.Fatalf("failed to login user: %v", err)
	}
	if randomUserToken.GetString("id") == "" {
		log.Fatalf("failed to login user: %v", err)
	}

	adminUserToken, err = loginUser(os.Getenv("WST_ADMIN_USERNAME"), os.Getenv("WST_ADMIN_PWD"), t)
	if err != nil {
		log.Fatalf("failed to login admin user: %v", err)
	}
	if adminUserToken.GetString("id") == "" {
		log.Fatalf("failed to login admin user: %v", err)
	}
	appInstance, err = createAppInstance()
	if err != nil {
		log.Fatalf("failed to create app instance: %v", err)
	}
	appBearer = model.CreateBearer(appInstance.Id, float64(time.Now().Unix()), float64(600), []string{"APP"})

	go func() {
		err := server.Start()
		if err != nil {
			log.Fatalf("failed to start: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	var exitCode int
	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error in tests:", r)
			exitCode = 1
		}
	}()

	exitCode = m.Run()

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

func createAppInstance() (*model.StatefulInstance, error) {
	// Create an app instance
	inst, err := appModel.Create(wst.M{
		"name": "westack-tests",
	}, systemContext)
	return inst.(*model.StatefulInstance), err
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

func revertAllTests() error {
	sharedDeleteManyWhere := &wst.Where{
		"_id": wst.M{
			"$ne": nil,
		},
	}
	for _, modelToPurge := range []*model.StatefulModel{
		noteModel,
		userModel,
		customerModel,
		orderModel,
		storeModel,
		footerModel,
		appModel,
		imageModel,
	} {
		deleteManyResult, err := modelToPurge.DeleteMany(sharedDeleteManyWhere, systemContext)
		if err != nil {
			return err
		}
		fmt.Printf("Deleted %d instances from model %s\n", deleteManyResult.DeletedCount, modelToPurge.Name)
		// Drop db
		ds := modelToPurge.Datasource
		if ds.SubViper.GetString("connector") == "mongodb" {
			// drop database
			mongoDatabase := ds.SubViper.GetString("database")
			err = ds.Db.(*mongo.Client).Database(mongoDatabase).Drop(context.Background())
			if err != nil {
				return err
			} else {
				fmt.Printf("Dropped database %s\n", mongoDatabase)
			}
		}
	}
	return nil
}
