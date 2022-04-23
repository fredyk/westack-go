package main

import (
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/westack/model"
	"log"
	"os"
	"regexp"
)

var DefaulUser = model.Config{
	Name:   "user",
	Plural: "users",
	Base:   "User",
	Public: true,
	Properties: map[string]model.Property{
		"email": {
			Type:     "string",
			Required: true,
		},
		"password": {
			Type:     "string",
			Required: true,
		},
	},
	Relations: &map[string]*model.Relation{},
	Hidden:    []string{"password"},
}

var DefaulRole = model.Config{
	Name:   "role",
	Plural: "roles",
	Base:   "Role",
	Public: true,
	Properties: map[string]model.Property{
		"name": {
			Type:     "string",
			Required: true,
		},
	},
	Relations: &map[string]*model.Relation{},
}

var DefaultDatasources = map[string]model.DataSourceConfig{
	"db": {
		Name:      "db",
		Connector: "mongodb",
		Host:      "localhost",
		Port:      27017,
		Database:  "example_db",
		User:      "",
		Password:  "",
	},
}

func initProject(cwd string) error {
	err := os.Chdir(cwd)
	if err != nil {
		return err
	}

	fqnCwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cwdName := regexp.MustCompile("([^\\\\/]+)$").FindString(fqnCwd)
	projectName := regexp.MustCompile("[^a-zA-Z0-9]+").ReplaceAllString(cwdName, "-")
	log.Println("Initializing project", projectName)

	if _, err := os.Stat("server"); os.IsNotExist(err) {
		err = os.Mkdir("server", 0755)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("server/datasources.json"); os.IsNotExist(err) {
		config := DefaultDatasources["db"]
		dbName := regexp.MustCompile("[^a-zA-Z0-9]+").ReplaceAllString(cwdName, "_")
		if dbName == "" {
			dbName = "example_db"
		}
		config.Database = dbName
		DefaultDatasources["db"] = config
		bytes, err := json.MarshalIndent(DefaultDatasources, "", "  ")
		err = os.WriteFile("server/datasources.json", bytes, 0644)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("server/model-config.json"); os.IsNotExist(err) {
		err = os.WriteFile("server/model-config.json", []byte("{}"), 0644)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("common/models"); os.IsNotExist(err) {

		if _, err := os.Stat("common"); os.IsNotExist(err) {
			err = os.Mkdir("common", 0755)
			if err != nil {
				return err
			}
		}

		err = os.Mkdir("common/models", 0755)
		if err != nil {
			return err
		}

	}

	entries, err := os.ReadDir("common/models")
	if err != nil {
		return err
	}

	foundUserModel := false
	foundRoleModel := false
	for _, entry := range entries {
		if regexp.MustCompile("\\.json$").MatchString(entry.Name()) {
			bytes, err := os.ReadFile("common/models/" + entry.Name())
			if err != nil {
				return err
			}
			var config *model.Config
			err = json.Unmarshal(bytes, &config)
			if err != nil {
				return err
			}

			if config.Base == "User" {
				foundUserModel = true
			} else if config.Base == "Role" {
				foundRoleModel = true
			}
		}
	}

	if !foundUserModel {
		err2 := addModel(DefaulUser, "db")
		if err2 != nil {
			return err2
		}
	}

	if !foundRoleModel {
		err2 := addModel(DefaulRole, "db")
		if err2 != nil {
			return err2
		}
	}

	return nil
}

func addModel(config model.Config, datasource string) error {

	path := fmt.Sprintf("common/models/%v.json", config.Name)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("model %v already exists", config.Name)
	}

	log.Printf("Adding model %v attached to datasource %v\n", config.Name, datasource)

	bytes, err := json.MarshalIndent(config, "", "  ")
	err = os.WriteFile(path, bytes, 0644)
	if err != nil {
		return err
	}

	var globalConfig map[string]model.SimplifiedConfig

	bytes, err = os.ReadFile("server/model-config.json")
	err = json.Unmarshal(bytes, &globalConfig)
	if err != nil {
		return err
	}

	globalConfig[config.Name] = model.SimplifiedConfig{
		Datasource: datasource,
	}

	bytes, err = json.MarshalIndent(globalConfig, "", "  ")
	err = os.WriteFile("server/model-config.json", bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
