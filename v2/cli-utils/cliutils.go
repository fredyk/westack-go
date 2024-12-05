package cliutils

import (
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/v2/westack"
	"log"
	"os"
)

func RunCli() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "init":

		if len(os.Args) < 3 {
			printHelp()
			return
		}

		err := initProject(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}
	case "model":
		if len(os.Args) < 3 {
			printHelp()
			return
		}

		switch os.Args[2] {
		case "add":
			if len(os.Args) < 5 {
				printHelp()
				return
			}
			err := addModel(model.Config{
				Name:       os.Args[3],
				Plural:     "",
				Base:       "PersistedModel",
				Public:     true,
				Properties: map[string]model.Property{},
				Relations:  &map[string]*model.Relation{},
				Hidden:     []string{},
				Casbin:     model.CasbinConfig{},
			}, os.Args[4])
			if err != nil {
				log.Fatalln(err)
			}
		}

	case "server":
		if len(os.Args) < 3 {
			printHelp()
			return
		} else {
			switch os.Args[2] {
			case "start":
				westack.InitAndServe(westack.Options{})
			default:
				printHelp()
			}
		}
	case "diagnose":
		if len(os.Args) < 3 {
			printHelp()
			return
		} else {
			switch os.Args[2] {
			case "permissions":
				diagnosePermissions()
			case "launcher":
				diagnoseLauncher()
			default:
				printHelp()
			}
		}
	case "generate":
		err := generate()
		if err != nil {
			log.Fatalf("Error generating go files: %v", err)
		}
	case "help":
		printHelp()
	default:
		printHelp()
	}
}
