package main

import cliutils "github.com/fredyk/westack-go/v2/cli-utils"

//export WstMain
func WstMain() {
	cliutils.RunCli()
}

func main() {

	WstMain()

}
