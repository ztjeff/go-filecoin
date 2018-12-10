package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	//"github.com/filecoin-project/go-filecoin/api"

	iptb "github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"

	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

var Debug = false

// must register all plugins
func init() {
	_, err := iptb.RegisterPlugin(iptb.IptbPlugin{
		From:       "<builtin>",
		NewNode:    localplugin.NewNode,
		PluginName: localplugin.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}
}

func MustExecute(fn func() (testbedi.Output, error), msg string) {
	fmt.Println(msg)

	output, err := fn()

	if err != nil {
		panic(err)
	}

	if output == nil {
		return
	}

	// did the command exit with nonstandard exit code?
	if output.ExitCode() > 0 {
		io.Copy(os.Stderr, output.Stderr())
		panic(fmt.Errorf("Filecoin command: %s, exited with non-zero exitcode: %d", output.Args(), output.ExitCode()))
	}

	io.Copy(os.Stdout, output.Stdout())
}

type Filecoin struct {
	testbedi.Core //localfilecoin, dockerfilecoin, kubernetesfilecoin

	pluginType string
	pluginDir  string

	ctx context.Context

	stderr io.Reader
}

func NewFilecoin(ctx context.Context, t, d string) (*Filecoin, error) {
	ns := iptb.NodeSpec{
		Type: t,
		Dir:  d,
	}

	c, err := ns.Load()
	if err != nil {
		return nil, err
	}

	return &Filecoin{
		Core: c,

		pluginType: t,
		pluginDir:  d,

		ctx: ctx,
	}, nil
}

func (f *Filecoin) openStderr() error {
	mn, ok := f.Core.(testbedi.Metric)
	if !ok {
		return fmt.Errorf("iptb metric not implemented")
	}

	stderr, err := mn.StderrReader()
	if err != nil {
		return err
	}

	f.stderr = stderr

	return nil
}

type NullWriter struct {
}

func NewNullWriter() io.Writer {
	return &NullWriter{}
}

func (nw *NullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (f *Filecoin) Start(ctx context.Context, wait bool, args ...string) (testbedi.Output, error) {
	defer f.openStderr()
	return f.Core.Start(ctx, wait, args...)
}

func (f *Filecoin) FastForward() {
	nw := NewNullWriter()
	io.Copy(nw, f.stderr)
}

func (f *Filecoin) StartGather() func() io.Reader {
	f.FastForward()

	return f.StopGather
}

func (f *Filecoin) StopGather() io.Reader {
	buf := make([]byte, 100000)
	bb := bytes.NewBuffer(buf)
	io.Copy(bb, f.stderr)
	return bb
}

// MustRunCmdJSON runs `args` against TestNode. The '--enc=json' flag is appened to the command specified by `args`,
// the result of the command is marshaled into `expOut`.
func (f *Filecoin) RunCmdJSON(out interface{}, args ...string) error {
	args = append(args, "--enc=json")
	mn, ok := f.Core.(testbedi.Metric)
	if !ok {
		return fmt.Errorf("iptb metric not implemented")
	}

	fmt.Println("Getting events")
	evs, err := mn.Events()
	if err != nil {
		return err
	}
	defer evs.Close()
	fmt.Println("Start...")

	go func() {
		buf := make([]byte, 100000)
		bb := bytes.NewBuffer(buf)
		io.Copy(bb, evs)
		io.Copy(os.Stdout, bb)
	}()

	fmt.Println("Start")
	output, err := f.RunCmd(f.ctx, nil, args...)
	if err != nil {
		return err
	}

	fmt.Println("Done")

	// did the command exit with nonstandard exit code?
	if output.ExitCode() > 0 {
		return fmt.Errorf("Filecoin command: %s, exited with non-zero exitcode: %d", output.Args(), output.ExitCode())
	}

	dec := json.NewDecoder(output.Stdout())
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("Failed to decode output from command: %s to struct: %s", output.Args(), reflect.TypeOf(out).Name())
	}

	return nil
}
