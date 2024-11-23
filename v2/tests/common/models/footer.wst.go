//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Footer.json
var _FooterRawConfig []byte

func (m *Footer) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Footer) GetRawConfig() []byte {
	return _FooterRawConfig
}

func (m *Footer) GetModelName() string {
	return "Footer"
}

func (m *Footer) GetCreated() time.Time {
	return m.Created
}
