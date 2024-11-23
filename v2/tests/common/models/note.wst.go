//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed Note.json
var _NoteRawConfig []byte

func (m *Note) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *Note) GetRawConfig() []byte {
	return _NoteRawConfig
}

func (m *Note) GetModelName() string {
	return "Note"
}

func (m *Note) GetCreated() time.Time {
	return m.Created
}
