package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
	"io"
	"log"
	"math/big"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
)

func init() {
	app = westack.New(westack.Options{
		DatasourceOptions: &map[string]*datasource.Options{
			"db": {
				MongoDB: &datasource.MongoDBDatasourceOptions{
					Registry: FakeMongoDbRegistry(),
					Monitor:  FakeMongoDbMonitor(),
					//Timeout:  3,
				},
				RetryOnError: true,
			},
		},
	})
	var err error
	app.Boot(func(app *westack.WeStack) {

		// Some hooks
		noteModel, err = app.FindModel("Note")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		noteModel.Observe("before save", func(ctx *model.EventContext) error {
			if ctx.IsNewInstance {
				(*ctx.Data)["__test"] = true
				(*ctx.Data)["__testDate"] = time.Now()
			}
			if (*ctx.Data)["__forceError"] == true {
				return fmt.Errorf("forced error")
			}
			if (*ctx.Data)["__overwriteWith"] != nil {
				ctx.Result = (*ctx.Data)["__overwriteWith"]
			}
			if (*ctx.Data)["__overwriteWithInstance"] != nil {
				ctx.Result, err = noteModel.Build((*ctx.Data)["__overwriteWithInstance"].(wst.M), model.NewBuildCache(), ctx)
				if err != nil {
					return err
				}
			}
			if (*ctx.Data)["__overwriteWithInstancePointer"] != nil {
				v, err := noteModel.Build((*ctx.Data)["__overwriteWithInstancePointer"].(wst.M), model.NewBuildCache(), ctx)
				if err != nil {
					return err
				}
				ctx.Result = &v
			}
			return nil
		})

		noteModel.Observe("after save", func(ctx *model.EventContext) error {
			if (*ctx.Data)["__forceAfterError"] == true {
				return fmt.Errorf("forced error")
			}
			return nil
		})

		userModel, err = app.FindModel("user")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		userModel.Observe("before save", func(ctx *model.EventContext) error {
			fmt.Println("saving user")
			return nil
		})

		customerModel, err = app.FindModel("Customer")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		orderModel, err = app.FindModel("Order")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		storeModel, err = app.FindModel("Store")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		footerModel, err = app.FindModel("Footer")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}

		noteModel.Observe("before load", func(ctx *model.EventContext) error {
			if ctx.BaseContext.Remote != nil {
				if ctx.BaseContext.Ctx.Query("mockResultTest124401") == "true" {
					// set the result as *model.InstanceA
					inst, err := noteModel.Build(wst.M{
						"title": "mocked124401",
					}, model.NewBuildCache(), ctx)
					if err != nil {
						return err
					}
					ctx.Result = &model.InstanceA{
						inst,
					}
				} else if ctx.BaseContext.Ctx.Query("mockResultTest124402") == "true" {
					// set the result as model.InstanceA
					inst, err := noteModel.Build(wst.M{
						"title": "mocked124402",
					}, model.NewBuildCache(), ctx)
					if err != nil {
						return err
					}
					ctx.Result = model.InstanceA{
						inst,
					}
				} else if ctx.BaseContext.Ctx.Query("mockResultTest124403") == "true" {
					// set the result as []*model.InstanceA
					inst, err := noteModel.Build(wst.M{
						"title": "mocked124403",
					}, model.NewBuildCache(), ctx)
					if err != nil {
						return err
					}
					ctx.Result = []*model.Instance{
						&inst,
					}
				} else if ctx.BaseContext.Ctx.Query("mockResultTest124404") == "true" {
					// set the result as wst.A
					ctx.Result = wst.A{
						{"title": "mocked124404"},
					}
				}
			}
			return nil
		})
	})
	go func() {
		err := app.Start()
		if err != nil {
			fmt.Printf("Error while starting server: %v", err)
			os.Exit(1)
		}
	}()
	time.Sleep(300 * time.Millisecond)
}

func createUser(t *testing.T, userData wst.M) (wst.M, error) {
	b := createBody(t, userData)

	request := httptest.NewRequest("POST", "/api/v1/users", b)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Server.Test(request, 45000)
	if err != nil {
		return nil, err
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return nil, fmt.Errorf("expected status code 200, got %v", response.StatusCode)
	}

	var responseBytes []byte
	responseBytes, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var responseMap wst.M
	err = json.Unmarshal(responseBytes, &responseMap)
	if err != nil {
		return nil, err
	}

	assert.Contains(t, responseMap, "id")
	return responseMap, nil
}

func login(t *testing.T, body wst.M) (string, string) {
	b := createBody(t, body)
	request := httptest.NewRequest("POST", "/api/v1/users/login", b)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Server.Test(request, 45000)
	if err != nil {
		t.Error(err)
		return "", ""
	}

	contentLength, err := strconv.Atoi(response.Header["Content-Length"][0])
	responseBytes := make([]byte, contentLength)
	count, err := response.Body.Read(responseBytes)
	if err != nil && err != io.EOF {
		t.Error(err)
		return "", ""
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return "", ""
	}

	if !assert.Equal(t, count, contentLength) {
		return "", ""
	}

	var loginResponse wst.M
	err = json.Unmarshal(responseBytes, &loginResponse)
	if err != nil {
		t.Error(err)
		return "", ""
	}

	if assert.NotEmpty(t, loginResponse["id"]) && assert.NotEmpty(t, loginResponse["userId"]) {
		return loginResponse["id"].(string), loginResponse["userId"].(string)
	} else {
		t.Error("Wrong response")
		return "", ""
	}
}

func Test_WeStackCreateUser(t *testing.T) {

	t.Parallel()

	randomUserSuffix := createRandomInt()
	email := fmt.Sprintf("email%v@example.com", randomUserSuffix)
	password := "test"
	body := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", randomUserSuffix)}
	user, err := createUser(t, body)
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Contains(t, user, "id")

}

func createBody(t *testing.T, body wst.M) *bytes.Buffer {
	bodyBytes := new(bytes.Buffer)
	if err := json.NewEncoder(bodyBytes).Encode(body); err != nil {
		t.Error(err)
		return nil
	}
	return bodyBytes
}

func Test_WeStackLogin(t *testing.T) {

	t.Parallel()

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"

	log.Println("Email", email)
	body := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	user, err := createUser(t, body)
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Contains(t, user, "id")

	login(t, body)

}

func Test_WeStackDelete(t *testing.T) {

	t.Parallel()

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"
	body := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	user, err := createUser(t, body)
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Contains(t, user, "id")

	bearer, userId := login(t, body)

	request := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%v", userId), nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", bearer))
	response, err := app.Server.Test(request, 45000)
	if err != nil {
		t.Error(err)
		return
	}

	if !assert.Equal(t, 204, response.StatusCode) {
		return
	}

}

//// after all tests, stop the server
//func TestMain(m *testing.M) {
//	code := m.Run()
//	err := app.Stop()
//	if err != nil {
//		log.Fatal(err)
//	}
//	os.Exit(code)
//}
