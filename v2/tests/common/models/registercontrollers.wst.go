package models

import (
	_ "embed"
	"github.com/fredyk/westack-go/v2/model"
)

func RegisterControllers(r model.ControllerRegistry) {
	// iterate configs

	r.RegisterController(&Account{})
	r.RegisterController(&App{})
	r.RegisterController(&Customer{})
	r.RegisterController(&Empty{})
	r.RegisterController(&Footer{})
	r.RegisterController(&Image{})
	r.RegisterController(&Note{})
	r.RegisterController(&NoteEntry{})
	r.RegisterController(&Order{})
	r.RegisterController(&PublicAccount{})
	r.RegisterController(&RequestCache{})
	r.RegisterController(&Store{})
	r.RegisterController(&role{})
}
