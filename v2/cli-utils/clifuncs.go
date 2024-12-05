package cliutils

import "log"

func printHelp() {
	log.Println("Usage: cli <command>")
	log.Println("\tinit <target> \tInitializes a new project in the <target> directory")
	log.Println("\tmodel add <model name> <datasource> \tCreates a new model with the given <model name> and attaches it to <datasource>")
	log.Println("\tserver start \tStarts the server")
	log.Println("\tdiagnose [permissions|launcher] \tRuns a diagnostic check on the server for debugging purposes")
	log.Println("\tgenerate \tGenerates all go files from .json files under common/models")
	log.Println()
}
