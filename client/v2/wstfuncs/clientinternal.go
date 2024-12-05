package wstfuncs

import (
	"bytes"
	"encoding/json"
	"fmt"
	wst "github.com/fredyk/westack-go/v2/common"
	"io"
	"net/http"
	"strings"
	"time"
)

var baseUrl string

func SetBaseUrl(url string) {
	baseUrl = url
}

func InvokeApiJsonM(method string, url string, body wst.M, headers wst.M) (result wst.M, err error) {
	result, err = InvokeApiTyped[wst.M](method, url, body, headers)
	return result, err
}

func InvokeApiJsonA(method string, url string, body wst.M, headers wst.M) (result wst.A, err error) {
	return InvokeApiTyped[wst.A](method, url, body, headers)
}

func InvokeApiFullResponse(method string, url string, body wst.M, headers wst.M) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", baseUrl, url), jsonToReaderOnlyIfNeeded(method, body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}
	//resp, err := app.Server.Test(req, 600000)
	client := &http.Client{
		Timeout: 45 * time.Second,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client.Do(req)
}

func jsonToReaderOnlyIfNeeded(method string, body wst.M) io.Reader {
	switch strings.ToLower(method) {
	case "get", "head", "delete":
		return nil
	default:
		return jsonToReader(body)
	}
}

func InvokeApiTyped[T any](method string, url string, body wst.M, headers wst.M) (result T, err error) {
	respBody, _ := invokeApiBytes(method, url, body, headers)
	var parsedRespBody T
	err = json.Unmarshal(respBody, &parsedRespBody)

	return parsedRespBody, err
}

func invokeApiBytes(method string, url string, body wst.M, headers wst.M) ([]byte, error) {
	resp, _ := InvokeApiFullResponse(method, url, body, headers)
	if resp == nil || resp.Body == nil {
		return make([]byte, 0), fmt.Errorf("nil or empty response")
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func jsonToReader(m wst.M) io.Reader {
	out, err := json.Marshal(m)
	fmt.Printf("Ignoring error %v\n", err)
	return bytes.NewReader(out)
}
