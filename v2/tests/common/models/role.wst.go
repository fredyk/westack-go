//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed role.json
var _roleRawConfig []byte

func (m *role) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *role) GetRawConfig() []byte {
	return _roleRawConfig
}

func (m *role) GetModelName() string {
	return "role"
}

func (m *role) GetCreated() time.Time {
	return m.Created
}
