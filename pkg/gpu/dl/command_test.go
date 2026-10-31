package dl

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSuccessAndArguments(t *testing.T) {
	dir := t.TempDir()
	fixture, err := filepath.Abs(filepath.Join("testdata", "dlsmi_output.xml"))
	require.NoError(t, err)
	t.Setenv("DLSMI_FIXTURE", fixture)
	argsPath := filepath.Join(dir, "args")
	t.Setenv("DLSMI_ARGS", argsPath)
	path := filepath.Join(dir, "dlsmi")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$DLSMI_ARGS\"\n/bin/cat \"$DLSMI_FIXTURE\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)

	infos, err := New().Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, infos, 8)
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "query\n--xml-format\n", string(args))
}
