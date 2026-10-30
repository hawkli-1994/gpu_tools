package mx

import (
	_ "embed"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/output.txt
var output string

//go:embed testdata/meta_2.2.12.txt
var output212 string

//go:embed testdata/debug-mx.txt
var output234 string

func TestParseMxOutput(t *testing.T) {
	infos, err := parseMxOutput(output)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Nil(t, infos[0].TemperatureEdge)
	assert.Equal(t, gpu.Float64(44), infos[0].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(62684897280), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:0f:00.0", infos[0].PCIBus)
}

func TestParseMxOutput212(t *testing.T) {
	infos, err := parseMxOutput(output212)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(38), infos[0].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(62379499520), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:19:00.0", infos[0].PCIBus)
	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, "MXN260", infos[1].CardModel)
	assert.Equal(t, gpu.Float64(37.5), infos[1].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[1].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(2467725312), infos[1].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[1].GPUUse)
	assert.Equal(t, "0000:1a:00.0", infos[1].PCIBus)
}

func TestParseMxOutput234(t *testing.T) {
	infos, err := parseMxOutput(output234)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(45.75), infos[0].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(700903424), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:0c:00.0", infos[0].PCIBus)
	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, gpu.Float64(44.25), infos[1].TemperatureJunction)
	assert.Equal(t, "0000:0f:00.0", infos[1].PCIBus)
}

func TestParseMxOutputEmptyOrInvalid(t *testing.T) {
	for _, input := range []string{"", "invalid mx-smi output"} {
		infos, err := parseMxOutput(input)
		require.NoError(t, err)
		assert.Empty(t, infos)
	}
}

func TestParseMxOutputInvalidNumericValue(t *testing.T) {
	_, err := parseMxOutput("GPU#0 MXN260 0000:0f:00.0\nGPU hotspot: invalid C\n")
	assert.Error(t, err)
}
