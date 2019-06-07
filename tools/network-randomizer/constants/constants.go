package constants

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
)

var (
	BinPath      string
	BlockTime    = 5 * time.Second
	SmallSectors = true
)

func init() {
	var err error
	// We default to the binary built in the project directory, fallback
	// to searching path.
	BinPath, err = getFilecoinBinary()
	if err != nil {
		BinPath, err = exec.LookPath("go-filecoin")
		if err != nil {
			panic(err)
		}
	}
}

func getFilecoinBinary() (string, error) {
	gopath, err := getGoPath()
	if err != nil {
		return "", err
	}

	bin := filepath.Join(gopath, "/src/github.com/filecoin-project/go-filecoin/go-filecoin")
	_, err = os.Stat(bin)
	if err != nil {
		return "", err
	}

	if os.IsNotExist(err) {
		return "", err
	}

	return bin, nil
}

func getGoPath() (string, error) {
	gp := os.Getenv("GOPATH")
	if gp != "" {
		return gp, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "go"), nil
}
