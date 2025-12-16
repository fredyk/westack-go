package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v3/model"
	"time"
)

type Header struct {
	Id       string    `json:"id,omitempty"`
	Created  time.Time `json:"created,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

func NewHeader() model.Controller {
	return &Header{}
}
