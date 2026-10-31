package enflame

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSuccessAndArguments(t *testing.T) {
	dir := t.TempDir()
	fixture, err := filepath.Abs(filepath.Join("testdata", "efs-15.txt"))
	require.NoError(t, err)
	t.Setenv("EFSMI_FIXTURE", fixture)
	argsPath := filepath.Join(dir, "args")
	t.Setenv("EFSMI_ARGS", argsPath)
	path := filepath.Join(dir, "efsmi")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$EFSMI_ARGS\"\n/bin/cat \"$EFSMI_FIXTURE\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)

	infos, err := New().Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, infos, 2)
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "-q\n-d\nTEMP,MEMORY,USAGE,PCIE\n", string(args))
}

func TestLoadCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "efsmi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0o755))
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := New().Load(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
