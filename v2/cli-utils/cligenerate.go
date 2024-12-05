package cliutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/tyler-sommer/stick"
	"go/format"
	"io"
	"log"
	"os"
	"strings"
)

func generate() error {
	if _, err := os.Stat("common/models"); os.IsNotExist(err) {
		return fmt.Errorf("not in a westack project")
	}

	log.Println("Generating go files from .json files under common/models")

	entries, err := os.ReadDir("common/models")
	if err != nil {
		log.Fatalln(err)
	}

	var configs []model.Config
	for _, f := range entries {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			config, err := generateModelForFile(f.Name())
			if err != nil {
				return err
			}
			configs = append(configs, config)
		}
	}

	goRegisterFilePath := "common/models/registercontrollers.wst.go"

	stickEnv := createGlobalStickContext(configs)
	data := convertToMap(model.Config{}, configs, goRegisterFilePath, "")
	file, err := os.OpenFile(goRegisterFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	err = executeStickForGo(RegisterFileTemplate, stickEnv, file, data)
	if err != nil {
		return fmt.Errorf("error generating model for %s: %w", goRegisterFilePath, err)
	}

	return nil
}

func executeStickForGo(template string, stickEnv *stick.Env, file *os.File, data map[string]stick.Value) error {
	b := make([]byte, 0)
	buffer := bytes.NewBuffer(b)
	err := stickEnv.Execute(template, buffer, data)
	if err != nil {
		return err
	}
	// Now format go code
	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return err
	}
	_, err = io.Copy(file, bytes.NewReader(formatted))
	if err != nil {
		return err
	}
	return nil
}

func generateModelForFile(jsonFileName string) (model.Config, error) {
	var modelConfig model.Config
	bytes, err := os.ReadFile("common/models/" + jsonFileName)
	if err != nil {
		return modelConfig, err
	}

	err = json.Unmarshal(bytes, &modelConfig)
	if err != nil {
		return modelConfig, err
	}

	modelName := modelConfig.Name
	targetGoBaseName := strings.ToLower(modelName) + ".go"
	goInterfaceFilePath := "common/models/" + targetGoBaseName
	goImplementationFilePath := fmt.Sprintf("common/models/%s.wst.go", strings.ToLower(modelName))

	stickEnv, data := createSingleModelStickContext(modelConfig, goInterfaceFilePath, jsonFileName)

	if _, err := os.Stat(goInterfaceFilePath); os.IsNotExist(err) {
		err := generateModelForConfig(StructFileTemplate, goInterfaceFilePath, stickEnv, data, modelConfig, jsonFileName)
		if err != nil {
			return modelConfig, err
		}
	}

	err = generateModelForConfig(ImplementationFileTemplate, goImplementationFilePath, stickEnv, data, modelConfig, jsonFileName)
	if err != nil {
		return modelConfig, err
	}

	return modelConfig, nil

}

func generateModelForConfig(template string, goFilePath string, stickEnv *stick.Env, data map[string]stick.Value, config model.Config, jsonFileName string) error {

	file, err := os.OpenFile(goFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	defer file.Close()
	err = executeStickForGo(template, stickEnv, file, data)
	if err != nil {
		return fmt.Errorf("error generating model for %s: %w", goFilePath, err)
	}

	return nil
}

func createSingleModelStickContext(config model.Config, goFilePath string, jsonFileName string) (*stick.Env, map[string]stick.Value) {
	stickEnv := stick.New(nil)

	addWestackProperties(&config)

	data := convertToMap(config, nil, goFilePath, jsonFileName)

	addStickFunctions(config, nil, stickEnv, data)

	return stickEnv, data
}

func addWestackProperties(c *model.Config) {
	allProperties := make(map[string]model.Property)
	allProperties["id"] = model.Property{
		Type: "string",
	}
	allProperties["created"] = model.Property{
		Type: "date",
	}
	allProperties["modified"] = model.Property{
		Type: "date",
	}
	for k, v := range c.Properties {
		allProperties[k] = v
	}
	c.Properties = allProperties
}

func createGlobalStickContext(configs []model.Config) *stick.Env {
	stickEnv := stick.New(nil)
	data := convertToMap(model.Config{}, configs, "", "")

	addStickFunctions(model.Config{}, configs, stickEnv, data)

	return stickEnv
}

func processNeededImports(config model.Config) []string {
	neededAsMap := make(map[string]bool)
	for _, prop := range config.Properties {
		if prop.Type == "date" {
			neededAsMap["time"] = true
		}
	}

	var neededImports []string
	for k := range neededAsMap {
		neededImports = append(neededImports, fmt.Sprintf("\"%s\"", k))
	}
	return neededImports
}

func convertToMap(config model.Config, configs []model.Config, path string, jsonFileName string) map[string]stick.Value {
	//bytes, _ := json.Marshal(config)
	//var parsedConfig wst.M
	//json.Unmarshal(bytes, &parsedConfig)
	return map[string]stick.Value{
		"config":       config,
		"configs":      configs,
		"path":         path,
		"jsonFileName": jsonFileName,
		"jsonFilePath": fmt.Sprintf("common/models/%s", jsonFileName),
	}
}
