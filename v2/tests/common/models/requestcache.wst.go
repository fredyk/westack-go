//wst:generated Don't edit this file
package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

//go:embed RequestCache.json
var _RequestCacheRawConfig []byte

func (m *RequestCache) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *RequestCache) GetRawConfig() []byte {
	return _RequestCacheRawConfig
}

func (m *RequestCache) GetModelName() string {
	return "RequestCache"
}

func (m *RequestCache) GetCreated() time.Time {
	return m.Created
}
