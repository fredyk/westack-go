package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

type Customer struct {
	Created  time.Time `json:"created,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
	Id       string    `json:"id,omitempty"`
}

func NewCustomer() model.Controller {
	return &Customer{}
}
