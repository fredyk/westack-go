package boot

import (
	"log"

	"github.com/fredyk/westack-go/westack"
	"github.com/fredyk/westack-go/westack/model"
)

func SetupRoles(app *westack.WeStack) {

	RoleModel, err := app.FindModel("role")
	if err != nil {
		log.Printf("ERROR: SetupRoles() --> %v\n", err)
		return
	}
	RoleModel.Observe("after load", func(eventContext *model.EventContext) error {
		log.Println("loaded role ", eventContext.Data)
		return nil
	})

}
