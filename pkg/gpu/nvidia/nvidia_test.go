package nvidia

import (
	_ "embed"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	csvData := `0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 1309 MiB, 0 %, 41, 00000000:50:00.0
1, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 13625 MiB, 5 %, 39, 00000000:51:00.0`
	infos, err := (&nvidiaSMICommand{}).parse([]byte(csvData))
	require.NoError(t, err)
	require.Len(t, infos, 2)

	first := infos[0]
	assert.Equal(t, 0, first.Num)
	assert.Equal(t, "0", first.DeviceID)
	assert.Equal(t, "NVIDIA GeForce RTX 4080 SUPER", first.CardModel)
	assert.Equal(t, "NVIDIA", first.CardVendor)
	assert.Equal(t, "NVIDIA", first.CardSeries)
	assert.Equal(t, "00000000:50:00.0", first.PCIBus)
	assert.Equal(t, gpu.Uint64(17171480576), first.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(1372585984), first.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), first.GPUUse)
	assert.Equal(t, gpu.Float64(41), first.TemperatureEdge)
	assert.Nil(t, first.TemperatureJunction)
	assert.Nil(t, first.TemperatureMemory)
	assert.Nil(t, first.AverageGraphicsPackagePower)

	second := infos[1]
	assert.Equal(t, 1, second.Num)
	assert.Equal(t, gpu.Uint64(14286848000), second.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(5), second.GPUUse)
	assert.Equal(t, gpu.Float64(39), second.TemperatureEdge)
}

func TestParseInvalidData(t *testing.T) {
	_, err := (&nvidiaSMICommand{}).parse([]byte(`0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB`))
	assert.ErrorContains(t, err, "expected 7 columns")
}

func TestParseInvalidMetric(t *testing.T) {
	_, err := (&nvidiaSMICommand{}).parse([]byte(`0, NVIDIA, invalid MiB, 0 MiB, 0 %, 41, 00000000:50:00.0`))
	assert.ErrorContains(t, err, "total memory")
}

func TestParseEmptyData(t *testing.T) {
	infos, err := (&nvidiaSMICommand{}).parse(nil)
	assert.NoError(t, err)
	assert.Empty(t, infos)
}

//go:embed testdata/nvidia_version.txt
var versionInfo string

func TestParseVersion(t *testing.T) {
	info, err := ParseVersion(versionInfo)
	require.NoError(t, err)
	assert.Equal(t, "550.54.14", info.ClientVersion)
	assert.Equal(t, "550.54.14", info.Version)
	assert.Equal(t, "12.4", info.LibVersion)
}
