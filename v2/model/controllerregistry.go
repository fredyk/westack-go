package model

type Controller interface {
	GetModelName() string
}

type ControllerRegistry interface {
	RegisterController(controller Controller)
}
