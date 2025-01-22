package client

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/goccy/go-json"
)

type Model interface {
	FindMany(filter *wst.Filter) ([]wst.M, error)
	FindById(id string, filter *wst.Filter) (wst.M, error)
	//FindOne(filter *wst.Filter) (*wst.M, error)
	Create(data wst.M) (wst.M, error)
	//UpdateById(id string, data wst.M) (*wst.M, error)
	//DeleteById(id string) error
}

type modelImpl struct {
	name   string
	client Client
	plural string
}

func (m *modelImpl) FindMany(filter *wst.Filter) ([]wst.M, error) {
	fullUrl := fmt.Sprintf("/%v", m.plural)
	if filter != nil {
		asMap, err := convertToMap(filter)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(asMap)
		if err != nil {
			return nil, err
		}
		fullUrl = fmt.Sprintf("%v?filter=%v", fullUrl, string(b))
	}
	return wstfuncs.InvokeApiJsonA("GET", fullUrl, nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", m.client.GetToken()),
	})
}

func (m *modelImpl) FindById(id string, filter *wst.Filter) (wst.M, error) {
	fullUrl := fmt.Sprintf("/%v/%v", m.plural, id)
	if filter != nil {
		asMap, err := convertToMap(filter)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(asMap)
		if err != nil {
			return nil, err
		}
		fullUrl = fmt.Sprintf("%v?filter=%v", fullUrl, string(b))
	}
	return wstfuncs.InvokeApiJsonM("GET", fullUrl, nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", m.client.GetToken()),
	})
}

func (m *modelImpl) Create(data wst.M) (wst.M, error) {
	headers := wst.M{
		"Content-Type": "application/json",
	}
	if m.client.GetToken() != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", m.client.GetToken())
	}
	return wstfuncs.InvokeApiJsonM("POST", fmt.Sprintf("/%v", m.plural), data, headers)
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
