package ix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFailureIncludesStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ixsmi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho 'ixsmi failed' >&2\nexit 23\n"), 0o755))
	_, err := New().load(context.Background(), path)
	assert.ErrorContains(t, err, "ixsmi failed")
	assert.False(t, errors.Is(err, gpu.ErrUnavailable))
}

func TestLoadCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ixsmi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0o755))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := New().load(ctx, path)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestScanCorexSmiPaths(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "corex-test", "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	path := filepath.Join(bin, "ixsmi")
	require.NoError(t, os.WriteFile(path, nil, 0o755))
	paths, err := scanCorexSmiPaths(root)
	require.NoError(t, err)
	assert.Equal(t, []string{path}, paths)
	_, err = scanCorexSmiPaths(filepath.Join(root, "missing"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}
