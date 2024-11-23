//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Order.json
var _OrderRawConfig []byte

func (m *Order) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Order) GetRawConfig() []byte {
	return _OrderRawConfig
}

func (m *Order) GetModelName() string {
	return "Order"
}

func (m *Order) GetCreated() time.Time {
	return m.Created
}
