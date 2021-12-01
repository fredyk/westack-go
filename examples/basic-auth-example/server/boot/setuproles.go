package boot

import (
	"github.com/fredyk/westack-go/westack"
	"github.com/fredyk/westack-go/westack/model"
	"log"
)

func SetupRoles(app *westack.WeStack) {

	app.FindModel("role").Observe("loaded", func(eventContext *model.EventContext) error {
		log.Println("loaded role ", eventContext.Data)
		return nil
	})

}
