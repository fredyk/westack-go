package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

type Note struct {
	Id                     string            `json:"id,omitempty"`
	Created                time.Time         `json:"created,omitempty"`
	DefaultTimeNow         time.Time         `json:"defaultTimeNow,omitempty"`
	DefaultBoolean         bool              `json:"defaultBoolean,omitempty"`
	DefaultList            []string          `json:"defaultList,omitempty"`
	DefaultMap             map[string]string `json:"defaultMap,omitempty"`
	DefaultNull            string            `json:"defaultNull,omitempty"`
	Modified               time.Time         `json:"modified,omitempty"`
	DefaultTimeHourFromNow time.Time         `json:"defaultTimeHourFromNow,omitempty"`
	DefaultString          string            `json:"defaultString,omitempty"`
	DefaultInt             int               `json:"defaultInt,omitempty"`
	DefaultFloat           int               `json:"defaultFloat,omitempty"`
	DefaultTimeHourAgo     time.Time         `json:"defaultTimeHourAgo,omitempty"`
}

func NewNote() model.Controller {
	return &Note{}
}
