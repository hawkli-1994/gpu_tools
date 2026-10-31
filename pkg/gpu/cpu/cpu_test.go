package cpu

import (
	"context"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
)

func TestLoader(t *testing.T) {
	loader := New()
	assert.Equal(t, "CPU", loader.Vendor())
	infos, err := loader.Load(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, infos)
	var _ gpu.Loader = loader
}
