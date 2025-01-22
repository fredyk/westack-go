package client

import (
	"fmt"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
)

type Client interface {
	GetToken() string
	SetToken(token string)
	GetBaseUrl() string
	SetBaseUrl(url string)
	GetAccountsEndpoint() string
	SetAccountsEndpoint(endpoint string)
	Model(config model.Config) Model
	Login(passwordsCredentials wst.M) (wst.M, error)
}

type clientImpl struct {
	token            string
	baseUrl          string
	accountsEndpoint string
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

func (c *clientImpl) GetAccountsEndpoint() string {
	return c.accountsEndpoint
}

func (c *clientImpl) SetAccountsEndpoint(endpoint string) {
	c.accountsEndpoint = endpoint
}

func (c *clientImpl) Model(config model.Config) Model {
	return NewModel(config, c)
}

func (c *clientImpl) Login(passwordsCredentials wst.M) (wst.M, error) {
	return wstfuncs.InvokeApiJsonM("POST", fmt.Sprintf("%s/login", c.GetAccountsEndpoint()), passwordsCredentials, wst.M{
		"Content-Type": "application/json",
	})
}

type ClientOptions struct {
	Token            string
	BaseUrl          string
	AccountsEndpoint string
}

func NewClient(options ClientOptions) Client {
	impl := &clientImpl{}
	impl.SetToken(options.Token)
	impl.SetBaseUrl(options.BaseUrl)
	impl.SetAccountsEndpoint(options.AccountsEndpoint)
	wstfuncs.SetBaseUrl(options.BaseUrl)
	return impl
}
