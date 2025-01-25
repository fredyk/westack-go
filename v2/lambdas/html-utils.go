package lambdas

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/fredyk/westack-go/v2/model"
)

func convertHttpPathToFileLocation(basePath string, path string) string {
	// Escape path
	// Map path to static file
	// Return file location
	path = strings.ReplaceAll(path, "..", "")

	re := regexp.MustCompile(`\:([a-zA-Z0-9_-]+)`).ReplaceAllString(basePath, "([^/]+)")
	re = strings.ReplaceAll(re, "/*", "")

	re = fmt.Sprintf("^%s/?", re)
	fmt.Printf("[DEBUG] Regexp: %s\n", re)
	path = regexp.MustCompile(re).ReplaceAllString(path, "./assets/dist/")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "./assets/dist/index.html"
	}

	return path
}

func readFileBytes(fileLocation string) ([]byte, error) {
	var fileContent []byte
	if fileLocation == "./assets/dist/" {
		fileLocation = "./assets/dist/index.html"
	}
	f, err := os.Open(fileLocation)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileContent, err = io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

func SendStaticAsset(ctx *model.EventContext) error {
	basePath := ctx.Data.GetString("base_path")
	path := ctx.Data.GetString("path")
	fmt.Printf("Path: %s\n", path)
	fmt.Printf("Base path: %s\n", basePath)

	fileLocation := convertHttpPathToFileLocation(basePath, path)
	fmt.Printf("File location: %s\n", fileLocation)
	var fileContent []byte
	var err error
	if fileLocation != "" {

		fileContent, err = readFileBytes(fileLocation)
	} else {
		fileContent = []byte("<html><body>No file found</body></html>")
	}

	ctx.Result = fileContent
	return err
}
