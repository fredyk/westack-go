package boot

import (
	"github.com/fredyk/westack-go/v2"
	"github.com/fredyk/westack-go/v2/model"
	"log"
)

func SetupUsers(app *westack.WeStack) {

	userModel, err := app.FindModel("user")
	if err != nil {
		log.Printf("ERROR: SetupUsers() --> %v\n", err)
		return
	}
	userModel.Observe("before save", func(eventContext *model.EventContext) error {
		log.Println("Before saving ", eventContext.Data, eventContext.IsNewInstance)
		return nil
	})

	userModel.Observe("after save", func(eventContext *model.EventContext) error {
		log.Println("After saving ", eventContext.Instance, eventContext.IsNewInstance)
		return nil
	})

}
