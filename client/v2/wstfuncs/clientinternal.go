package wstfuncs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	wst "github.com/fredyk/westack-go/v2/common"
	"golang.org/x/net/publicsuffix"
)

var baseUrl string

type RequestOptions struct {
	FollowRedirects bool
}

func SetBaseUrl(url string) {
	baseUrl = url
}

func GetBaseUrl() string {
	return baseUrl
}

func InvokeApiJsonM(method string, url string, body wst.M, headers wst.M) (wst.M, error) {
	return InvokeApiTyped[wst.M](method, url, body, headers)
}

func InvokeApiJsonA(method string, url string, body wst.M, headers wst.M) (wst.A, error) {
	return InvokeApiTyped[wst.A](method, url, body, headers)
}

func InvokeApiFullResponse(method string, url string, body wst.M, headers wst.M, options ...RequestOptions) (*http.Response, error) {
	bodyReader, err := jsonToReaderOnlyIfNeeded(method, body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", baseUrl, url), bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}

	// All users of cookiejar should import "golang.org/x/net/publicsuffix"
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	//resp, err := app.Server.Test(req, 600000)
	client := &http.Client{
		Timeout: 45 * time.Second,
		Jar:     jar,
	}

	followRedirects := false
	if len(options) > 0 {
		followRedirects = options[0].FollowRedirects
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !followRedirects {
			return http.ErrUseLastResponse
		} else {
			return nil
		}
	}

	return client.Do(req)
}

func jsonToReaderOnlyIfNeeded(method string, body wst.M) (io.Reader, error) {
	switch strings.ToLower(method) {
	case "get", "head", "delete":
		return nil, nil
	default:
		return jsonToReader(body)
	}
}

func InvokeApiTyped[T any](method string, url string, body wst.M, headers wst.M) (T, error) {
	respBody, err := invokeApiBytes(method, url, body, headers)
	var parsedRespBody T
	if err != nil {
		return parsedRespBody, err
	}
	err = json.Unmarshal(respBody, &parsedRespBody)

	return parsedRespBody, err
}

func invokeApiBytes(method string, url string, body wst.M, headers wst.M) ([]byte, error) {
	resp, err := InvokeApiFullResponse(method, url, body, headers)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("nil or empty response")
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func jsonToReader(m wst.M) (io.Reader, error) {
	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(out), nil
}
