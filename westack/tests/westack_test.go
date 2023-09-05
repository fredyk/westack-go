package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/westack/datasource"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
					//Registry: FakeMongoDbRegistry(),
					Monitor: FakeMongoDbMonitor(),
					//Timeout:  3,
				},
				RetryOnError: true,
			},
		},
		Logger: createMockLogger(),
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

		userModel.On("sendResetPasswordEmail", func(ctx *model.EventContext) error {
			fmt.Println("sending reset password email")
			ctx.Result = wst.M{
				"result":  "OK",
				"message": "Reset password email sent",
			}
			return nil
		})

		userModel.On("sendVerificationEmail", func(ctx *model.EventContext) error {
			fmt.Println("sending verify email")
			// This bearer would be sent in the email in a real case, because it contains
			// a special claim that allows the user to verify the email
			bearerForEmailVerification := ctx.Bearer.Raw
			ctx.Result = wst.M{
				"result":  "OK",
				"message": "Verification email sent",
				"bearer":  bearerForEmailVerification,
			}
			return nil
		})

		userModel.On("performEmailVerification", func(ctx *model.EventContext) error {
			fmt.Println("performing email verification")
			ctx.Result = wst.M{
				"result":  "OK",
				"message": "Email verified",
			}
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
				} else if ctx.BaseContext.Ctx.Query("forceError1719") == "true" {
					return wst.CreateError(fiber.ErrBadRequest, "ERR_1719", fiber.Map{"message": "forced error 1719"}, "Error")
				}
			}
			return nil
		})

		app.Server.Get("/api/v1/endpoint-using-codecs", func(ctx *fiber.Ctx) error {
			type localNote struct {
				ID      primitive.ObjectID `json:"id" bson:"_id"`
				Title   string             `json:"title" bson:"title"`
				Content string             `json:"content" bson:"content"`
				DueDate time.Time          `json:"dueDate" bson:"dueDate"`
			}
			var noteToInsert localNote
			noteToInsert.ID = primitive.NewObjectID()
			noteToInsert.Title = "test"
			noteToInsert.Content = "test"
			noteToInsert.DueDate = time.Now().Add(7 * 24 * time.Hour)
			note, err := noteModel.Create(noteToInsert, systemContext)
			if err != nil {
				return err
			}
			var resultNote localNote
			err = note.Transform(&resultNote)
			if err != nil {
				return err
			}
			return ctx.JSON(resultNote)
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

type mockLogger struct {
	output          io.Writer
	flags           int
	internalLogger  *log.Logger
	prefix          string
	lastFatalOutput interface{}
}

func (l *mockLogger) Printf(format string, v ...any) {
	fmt.Printf(format, v...)
}

func (l *mockLogger) Print(v ...any) {
	fmt.Print(v...)
}
func (l *mockLogger) Println(v ...any) {
	fmt.Println(v...)
}
func (l *mockLogger) Fatal(v ...any) {
	st := fmt.Sprint(v...)
	l.lastFatalOutput = st
	fmt.Print(v...)
	panic(st)
}
func (l *mockLogger) Fatalf(format string, v ...any) {
	st := fmt.Sprintf(format, v...)
	l.lastFatalOutput = st
	fmt.Print(st)
	panic(st)
}
func (l *mockLogger) Fatalln(v ...any) {
	st := fmt.Sprintf("%v\n", v...)
	l.lastFatalOutput = st
	fmt.Print(st)
	panic(st)
}
func (l *mockLogger) Panic(v ...any) {
	panic(fmt.Sprintf("%v", v...))
}
func (l *mockLogger) Panicf(format string, v ...any) {
	panic(fmt.Sprintf(format, v...))
}
func (l *mockLogger) Panicln(v ...any) {
	panic(fmt.Sprintf("%v\n", v...))
}

func (l *mockLogger) Flags() int {
	return l.flags
}
func (l *mockLogger) SetFlags(flag int) {
	l.flags = flag
}
func (l *mockLogger) Prefix() string {
	return l.prefix
}

func (l *mockLogger) SetPrefix(prefix string) {
	l.prefix = prefix
}

func createMockLogger() wst.ILogger {
	return &mockLogger{
		internalLogger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func createUser(t *testing.T, userData wst.M) wst.M {
	var user wst.M
	var err error
	user, err = invokeApiJsonM(t, "POST", "/users", userData, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, user, "id")
	return user
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
	plainUser := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", randomUserSuffix)}
	createUser(t, plainUser)

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
	plainUser := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	createUser(t, plainUser)

	login(t, plainUser)

}

func Test_WeStackDelete(t *testing.T) {

	t.Parallel()

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"
	plainUser := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	createUser(t, plainUser)

	bearer, userId := login(t, plainUser)

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

func Test_InitAndServe(t *testing.T) {

	t.Parallel()

	go func() {
		westack.InitAndServe(westack.Options{Port: 8021})
	}()

	time.Sleep(5 * time.Second)

}

func Test_MissingCasbinOutputDirectory(t *testing.T) {

	t.Parallel()

	app := westack.New(westack.Options{
		Logger: createMockLogger(),
	})
	app.Viper.Set("casbin.policies.outputDirectory", "/invalid/path")

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, "failed to create casbin model: mkdir /invalid: permission denied", r)
			// mark as ok
			t.Log("OK")
		}
	}()

	app.Boot()

}

func Test_InvalidCasbinOutputDirectory1(t *testing.T) {

	t.Parallel()

	app := westack.New(westack.Options{
		Logger: createMockLogger(),
	})
	app.Viper.Set("casbin.policies.outputDirectory", "/proc/1/cwd/a")

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, "failed to create casbin model: stat /proc/1/cwd/a: permission denied", r)
			// mark as ok
			t.Log("OK")
		} else {
			t.Error("Should have panicked")
		}
	}()

	app.Boot()

}

func Test_InvalidCasbinOutputDirectory2(t *testing.T) {

	t.Parallel()

	app := westack.New(westack.Options{
		Logger: createMockLogger(),
	})
	app.Viper.Set("casbin.policies.outputDirectory", "/lib")

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			assert.Regexp(t, `failed to create casbin model: open /lib/\w+.policies.csv: permission denied`, r)
			// mark as ok
			t.Log("OK")
		} else {
			t.Error("Should have panicked")
		}
	}()

	app.Boot()

}
