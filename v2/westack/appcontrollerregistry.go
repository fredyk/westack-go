package westack

import (
	"github.com/fredyk/westack-go/v2/lib/swaggerhelper"
	"github.com/fredyk/westack-go/v2/model"
)

type controllerRegistryImpl struct {
	app *WeStack
}

func (c *controllerRegistryImpl) RegisterController(controller model.Controller) {
	swaggerhelper.RegisterGenericComponentForSample(c.app.swaggerHelper, controller)
}
