package tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/andybalholm/brotli"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_Get_Swagger_Docs(t *testing.T) {

	t.Parallel()

	// start client
	client := http.Client{}

	// test for error
	res, err := client.Get("http://localhost:8020/swagger/doc.json")
	assert.Nilf(t, err, "Get Swagger Error while getting response: %s", err)

	// read response
	var out wst.M
	body, err := io.ReadAll(res.Body)
	assert.Nilf(t, err, "Get Swagger Error while reading body: %s", err)

	assert.Equalf(t, 200, res.StatusCode, "Get Swagger Error invalid status code: %d body: %s", res.StatusCode, string(body))

	err = json.Unmarshal(body, &out)
	assert.Nilf(t, err, "Get Swagger Error while unmarshaling body: %s", err)

	assert.Equalf(t, "3.0.1", out["openapi"], "Invalid openapi version %v", out["openapi"])
	assert.Equalf(t, "Swagger API", out.GetM("info").GetString("title"), "Invalid title %v", out.GetM("info").GetString("title"))
	assert.Equalf(t, "3.0", out.GetM("info").GetString("version"), "Invalid version %v", out.GetM("info").GetString("version"))

}

func Test_Get_Swagger_UI(t *testing.T) {

	t.Parallel()

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
	body, err := io.ReadAll(brotli.NewReader(res.Body))
	if err != nil {
		t.Errorf("Get Swagger Error: %s", err)
		return
	}

	if len(body) == 0 {
		t.Errorf("Get Swagger Error: empty response")
		return
	}

	fmt.Printf("DEBUG: Swagger: got %v bytes <-- %v\n", len(body), string(body[:32]))
}

func Test_GetCorruptSwagger(t *testing.T) {

	t.Parallel()

	// Wait 5 seconds, that way we don't break the other tests
	time.Sleep(5 * time.Second)

	// write <invalid json> to data/swagger.json
	err := os.WriteFile("data/swagger.json", []byte("<invalid json>"), 0600)
	assert.NoError(t, err)

	// start client
	client := http.Client{}

	// test for error
	resp, err := client.Get("http://localhost:8020/swagger/doc.json")
	assert.NoError(t, err)
	assert.EqualValues(t, 500, resp.StatusCode)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.EqualValues(t, "{\"error\":\"parse error: syntax error near offset 0 of '\\u003cinvalid j...'\"}", string(body))

}
