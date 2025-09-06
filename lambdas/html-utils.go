package lambdas

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	modelentities "github.com/fredyk/westack-go/lambdas/entities/model-entities"
)

// ConvertHttpPathToFileLocation converts the HTTP path to a file location
// It takes the base path and the HTTP path as input
// and returns the file location
// It also checks if the file exists and returns a boolean value indicating
// whether the file exists or not
// If the file does not exist, it returns the default index.html file
// If the path is a directory, it returns the index.html file in that directory
// If the path is empty, it returns the default index.html file
func convertHttpPathToFileLocation(basePath string, path string) (bool, string) {
	// Escape path
	// Map path to static file
	// Return file location
	path = strings.ReplaceAll(path, "..", "")

	re := regexp.MustCompile(`\:([a-zA-Z0-9_-]+)`).ReplaceAllString(basePath, "([^/]+)")
	re = strings.ReplaceAll(re, "/*", "")

	re = fmt.Sprintf("^%s/?", re)
	path = regexp.MustCompile(re).ReplaceAllString(path, "./assets/dist/")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, "./assets/dist/index.html"
	} else {
		// if it is a directory, return the index.html file
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			path = fmt.Sprintf("%s/index.html", path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return false, "./assets/dist/index.html"
			} else {
				return true, path
			}
		} else {
			return true, path
		}
	}

}

func openFile(fileLocation string) (f io.ReadCloser, err error) {
	if fileLocation == "./assets/dist/" {
		fileLocation = "./assets/dist/index.html"
	}
	f, err = os.Open(fileLocation)
	return
}

func SendStaticAsset(w http.ResponseWriter, r *http.Request, req LambdaRequest) (f io.ReadCloser, err error) {

	path := req.Path
	basePath := req.BasePath

	if path != "" {
		path, err = url.QueryUnescape(path)
		if err != nil {
			return nil, fmt.Errorf("failed to decode URL: %w", err)
		}
	}

	exists, fileLocation := convertHttpPathToFileLocation(basePath, path)
	if !exists {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		err = modelentities.ErrNotFound
	}
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
