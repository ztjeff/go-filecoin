package process

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	iptb "github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/address"
	dockerplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/docker"
	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

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

	_, err = iptb.RegisterPlugin(iptb.IptbPlugin{
		From:       "<builtin>",
		NewNode:    dockerplugin.NewNode,
		PluginName: dockerplugin.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}
}

type Filecoin struct {
	testbedi.Core

	ID                peer.ID
	DefaultWalletAddr address.Address

	// Wallet addr backing the miner
	MinerAddress address.Address
	MinerOwner   address.Address

	pluginType string
	pluginDir  string

	logWindow *LogWindow

	out testbedi.Output

	ctx context.Context
}

func NewProcess(ctx context.Context, t, d string, c testbedi.Core) *Filecoin {
	return &Filecoin{
		Core: c,

		pluginType: t,
		pluginDir:  d,

		ctx: ctx,
	}
}

// CopyLogWindow copies the last log window to out.
func (f *Filecoin) CopyLogWindow(out io.Writer) error {
	_, err := io.Copy(out, f.logWindow.Window())

	return err
}

func (f *Filecoin) DumpLastOutput() {
	fmt.Println("=============== CMD ===================")
	fmt.Printf("%3d %s\n", f.out.ExitCode(), f.out.Args())
	fmt.Println("=============== STDOUT ================")
	io.Copy(os.Stdout, f.out.Stdout())
	fmt.Println("=============== STDERR ================")
	io.Copy(os.Stdout, f.out.Stderr())
	fmt.Println("=============== LOGS ==================")
	if !f.logWindow.Empty() {
		f.CopyLogWindow(os.Stdout)
	}
	fmt.Println("=============== END ===================")
}

func (f *Filecoin) RunCmd(ctx context.Context, stdin io.Reader, args ...string) (testbedi.Output, error) {
	stopCapture := f.logWindow.StartCapture()
	defer func() {
		stopCapture()
	}()

	out, err := f.Core.RunCmd(ctx, stdin, args...)
	f.out = out
	return out, err
}

// MustRunCmdJSON runs `args` against TestNode. The '--enc=json' flag is appened to the command specified by `args`,
// the result of the command is marshaled into `v`.
func (f *Filecoin) RunCmdJSON(v interface{}, args ...string) error {
	return f.RunCmdJSONWithStdin(v, nil, args...)
}

// MustRunCmdJSONWithStdin runs `args` against TestNode. The '--enc=json' flag is appened to the command specified by `args`,
// the result of the command is marshaled into `v`.
func (f *Filecoin) RunCmdJSONWithStdin(v interface{}, stdin io.Reader, args ...string) error {
	args = append(args, "--enc=json")
	out, err := f.RunCmd(f.ctx, stdin, args...)
	if err != nil {
		return err
	}
	// did the command exit with nonstandard exit code?
	if out.ExitCode() > 0 {
		return errors.New(fmt.Sprintf("Filecoin command: %s, exited with non-zero exitcode: %d", out.Args(), out.ExitCode()))
	}

	dec := json.NewDecoder(out.Stdout())
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

type LineIterator interface {
	Next() ([]byte, error)
	HasNext() bool
	Close() error
}

type LdJsonIterator struct {
	input *bufio.Reader
	buf   []byte
}

func NewLdJsonIterator(input io.Reader) *LdJsonIterator {
	return &LdJsonIterator{
		input: bufio.NewReader(input),
	}
}

func (ldj *LdJsonIterator) Next() ([]byte, error) {
	if len(ldj.buf) != 0 {
		buf := ldj.buf
		ldj.buf = []byte{}
		return buf, nil
	}

	return ldj.input.ReadBytes('\n')
}

func (ldj *LdJsonIterator) Close() error {
	return nil
}

func (ldj *LdJsonIterator) HasNext() bool {
	buf, err := ldj.Next()
	if err == io.EOF {
		return false
	}

	if err != nil {
		return false
	}

	ldj.buf = buf

	return true
}

func (f *Filecoin) RunCmdLDJSON(args ...string) (LineIterator, error) {
	return f.RunCmdLDJSONWithStdin(nil, args...)
}

func (f *Filecoin) RunCmdLDJSONWithStdin(stdin io.Reader, args ...string) (LineIterator, error) {
	args = append(args, "--enc=json")
	out, err := f.RunCmd(f.ctx, stdin, args...)
	if err != nil {
		return nil, err
	}

	// did the command exit with nonstandard exit code?
	if out.ExitCode() > 0 {
		return nil, errors.New(fmt.Sprintf("Filecoin command: %s, exited with non-zero exitcode: %d", out.Args(), out.ExitCode()))
	}

	return NewLdJsonIterator(out.Stdout()), nil
}
