package boot

import (
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/westack"
	"log"
)

func SetupUsers(app *westack.WeStack) {

	accountModel, err := app.FindModel("account")
	if err != nil {
		log.Printf("ERROR: SetupUsers() --> %v\n", err)
		return
	}
	accountModel.Observe("before save", func(eventContext *model.EventContext) error {
		log.Println("Before saving ", eventContext.Data, eventContext.IsNewInstance)
		return nil
	})

	accountModel.Observe("after save", func(eventContext *model.EventContext) error {
		log.Println("After saving ", eventContext.Instance, eventContext.IsNewInstance)
		return nil
	})

}
