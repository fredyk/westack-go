package server

import (
	"github.com/fredyk/westack-go/examples/basic-auth-example/server/boot"
)

func ServerBoot(app *westack.WeStack) {

	boot.SetupRoles(app)
	boot.SetupUsers(app)
	boot.SetupNotes(app)

}
