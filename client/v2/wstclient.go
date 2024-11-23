package client

import (
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	"github.com/fredyk/westack-go/v2/model"
)

type Client interface {
	GetToken() string
	SetToken(token string)
	GetBaseUrl() string
	SetBaseUrl(url string)
	Model(config model.Config) Model
}

type clientImpl struct {
	token   string
	baseUrl string
}

func (c *clientImpl) GetToken() string {
	return c.token
}

func (c *clientImpl) SetToken(token string) {
	c.token = token
}

func (c *clientImpl) GetBaseUrl() string {
	return c.baseUrl
}

func (c *clientImpl) SetBaseUrl(url string) {
	c.baseUrl = url
}

func (c *clientImpl) Model(config model.Config) Model {
	return NewModel(config, c)
}

type ClientOptions struct {
	Token   string
	BaseUrl string
}

func NewClient(options ClientOptions) Client {
	impl := &clientImpl{}
	impl.SetToken(options.Token)
	impl.SetBaseUrl(options.BaseUrl)
	wstfuncs.SetBaseUrl(options.BaseUrl)
	return impl
}
