//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Image.json
var _ImageRawConfig []byte

func (m *Image) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Image) GetRawConfig() []byte {
	return _ImageRawConfig
}

func (m *Image) GetModelName() string {
	return "Image"
}

func (m *Image) GetCreated() time.Time {
	return m.Created
}
