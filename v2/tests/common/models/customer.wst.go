//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Customer.json
var _CustomerRawConfig []byte

func (m *Customer) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Customer) GetRawConfig() []byte {
	return _CustomerRawConfig
}

func (m *Customer) GetModelName() string {
	return "Customer"
}

func (m *Customer) GetCreated() time.Time {
	return m.Created
}
