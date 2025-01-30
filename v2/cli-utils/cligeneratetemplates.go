package cliutils

// uses twig template engine

var StructFileTemplate = `package models

{{ renderImports(true, "github.com/fredyk/westack-go/v2/model") }}
{{ renderStruct() }}

func New{{ config.Name }}() model.Controller {
	return &{{ config.Name}}{}
}

`

var StructTemplate = `

type {{ config.Name }} struct {

	{% for key, value in config.Properties %} {{ capitalize(key) }} {{ renderType(value.Type) }} ` + "`json:\"{{ key }},omitempty\"`" + `
	{% endfor %}
}

`

var ImplementationFileTemplate = `//wst:generated Don't edit this file
package models

{{ renderImports(false, "github.com/fredyk/westack-go/v2/model") }}

//go:embed {{ jsonFileName }}
var _{{ config.Name }}RawConfig []byte

func (m *{{ config.Name }}) Register(r model.ControllerRegistry) {
	r.RegisterController(m)
}

func (m *{{ config.Name }}) GetRawConfig() []byte {
	return _{{ config.Name }}RawConfig
}

func (m *{{ config.Name }}) GetModelName() string {
	return "{{ config.Name }}"
}

`

var RegisterFileTemplate = `package models

{{ renderImports(false, "github.com/fredyk/westack-go/v2/model") }}

func RegisterControllers(r model.ControllerRegistry) {
	// iterate configs
	{% for _, config in configs %}
		r.RegisterController(&{{ config.Name }}{}) {% endfor %}
}
`
