package client

import (
	"fmt"
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"regexp"
	"strings"
)

type Model interface {
	FindMany(filter *wst.Filter) ([]wst.M, error)
	FindById(id string, filter *wst.Filter) (wst.M, error)
	//FindOne(filter *wst.Filter) (*wst.M, error)
	//Create(data wst.M) (*wst.M, error)
	//UpdateById(id string, data wst.M) (*wst.M, error)
	//DeleteById(id string) error
}

type modelImpl struct {
	name   string
	client Client
	plural string
}

func (m *modelImpl) FindMany(filter *wst.Filter) ([]wst.M, error) {
	asMap, err := convertToMap(filter)
	if err != nil {
		return nil, err
	}
	return wstfuncs.InvokeApiJsonA("GET", fmt.Sprintf("/%v", m.plural), asMap, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", m.client.GetToken()),
	})
}

func (m *modelImpl) FindById(id string, filter *wst.Filter) (wst.M, error) {
	asMap, err := convertToMap(filter)
	if err != nil {
		return nil, err
	}
	return wstfuncs.InvokeApiJsonM("GET", fmt.Sprintf("/%v/%v", m.plural, id), asMap, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", m.client.GetToken()),
	})
}

func NewModel(config model.Config, client Client) Model {

	plural := config.Plural

	if config.Plural == "" {
		plural = regexp.MustCompile("([a-z])([A-Z])").ReplaceAllString(config.Name, "${1}-${2}")
		plural = strings.ToLower(plural)
		plural = regexp.MustCompile("y$").ReplaceAllString(plural, "ie") + "s"
	}
	return &modelImpl{
		name:   config.Name,
		client: client,
		plural: plural,
	}
}
