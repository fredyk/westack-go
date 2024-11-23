//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Account.json
var _AccountRawConfig []byte

func (m *Account) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Account) GetRawConfig() []byte {
	return _AccountRawConfig
}

func (m *Account) GetModelName() string {
	return "Account"
}

func (m *Account) GetCreated() time.Time {
	return m.Created
}
