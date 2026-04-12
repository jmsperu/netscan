package main

import (
	"github.com/jmsperu/netscan/cmd"
)

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
