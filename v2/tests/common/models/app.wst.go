//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed App.json
var _AppRawConfig []byte

func (m *App) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *App) GetRawConfig() []byte {
	return _AppRawConfig
}

func (m *App) GetModelName() string {
	return "App"
}

func (m *App) GetCreated() time.Time {
	return m.Created
}
