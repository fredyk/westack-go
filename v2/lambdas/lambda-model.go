package lambdas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/datasource"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
)

var (
	_ model.Model = New(model.Config{})
)

type lambdaRemoteModel struct {
	apiUrl  string
	baseUrl string
	config  model.Config
}

func (rtModel *lambdaRemoteModel) FindMany(filterMap *wst.Filter, currentContext *model.EventContext) model.Cursor {

	fullUrl := rtModel.baseUrl
	if filterMap != nil {
		filterSt, err := marshalFilter(filterMap)
		if err != nil {
			return model.NewErrorCursor(err)
		}
		fullUrl = fmt.Sprintf("%s?filter=%s", fullUrl, filterSt)
	}

	c := make(chan model.Instance)
	result := model.NewChannelCursor(c)

	go func() {
		defer close(c)
		req, err := http.NewRequest("GET", fullUrl, nil)
		if err != nil {
			result.(*model.ChannelCursor).Err = err
			return
		}

		appendBearer(currentContext, req)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			result.(*model.ChannelCursor).Err = err
			return
		}

		if resp.StatusCode != http.StatusOK {
			result.(*model.ChannelCursor).Err = wst.CreateError(fiber.NewError(resp.StatusCode, resp.Status), "ERR_HTTP_STATUS_CODE", fiber.Map{"message": fmt.Sprintf("HTTP status code %d", resp.StatusCode)}, "Error")
			return
		}

		var instances []model.Instance
		err = json.NewDecoder(resp.Body).Decode(&instances)
		if err != nil {
			result.(*model.ChannelCursor).Err = err
			return
		}

		for _, instance := range instances {
			c <- instance
		}

	}()

	return result

}

func appendBearer(currentContext *model.EventContext, req *http.Request) {
	if currentContext != nil && currentContext.Bearer != nil && currentContext.Bearer.Raw != "" {
		req.Header.Set("Authorization", "Bearer "+currentContext.Bearer.Raw)
	}
}

func marshalFilter(filterMap *wst.Filter) (string, error) {
	filterBytes, err := json.Marshal(filterMap)
	if err != nil {
		return "", err
	}
	filterSt := string(filterBytes)
	return filterSt, nil
}

func (rtModel *lambdaRemoteModel) Create(data interface{}, currentContext *model.EventContext) (model.Instance, error) {

	fullUrl := rtModel.baseUrl
	var finalData wst.M

	if data != nil {
		if v, ok := data.(wst.M); ok {
			finalData = v
		} else if v, ok := data.(map[string]interface{}); ok {
			finalData = wst.M(v)
		} else if v, ok := data.(*wst.M); ok {
			finalData = *v
		} else {
			return nil, fmt.Errorf("unsupported data type: %T", data)
		}
	}

	var dataBytes []byte
	var err error
	if finalData != nil {
		dataBytes, err = easyjson.Marshal(&finalData)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", fullUrl, bytes.NewReader(dataBytes))
	if err != nil {
		return nil, err
	}

	appendBearer(currentContext, req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, wst.CreateError(fiber.NewError(resp.StatusCode, resp.Status), "ERR_HTTP_STATUS_CODE", fiber.Map{"message": fmt.Sprintf("HTTP status code %d", resp.StatusCode)}, "Error")
	}

	var plainDoc wst.M
	err = json.NewDecoder(resp.Body).Decode(&plainDoc)
	if err != nil {
		return nil, err
	}

	return buildInstance(plainDoc, rtModel.config), nil

}

func buildInstance(plainDoc wst.M, config model.Config) model.Instance {
	return &lambdaRemoteInstance{
		data: &plainDoc,
	}
}

func (rtModel *lambdaRemoteModel) FindById(id interface{}, filterMap *wst.Filter, baseContext *model.EventContext) (model.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) Count(filterMap *wst.Filter, currentContext *model.EventContext) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) DeleteById(id interface{}, currentContext *model.EventContext) (datasource.DeleteResult, error) {
	return datasource.DeleteResult{}, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) UpdateById(id interface{}, data interface{}, currentContext *model.EventContext) (model.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) GetConfig() *model.Config {
	return &rtModel.config
}

func (rtModel *lambdaRemoteModel) GetName() string {
	return rtModel.config.Name
}

func New(config model.Config) model.Model {

	apiUrl := os.Getenv("WST_API_URL")
	var plural string
	var modelBaseUrl string
	// convert ModelSpicy to model-spicies

	plural = regexp.MustCompile("([a-z])([A-Z])").ReplaceAllString(config.Name, "${1}-${2}")
	plural = strings.ToLower(plural)
	plural = regexp.MustCompile("y$").ReplaceAllString(plural, "ie") + "s"

	if config.Plural == "" {
		config.Plural = plural
	}

	modelBaseUrl = fmt.Sprintf("%s/%s", apiUrl, plural)

	return &lambdaRemoteModel{
		apiUrl:  apiUrl,
		baseUrl: modelBaseUrl,
		config:  config,
	}
}
