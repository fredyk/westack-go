package cliutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"strings"

	"github.com/fredyk/westack-go/v2/model"
	"github.com/spf13/viper"
	"github.com/tyler-sommer/stick"
)

var defaultModelsPath = "common/models"

func generate() error {

	config := viper.New()

	config.SetConfigName("config")

	config.SetConfigType("json")

	config.AddConfigPath("server")
	config.AddConfigPath(".")

	err := config.ReadInConfig()
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	var allModelPaths []string
	if _, err := os.Stat(defaultModelsPath); !os.IsNotExist(err) {
		allModelPaths = append(allModelPaths, defaultModelsPath)
	}

	additionalIncluded := config.GetStringSlice("models.include")
	allModelPaths = append(allModelPaths, additionalIncluded...)

	if len(allModelPaths) == 0 {
		return fmt.Errorf("no models to generate")
	}

	var goRegisterFilePath string
	for idx, modelsPath := range allModelPaths {

		modelsPath := strings.TrimSuffix(modelsPath, "/")

		fmt.Printf("Generating go files from .json files under %s\n", modelsPath)

		if idx == 0 {
			goRegisterFilePath = fmt.Sprintf("%s/registercontrollers.wst.go", modelsPath)
		}

		entries, err := os.ReadDir(modelsPath)
		if err != nil {
			log.Fatalln(err)
		}

		var configs []model.Config
		for _, f := range entries {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
				config, err := generateModelForFile(modelsPath, f.Name())
				if err != nil {
					return err
				}
				configs = append(configs, config)
			}
		}

		stickEnv := createGlobalStickContext(modelsPath, configs)
		data := convertToMap(modelsPath, model.Config{}, configs, goRegisterFilePath, "")
		file, err := os.OpenFile(goRegisterFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		err = executeStickForGo(RegisterFileTemplate, stickEnv, file, data)
		if err != nil {
			return fmt.Errorf("error generating model for %s: %w", goRegisterFilePath, err)
		}
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

func generateModelForFile(modelsPath, jsonFileName string) (model.Config, error) {
	var modelConfig model.Config
	bytes, err := os.ReadFile(fmt.Sprintf("%s/%s", modelsPath, jsonFileName))
	if err != nil {
		return modelConfig, err
	}

	err = json.Unmarshal(bytes, &modelConfig)
	if err != nil {
		return modelConfig, err
	}

	modelName := modelConfig.Name
	targetGoBaseName := strings.ToLower(modelName) + ".go"
	goInterfaceFilePath := fmt.Sprintf("%s/%s", modelsPath, targetGoBaseName)
	goImplementationFilePath := fmt.Sprintf("%s/%s.wst.go", modelsPath, strings.ToLower(modelName))

	stickEnv, data := createSingleModelStickContext(modelsPath, modelConfig, goInterfaceFilePath, jsonFileName)

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

func createSingleModelStickContext(modelsPath string, config model.Config, goFilePath string, jsonFileName string) (*stick.Env, map[string]stick.Value) {
	stickEnv := stick.New(nil)

	addWestackProperties(&config)

	data := convertToMap(modelsPath, config, nil, goFilePath, jsonFileName)

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

func createGlobalStickContext(modelsPath string, configs []model.Config) *stick.Env {
	stickEnv := stick.New(nil)
	data := convertToMap(modelsPath, model.Config{}, configs, "", "")

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

func convertToMap(modelsPath string, config model.Config, configs []model.Config, path string, jsonFileName string) map[string]stick.Value {
	//bytes, _ := json.Marshal(config)
	//var parsedConfig wst.M
	//json.Unmarshal(bytes, &parsedConfig)
	return map[string]stick.Value{
		"config":       config,
		"configs":      configs,
		"path":         path,
		"jsonFileName": jsonFileName,
		"jsonFilePath": fmt.Sprintf("%s/%s", modelsPath, jsonFileName),
	}
}
