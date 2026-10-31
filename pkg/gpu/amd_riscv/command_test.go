package amdriscv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSensorsFailureIncludesStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensors")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho 'sensors failed' >&2\nexit 23\n"), 0o755))
	_, err := New().load(context.Background(), t.TempDir(), path)
	assert.ErrorContains(t, err, "sensors failed")
}

func TestLoadSensorsCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensors")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0o755))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := New().load(ctx, t.TempDir(), path)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestLoadSensorsSuccessKeepsStderrOutOfJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensors")
	script := "#!/bin/sh\necho '{}'\necho 'sensor warning' >&2\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	infos, err := New().load(context.Background(), t.TempDir(), path)
	assert.NoError(t, err)
	assert.Empty(t, infos)
}
