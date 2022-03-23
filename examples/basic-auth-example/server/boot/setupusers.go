package boot

import (
	"github.com/fredyk/westack-go/westack"
	"github.com/fredyk/westack-go/westack/model"
	"github.com/gofiber/fiber/v2"
	"log"
	"time"
)

func SetupUsers(app *westack.WeStack) {

	userModel := app.FindModel("user")

	userModel.Observe("before save", func(eventContext *model.EventContext) error {
		log.Println("Before saving ", eventContext.Data, eventContext.IsNewInstance)
		timeNow := time.Now()
		if eventContext.IsNewInstance {
			eventContext.Data["created"] = timeNow
		}
		eventContext.Data["modified"] = timeNow
		return nil
	})

	userModel.Observe("after save", func(eventContext *model.EventContext) error {
		log.Println("After saving ", eventContext.Instance, eventContext.IsNewInstance)
		return nil
	})

	//userModel.On("login", func(ctx *model.EventContext) error {
	//	log.Println("login instance ", ctx.Instance)
	//	log.Println("login data ", ctx.Data)
	//	ctx.Result = fiber.Map{"status": "Override login", "initial": ctx.Result}
	//	return nil
	//})

	userModel.RemoteMethod(func(context *model.EventContext) error {
		return userModel.SendError(context.Ctx, (context).RestError(fiber.ErrTeapot, fiber.Map{"error": "I used to be a cup"}))
	}, model.RemoteMethodOptions{
		Name:        "exampleMethod",
		Description: "Example error",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/example-error",
			Verb: "get",
		},
	})

	//userModel.RemoteMethod(func(c *fiber.Ctx) error {
	//	TODO:
	//}, model.RemoteMethodOptions{
	//	Description: "Example error",
	//	Http: model.RemoteMethodOptionsHttp{
	//		Path: "/api/users/me",
	//		Verb: "get",
	//	},
	//})

}
