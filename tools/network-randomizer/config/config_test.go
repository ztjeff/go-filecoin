package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRoundtrip(t *testing.T) {

	dir, err := ioutil.TempDir("", "config")
	assert.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	cfg := NewDefaultConfig()

	cfgpath := filepath.Join(dir, "config.json")
	assert.NoError(t, cfg.WriteFile(cfgpath))

	cfgout, err := ReadFile(cfgpath)
	assert.NoError(t, err)

	assert.Equal(t, cfg, cfgout)
}
