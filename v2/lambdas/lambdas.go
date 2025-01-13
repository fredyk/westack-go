package lambdas

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/mailru/easyjson"
)

type LambdaError struct {
	Message string `json:"message"`
}

type LambdaResult struct {
	StatusCode  int          `json:"statusCode"`
	ContentType string       `json:"contentType"`
	Headers     wst.M        `json:"headers"`
	Body        wst.M        `json:"body"`
	RawBody     []byte       `json:"rawBody"`
	Error       *LambdaError `json:"error"`
	Json        bool         `json:"json"`
}

func invoke(name string, method string, path string, payload wst.M) (result *LambdaResult, err error) {

	baseUrl := os.Getenv("WST_API_URL")

	if baseUrl == "" {
		return nil, errors.New("WST_API_URL environment variable not set")
	}

	url := baseUrl + path

	var reqBytes []byte
	if payload != nil {
		reqBytes, err = easyjson.Marshal(&payload)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	headers := wst.M{}
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	result = &LambdaResult{
		StatusCode:  resp.StatusCode,
		ContentType: wst.CleanContentType(resp.Header.Get("Content-Type")),
		Headers:     headers,
	}
	if resp.ContentLength > 0 {
		rawResponseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// for application/json, parse it and not deliver rawBody, else deliver rawBody
		if result.ContentType == "application/json" {
			err = easyjson.Unmarshal(rawResponseBody, &result.Body)
			if err != nil {
				return nil, err
			}
			result.Json = true
		} else {
			result.RawBody, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
		}
	}
	if result.StatusCode >= 400 {
		err = fmt.Errorf("lambda returned status code %d", result.StatusCode)
		result.Error = &LambdaError{Message: err.Error()}
		return result, err
	}

	return result, nil

}

var InvokeLambda = invoke
