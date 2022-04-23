package main

import "log"

func printHelp() {
	log.Println("Usage: cli <command>")
	log.Println("\tinit <target> \tInitializes a new project in the <target> directory")
	log.Println("\tmodel add <model name> <datasource> \tCreates a new model with the given <model name> and attaches it to <datasource>")
	log.Println()
}
