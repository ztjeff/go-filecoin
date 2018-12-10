package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/filecoin-project/go-filecoin/deployment/process"
	"github.com/ipfs/iptb/testbed/interfaces"
)

func init() {
	process.Debug = true
}

type ID struct {
	ID string
}

func main() {

	if err := os.RemoveAll("./repo"); err != nil {
		panic(err)
	}

	if err := os.MkdirAll("./repo", 0755); err != nil {
		panic(err)
	}

	p, err := process.NewFilecoin(context.Background(), "localfilecoin", "/home/travis/src/github.com/filecoin-project/go-filecoin/deployment/repo")
	if err != nil {
		panic(err)
	}

	process.MustExecute(func() (testbedi.Output, error) {
		return p.Init(context.Background())
	}, "Init")

	process.MustExecute(func() (testbedi.Output, error) {
		return p.Start(context.Background(), true)
	}, "Start")

	id := ID{}
	process.MustExecute(func() (testbedi.Output, error) {
		p.StartGather()
		defer func() {
			io.Copy(os.Stdout, p.StopGather())
		}()
		return nil, p.RunCmdJSON(&id, "go-filecoin", "id")
	}, "Command")

	fmt.Println(id)

	process.MustExecute(func() (testbedi.Output, error) {
		return nil, p.Stop(context.Background())
	}, "Stop")
}
