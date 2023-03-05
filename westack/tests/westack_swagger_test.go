package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_Get_Swagger_Docs(t *testing.T) {

	t.Parallel()

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Get("http://localhost:8020/swagger/doc.json")
	if err != nil {
		t.Errorf("Get Swagger Error: %s", err)
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("Get Swagger Error: %d", res.StatusCode)
		return
	}

	// read response
	var out wst.M
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Get Swagger Error: %s", err)
		return
	}
	err = json.Unmarshal(body, &out)
	if err != nil {
		t.Errorf("Get Swagger Error: %s", err)
		return
	}

	if out["openapi"] != "3.0.1" {
		t.Errorf("Invalid openapi version: %s", out["openapi"])
		return
	}

}

func Test_Get_Swagger_UI(t *testing.T) {

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Get("http://localhost:8020/swagger/")
	if err != nil {
		t.Errorf("Get Swagger Error: %s", err)
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("Get Swagger Error: %d", res.StatusCode)
		return
	}

	// read response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Get Swagger Error: %s", err)
		return
	}

	if len(body) == 0 {
		t.Errorf("Get Swagger Error: empty response")
		return
	}
}
