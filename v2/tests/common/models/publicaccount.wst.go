//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed PublicAccount.json
var _PublicAccountRawConfig []byte

func (m *PublicAccount) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *PublicAccount) GetRawConfig() []byte {
	return _PublicAccountRawConfig
}

func (m *PublicAccount) GetModelName() string {
	return "PublicAccount"
}

func (m *PublicAccount) GetCreated() time.Time {
	return m.Created
}
