package cliutils

import (
	"fmt"
	"log"
	"os"
)

func diagnosePermissions() {
	diagnoseModelsWithoutEveryoneRead()
}

func diagnoseModelsWithoutEveryoneRead() {
	// Load all models
	allModels, err := loadAllModels()
	if err != nil {
		log.Fatalf("Error loading models: %v", err)
	}
	fmt.Println()
	fmt.Println("Checking for possible 401 reasons:")
	fmt.Println()
	issuesFound := false
	// Check if some model does not have "$everyone,*,read,allow" policy
	for _, model := range allModels {
		if model.Base != "Account" && model.Base != "Role" && model.Base != "App" {
			if model.Casbin.Policies == nil {
				fmt.Printf("WARNING: Model %v does not have any policies\n", model.Name)
			} else {
				found := false
				for _, policy := range model.Casbin.Policies {
					if policy == "$authenticated,*,read,allow" {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("INFO: Model %v does not have a policy allowing read for $authenticated\n", model.Name)
					issuesFound = true
				}
			}
		}
	}

	fmt.Println()

	// Check if some model does not have a relation to Account model
	for _, model := range allModels {
		if model.Base != "Account" && model.Base != "Role" {
			if model.Relations == nil {
				fmt.Printf("WARNING: Model %v does not have any relations\n", model.Name)
			} else {
				found := false
				for _, relation := range *model.Relations {
					if (relation.Model == "Account" || relation.Model == "App") && relation.Type == "belongsTo" {
						found = true
						break
					}
				}
				if !found {
					if model.Base == "App" {
						fmt.Printf("ERROR: Model %v does not have a relation to Account or App model\n", model.Name)
					} else {
						fmt.Printf("INFO: Model %v does not have a relation to Account or App model\n", model.Name)
					}
					issuesFound = true
				}
			}
		}
	}
	if !issuesFound {
		fmt.Println("No issues found")
	}
	fmt.Println()
	fmt.Println()

	fmt.Println("Checking for possible excessive permissions:")
	fmt.Println()
	issuesFound = false
	for _, model := range allModels {
		if model.Base != "Account" && model.Base != "Role" && model.Base != "App" {
			if model.Casbin.Policies != nil {
				for _, policy := range model.Casbin.Policies {
					if policy == "$everyone,*,read,allow" {
						fmt.Printf("WARNING: Model %v has a policy allowing read for $everyone\n", model.Name)
						issuesFound = true
					}
				}
			}
		}
	}
	if !issuesFound {
		fmt.Println("No issues found")
	}
}

func diagnoseLauncher() {
	// Check environment variables
	envVars := []string{"JWT_SECRET", "WST_ADMIN_USERNAME", "WST_ADMIN_PWD"}
	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			fmt.Printf("ERROR: Environment variable %v is not set in current terminal\n", envVar)
		}
	}
}
