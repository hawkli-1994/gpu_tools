package all

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	loaders := registry.Loaders()
	require.Len(t, loaders, 11)
	vendors := make([]string, 0, len(loaders))
	for _, loader := range loaders {
		vendors = append(vendors, loader.Vendor())
	}
	assert.Equal(t, []string{
		"AMD", "AMD-riscv", "CPU", "Denglin", "Enflame", "Houmo",
		"Huawei", "Iluvatar", "NVIDIA", "XDX", "mx",
	}, vendors)
}
