package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
	"time"
)

type PublicAccount struct {
	Created  time.Time `json:"created,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
	Email    string    `json:"email,omitempty"`
	Phone    string    `json:"phone,omitempty"`
	Id       string    `json:"id,omitempty"`
}

func NewPublicAccount() model.Controller {
	return &PublicAccount{}
}
