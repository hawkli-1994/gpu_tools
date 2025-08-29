package nvidia

import (
	"testing"

	_ "embed"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	// Test data based on the example from Python code
	csvData := `0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 1309 MiB, 0 %, 41
1, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 13625 MiB, 5 %, 39`

	nvidia := &nvidiaSMICommand{}

	gpuInfoList, err := nvidia.parse([]byte(csvData))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(gpuInfoList.GPUInfos))

	// Check first GPU
	gpu0 := gpuInfoList.GPUInfos[0]
	assert.Equal(t, 0, gpu0.Num)
	assert.Equal(t, "0", gpu0.DeviceID)
	assert.Equal(t, "NVIDIA GeForce RTX 4080 SUPER", gpu0.CardModel)
	assert.Equal(t, "NVIDIA", gpu0.CardVendor)
	assert.Equal(t, "NVIDIA", gpu0.CardSeries)

	// Memory: 16376 MiB = 16376 * 1024 * 1024 = 17171480576 bytes
	assert.Equal(t, "17171480576", gpu0.VRAMTotalMemory)
	// Memory used: 1309 MiB = 1309 * 1024 * 1024 = 1372585984 bytes
	assert.Equal(t, "1372585984", gpu0.VRAMTotalUsedMemory)

	assert.Equal(t, "0.0", gpu0.GPUUse)
	assert.Equal(t, "41.0", gpu0.TemperatureEdge)
	assert.Equal(t, "41.0", gpu0.TemperatureJunction)
	assert.Equal(t, "41.0", gpu0.TemperatureMemory)

	// Check second GPU
	gpu1 := gpuInfoList.GPUInfos[1]
	assert.Equal(t, 1, gpu1.Num)
	assert.Equal(t, "1", gpu1.DeviceID)
	assert.Equal(t, "NVIDIA GeForce RTX 4080 SUPER", gpu1.CardModel)

	// Memory used: 13625 MiB = 13625 * 1024 * 1024 = 14286848000 bytes
	assert.Equal(t, "14286848000", gpu1.VRAMTotalUsedMemory)
	assert.Equal(t, "5.0", gpu1.GPUUse)
	assert.Equal(t, "39.0", gpu1.TemperatureEdge)
}

func TestParseInvalidData(t *testing.T) {
	nvidia := &nvidiaSMICommand{}

	// Test with insufficient columns
	csvData := `0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB`

	gpuInfoList, err := nvidia.parse([]byte(csvData))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(gpuInfoList.GPUInfos))
}

func TestParseEmptyData(t *testing.T) {
	nvidia := &nvidiaSMICommand{}

	gpuInfoList, err := nvidia.parse([]byte(""))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(gpuInfoList.GPUInfos))
}

//go:embed testdata/nvidia_version.txt
var versionInfo string

func TestParseVersion(t *testing.T) {
	r, err := ParseVersion(versionInfo)
	if err != nil {
		assert.NoError(t, err)
	}
	assert.Equal(t, "550.54.14", r.ClientVersion)
	assert.Equal(t, "550.54.14", r.Version)
	assert.Equal(t, "12.4", r.LibVersion)
}
