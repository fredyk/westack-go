//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
)

//go:embed Empty.json
var _EmptyRawConfig []byte

func (m *Empty) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Empty) GetRawConfig() []byte {
	return _EmptyRawConfig
}

func (m *Empty) GetModelName() string {
	return "Empty"
}
