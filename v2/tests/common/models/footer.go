package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

type Footer struct {
	Id       string    `json:"id,omitempty"`
	Created  time.Time `json:"created,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

func NewFooter() model.Controller {
	return &Footer{}
}
