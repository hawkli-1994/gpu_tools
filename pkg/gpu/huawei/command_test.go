package huawei

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "npu-smi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0o755))
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := New().Load(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
