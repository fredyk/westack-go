package main

import (
	cliutils "github.com/fredyk/westack-go/cli-utils"
)

func main() {

	// Deprecated. Now should run as 'westack server start'
	//westack.InitAndServe()

	cliutils.RunCli()

}
