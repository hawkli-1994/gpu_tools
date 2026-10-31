package houmoai

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSuccessAndArguments(t *testing.T) {
	dir := t.TempDir()
	fixture, err := filepath.Abs(filepath.Join("testdata", "hm_smi.txt"))
	require.NoError(t, err)
	t.Setenv("HM_SMI_FIXTURE", fixture)
	argsPath := filepath.Join(dir, "args")
	t.Setenv("HM_SMI_ARGS", argsPath)
	path := filepath.Join(dir, "hm_smi")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$HM_SMI_ARGS\"\n/bin/cat \"$HM_SMI_FIXTURE\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)

	infos, err := New().Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, infos, 2)
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "-a\n", string(args))
}

func TestLoadFailureAndMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hm_smi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho 'hm_smi failed' >&2\nexit 23\n"), 0o755))
	t.Setenv("PATH", dir)
	_, err := New().Load(context.Background())
	assert.ErrorContains(t, err, "hm_smi failed")
	assert.False(t, errors.Is(err, gpu.ErrUnavailable))

	t.Setenv("PATH", t.TempDir())
	_, err = New().Load(context.Background())
	assert.ErrorIs(t, err, gpu.ErrUnavailable)
}
