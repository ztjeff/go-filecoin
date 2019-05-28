package main

import (
	"os"

	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/cmds"
)

func main() {
	code, _ := cmds.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
