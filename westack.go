package main

import (
	cliutils "github.com/fredyk/westack-go/cli-utils"
)

//export WstMain
func WstMain() {
	cliutils.RunCli()
}

func main() {

	WstMain()

}
