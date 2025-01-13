package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/v2/lib/uploads"
	"io"
	"log"
	"math/big"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	"github.com/fredyk/westack-go/v2/westack"

	"github.com/fredyk/westack-go/v2/datasource"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/stretchr/testify/assert"

	wst "github.com/fredyk/westack-go/v2/common"
)

func init() {
	app = westack.New(westack.Options{
		DatasourceOptions: &map[string]*datasource.Options{
			"db": {
				MongoDB: &datasource.MongoDBDatasourceOptions{
					Registry: wst.CreateDefaultMongoRegistry(),
					Monitor:  FakeMongoDbMonitor(),
					//Timeout:  3,
				},
				RetryOnError: true,
			},
		},
		Logger: createMockLogger(),
	})
	var err error
	app.Boot(westack.BootOptions{
		RegisterControllers: func(r model.ControllerRegistry) {},
	}, func(app *westack.WeStack) {

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
				ctx.Result, err = noteModel.Build((*ctx.Data)["__overwriteWithInstance"].(wst.M), ctx)
				if err != nil {
					return err
				}
			}
			if (*ctx.Data)["__overwriteWithInstancePointer"] != nil {
				v, err := noteModel.Build((*ctx.Data)["__overwriteWithInstancePointer"].(wst.M), ctx)
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

		userModel, err = app.FindModel("Account")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		userModel.Observe("before save", func(ctx *model.EventContext) error {
			if !ctx.IsNewInstance && ctx.Data.GetString("testEphemeral") == "ephemeralAttribute1503" {
				delete(*ctx.Data, "testEphemeral")
				ctx.BaseContext.UpdateEphemeral(&wst.M{
					"ephemeralAttribute1503": "ephemeralValue1503",
				})
			}
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
		imageModel, err = app.FindModel("Image")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}
		appModel, err = app.FindModel("App")
		if err != nil {
			log.Fatalf("failed to find model: %v", err)
		}

		noteModel.Observe("before load", func(ctx *model.EventContext) error {
			if ctx.BaseContext.Remote != nil {
				if ctx.BaseContext.Query.GetString("mockResultTest124401") == "true" {
					// set the result as model.InstanceA
					inst, err := noteModel.Build(wst.M{
						"title": "mocked124401",
					}, ctx)
					if err != nil {
						return err
					}
					ctx.Result = &model.InstanceA{
						inst,
					}
				} else if ctx.BaseContext.Query.GetString("mockResultTest124402") == "true" {
					// set the result as model.InstanceA
					inst, err := noteModel.Build(wst.M{
						"title": "mocked124402",
					}, ctx)
					if err != nil {
						return err
					}
					ctx.Result = model.InstanceA{
						inst,
					}
				} else if ctx.BaseContext.Query.GetString("mockResultTest124403") == "true" {
					// set the result as []model.InstanceA
					inst, err := noteModel.Build(wst.M{
						"title": "mocked124403",
					}, ctx)
					if err != nil {
						return err
					}
					ctx.Result = []*model.StatefulInstance{
						inst.(*model.StatefulInstance),
					}
				} else if ctx.BaseContext.Query.GetString("mockResultTest124404") == "true" {
					// set the result as wst.A
					ctx.Result = wst.A{
						{"title": "mocked124404"},
					}
				} else if ctx.BaseContext.Query.GetString("forceError1719") == "true" {
					return wst.CreateError(fiber.ErrBadRequest, "ERR_1719", fiber.Map{"message": "forced error 1719"}, "Error")
				}
			}
			return nil
		})

		noteModel.Observe("after load", func(ctx *model.EventContext) error {
			if ctx.BaseContext.Remote != nil {
				if ctx.BaseContext.Query.GetString("forceError1753") == "true" && ctx.Instance.GetString("title") == "Note 3" {
					return fmt.Errorf("forced error 1753")
				}
			}
			return nil
		})

		noteModel.Observe("before build", func(ctx *model.EventContext) error {
			if ctx.BaseContext.Remote != nil {
				if ctx.BaseContext.Query.GetString("forceError1556") == "true" {
					return fmt.Errorf("forced error 1556")
				}
			}
			return nil
		})

		noteModel.RemoteMethod(func(ctx *model.EventContext) error {
			d := ctx.Query.GetString("someDate")
			ctx.Result = wst.M{
				"someDate": d,
			}
			return nil
		}, model.RemoteMethodOptions{
			Name:        "remoteMethodWithDate",
			Description: "",
			Accepts: model.RemoteMethodOptionsHttpArgs{
				{
					Arg:         "someDate",
					Type:        "date",
					Description: "",
					Http:        model.ArgHttp{Source: "query"},
					Required:    true,
				},
			},
			Http: model.RemoteMethodOptionsHttp{
				Path: "/method-with-date",
				Verb: "get",
			},
		})

		model.BindRemoteOperation(noteModel, RemoteOperationExample)
		model.BindRemoteOperationWithOptions(noteModel, RateLimitedOperation, model.RemoteOptions().
			WithRateLimits(
				model.NewRateLimit("rl-1-second", 1, time.Second, false),
				model.NewRateLimit("rl-5-seconds", 4, 5*time.Second, false),
				model.NewRateLimit("rl-14-seconds", 10, 14*time.Second, false),
			))

		minioDomain := os.Getenv("MINIO_DOMAIN")
		minioClient := uploads.MinioClient{
			Bucket:    "wstuploadstest",
			Endpoint:  fmt.Sprintf("%v:443", minioDomain),
			AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
			SecretKey: os.Getenv("MINIO_SECRET_KEY"),
			PublicUrl: fmt.Sprintf("https://%v", minioDomain),
			Region:    "us-east-1",
		}
		model.BindRemoteOperationWithOptions(appModel, minioClient.UploadFile, model.RemoteOptions().
			WithName("upload").
			WithPath("/upload").
			WithContentType("multipart/form-data"))

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
			err = note.(*model.StatefulInstance).Transform(&resultNote)
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

func createAccount(t *testing.T, userData wst.M) wst.M {
	var user wst.M
	var err error
	user, err = wstfuncs.InvokeApiJsonM("POST", "/accounts", userData, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, user, "id")
	return user
}

func login(t *testing.T, body wst.M) (string, string) {
	b := createBody(t, body)
	request := httptest.NewRequest("POST", "/api/v1/accounts/login", b)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Server.Test(request, 45000)
	if err != nil {
		t.Error(err)
		return "", ""
	}

	contentLength, err := strconv.Atoi(response.Header["Content-Length"][0])
	if err != nil {
		t.Error(err)
		return "", ""
	}
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
	err = easyjson.Unmarshal(responseBytes, &loginResponse)
	if err != nil {
		t.Error(err)
		return "", ""
	}

	if assert.NotEmpty(t, loginResponse["id"]) && assert.NotEmpty(t, loginResponse["accountId"]) {
		return loginResponse["id"].(string), loginResponse["accountId"].(string)
	} else {
		t.Error("Wrong response")
		return "", ""
	}
}

func Test_WeStackCreateUser(t *testing.T) {

	t.Parallel()

	randomUserSuffix := createRandomInt()
	email := fmt.Sprintf("email%v@example.com", randomUserSuffix)
	password := "Abcd1234."
	plainUser := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", randomUserSuffix)}
	createAccount(t, plainUser)

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
	password := "Abcd1234."

	log.Println("Email", email)
	plainUser := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	createAccount(t, plainUser)

	login(t, plainUser)

}

func Test_WeStackDelete(t *testing.T) {

	t.Parallel()

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "Abcd1234."
	plainUser := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	createAccount(t, plainUser)

	bearer, userId := login(t, plainUser)

	request := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/accounts/%v", userId), nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", bearer))
	response, err := app.Server.Test(request, 45000)
	if err != nil {
		t.Error(err)
		return
	}

	if !assert.Equal(t, fiber.StatusOK, response.StatusCode) {
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

	// var bootedApp *westack.WeStack
	t.Parallel()

	go func() {
		westack.InitAndServe(westack.Options{Port: 8021, DisablePortEnvVar: true}, func(app *westack.WeStack) {
			// bootedApp = app
		})
	}()

	time.Sleep(3 * time.Second)
	// bootedApp.Stop()

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
			assert.Equal(t, "Error while loading models: could not create policies directory /invalid/path: mkdir /invalid: permission denied", r)
			// mark as ok
			t.Log("OK")
		}
	}()

	app.Boot(westack.BootOptions{
		RegisterControllers: func(r model.ControllerRegistry) {},
	})

}

func Test_InvalidCasbinOutputDirectory1(t *testing.T) {

	t.Parallel()

	app := westack.New(westack.Options{
		Logger: createMockLogger(),
	})
	app.Viper.Set("casbin.policies.outputDirectory", "/home/invalid")

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			assert.Regexp(t, "Error while loading models: could not (check|create) policies directory /home/invalid: (stat|mkdir) /home/invalid: permission denied", r)
		} else {
			t.Error("Should have panicked")
		}
	}()

	app.Boot(westack.BootOptions{
		RegisterControllers: func(r model.ControllerRegistry) {},
	})

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
			assert.Regexp(t, `Error while loading models: could not open policies file Account: open /lib/Account.policies.csv: permission denied`, r)
			// mark as ok
			t.Log("OK")
		} else {
			t.Error("Should have panicked")
		}
	}()

	app.Boot(westack.BootOptions{
		RegisterControllers: func(r model.ControllerRegistry) {},
	})

}

func Test_FindModelNonExistent(t *testing.T) {

	t.Parallel()

	_, err := app.FindModel("NonExistent")
	assert.EqualError(t, err, "model NonExistent not found")

}

func Test_FindDatasourceNonExistent(t *testing.T) {

	t.Parallel()

	_, err := app.FindDatasource("NonExistent")
	assert.EqualError(t, err, "datasource NonExistent not found")

}

func Test_WeStackStop(t *testing.T) {

	t.Parallel()

	app := westack.New(westack.Options{
		Port:              8022,
		DisablePortEnvVar: true,
	})
	app.Boot(westack.BootOptions{
		RegisterControllers: func(r model.ControllerRegistry) {},
	})

	go func() {
		time.Sleep(3 * time.Second)
		err := app.Stop()
		assert.NoError(t, err)
	}()

	err := app.Start()
	assert.NoError(t, err)
	//err = app.Stop()
	//assert.NoError(t, err)

}

func Test_GetWeStackLoggerPrefix(t *testing.T) {

	t.Parallel()

	logger := app.Logger()
	logger.SetPrefix("test")
	assert.Equal(t, "test", logger.Prefix())

}
