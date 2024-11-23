//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Store.json
var _StoreRawConfig []byte

func (m *Store) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Store) GetRawConfig() []byte {
	return _StoreRawConfig
}

func (m *Store) GetModelName() string {
	return "Store"
}

func (m *Store) GetCreated() time.Time {
	return m.Created
}
