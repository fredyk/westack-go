package lambdas

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	modelentities "github.com/fredyk/westack-go/v2/lambdas/entities/model-entities"
)

func convertHttpPathToFileLocation(basePath string, path string) string {
	// Escape path
	// Map path to static file
	// Return file location
	path = strings.ReplaceAll(path, "..", "")

	re := regexp.MustCompile(`\:([a-zA-Z0-9_-]+)`).ReplaceAllString(basePath, "([^/]+)")
	re = strings.ReplaceAll(re, "/*", "")

	re = fmt.Sprintf("^%s/?", re)
	path = regexp.MustCompile(re).ReplaceAllString(path, "./assets/dist/")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "./assets/dist/index.html"
	} else {
		// if it is a directory, return the index.html file
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			path = fmt.Sprintf("%s/index.html", path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return "./assets/dist/index.html"
			} else {
				return path
			}
		}
	}

	return path
}

func openFile(fileLocation string) (f io.ReadCloser, err error) {
	if fileLocation == "./assets/dist/" {
		fileLocation = "./assets/dist/index.html"
	}
	f, err = os.Open(fileLocation)
	return
}

func SendStaticAsset(req LambdaRequest) (f io.ReadCloser, err error) {

	path := req.Path
	basePath := req.BasePath

	if path != "" {
		path, err = url.QueryUnescape(path)
		if err != nil {
			return nil, fmt.Errorf("failed to decode URL: %w", err)
		}
	}

	fileLocation := convertHttpPathToFileLocation(basePath, path)
	if fileLocation != "" {
		f, err = openFile(fileLocation)
	} else {
		f = io.NopCloser(bytes.NewReader([]byte("<html><body>No file found</body></html>")))
		err = modelentities.ErrNotFound
	}
	return
}

func CleanContentType(contentType string) string {
	return strings.TrimSpace(strings.Split(contentType, ";")[0])
}
