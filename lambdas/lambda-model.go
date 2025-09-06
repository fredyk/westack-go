package lambdas

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/goccy/go-json"

	"github.com/fredyk/westack-go/lambdas/entities"
	modelentities "github.com/fredyk/westack-go/lambdas/entities/model-entities"
)

var (
	_ modelentities.Model = New(modelentities.Config{})
)

type lambdaRemoteModel struct {
	apiUrl  string
	baseUrl string
	config  modelentities.Config
}

func (rtModel *lambdaRemoteModel) FindMany(filterMap *entities.Filter, currentContext *modelentities.EventContext) modelentities.Cursor {

	fullUrl := rtModel.baseUrl
	if filterMap != nil {
		filterSt, err := marshalFilter(filterMap)
		if err != nil {
			return modelentities.NewErrorCursor(err)
		}
		fullUrl = fmt.Sprintf("%s?filter=%s", fullUrl, filterSt)
	}

	c := make(chan modelentities.Instance)
	result := modelentities.NewChannelCursor(c)

	go func() {
		defer close(c)
		req, err := http.NewRequest("GET", fullUrl, nil)
		if err != nil {
			result.(*modelentities.ChannelCursor).Err = err
			return
		}

		appendBearer(currentContext, req)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			result.(*modelentities.ChannelCursor).Err = err
			return
		}

		if resp.StatusCode != http.StatusOK {
			result.(*modelentities.ChannelCursor).Err = modelentities.CreateError(modelentities.NewError(resp.StatusCode, resp.Status), "ERR_HTTP_STATUS_CODE", entities.M{"message": fmt.Sprintf("HTTP status code %d", resp.StatusCode)}, "Error")
			return
		}

		var instances []modelentities.Instance
		err = json.NewDecoder(resp.Body).Decode(&instances)
		if err != nil {
			result.(*modelentities.ChannelCursor).Err = err
			return
		}

		for _, instance := range instances {
			c <- instance
		}

	}()

	return result

}

func appendBearer(currentContext *modelentities.EventContext, req *http.Request) {
	if currentContext != nil && currentContext.Bearer != nil && currentContext.Bearer.Raw != "" {
		req.Header.Set("Authorization", "Bearer "+currentContext.Bearer.Raw)
	}
}

func marshalFilter(filterMap *entities.Filter) (string, error) {
	filterBytes, err := json.Marshal(filterMap)
	if err != nil {
		return "", err
	}
	filterSt := string(filterBytes)
	return filterSt, nil
}

func (rtModel *lambdaRemoteModel) Create(data interface{}, currentContext *modelentities.EventContext) (modelentities.Instance, error) {

	fullUrl := rtModel.baseUrl
	var finalData entities.M

	if data != nil {
		if v, ok := data.(entities.M); ok {
			finalData = v
		} else if v, ok := data.(map[string]interface{}); ok {
			finalData = entities.M(v)
		} else if v, ok := data.(*entities.M); ok {
			finalData = *v
		} else {
			return nil, fmt.Errorf("unsupported data type: %T", data)
		}
	}

	var dataBytes []byte
	var err error
	if finalData != nil {
		dataBytes, err = json.Marshal(&finalData)
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
		return nil, modelentities.CreateError(modelentities.NewError(resp.StatusCode, resp.Status), "ERR_HTTP_STATUS_CODE", entities.M{"message": fmt.Sprintf("HTTP status code %d", resp.StatusCode)}, "Error")
	}

	var plainDoc entities.M
	err = json.NewDecoder(resp.Body).Decode(&plainDoc)
	if err != nil {
		return nil, err
	}

	return buildInstance(plainDoc, rtModel.config), nil

}

func buildInstance(plainDoc entities.M, config modelentities.Config) modelentities.Instance {
	return &lambdaRemoteInstance{
		data: &plainDoc,
	}
}

func (rtModel *lambdaRemoteModel) FindById(id interface{}, filterMap *entities.Filter, baseContext *modelentities.EventContext) (modelentities.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) Count(filterMap *entities.Filter, currentContext *modelentities.EventContext) (modelentities.CountResult, error) {
	return modelentities.CountResult{}, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) DeleteById(id interface{}, currentContext *modelentities.EventContext) (modelentities.DeleteResult, error) {
	return modelentities.DeleteResult{}, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) UpdateById(id interface{}, data interface{}, currentContext *modelentities.EventContext) (modelentities.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rtModel *lambdaRemoteModel) GetConfig() *modelentities.Config {
	return &rtModel.config
}

func (rtModel *lambdaRemoteModel) GetName() string {
	return rtModel.config.Name
}

func New(config modelentities.Config) modelentities.Model {

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
