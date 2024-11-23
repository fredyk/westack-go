//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed NoteEntry.json
var _NoteEntryRawConfig []byte

func (m *NoteEntry) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *NoteEntry) GetRawConfig() []byte {
	return _NoteEntryRawConfig
}

func (m *NoteEntry) GetModelName() string {
	return "NoteEntry"
}

func (m *NoteEntry) GetCreated() time.Time {
	return m.Created
}
