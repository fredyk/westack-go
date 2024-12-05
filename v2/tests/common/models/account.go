package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

type Account struct {
	Created  time.Time `json:"created,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
	Email    string    `json:"email,omitempty"`
	Id       string    `json:"id,omitempty"`
}

func NewAccount() model.Controller {
	return &Account{}
}
