package xdx

import (
	_ "embed"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/output.txt
var output []byte

func TestXDXParse(t *testing.T) {
	expected := []gpu.GPUInfo{
		{
			Num: 0, DeviceID: "0", CardVendor: "XDX", CardModel: "R1900", CardSeries: "PANGU_A0",
			TemperatureEdge: gpu.Float64(35),
			VRAMTotalMemory: gpu.Uint64(16384 * 1024 * 1024), AverageGraphicsPackagePower: gpu.Float64(41), PCIBus: "0000:19:00.0",
		},
		{
			Num: 1, DeviceID: "1", CardVendor: "XDX", CardModel: "R1900", CardSeries: "PANGU_A0",
			TemperatureEdge: gpu.Float64(35),
			VRAMTotalMemory: gpu.Uint64(16384 * 1024 * 1024), AverageGraphicsPackagePower: gpu.Float64(41), PCIBus: "0000:1a:00.0",
		},
	}
	infos, err := New().parse(output)
	require.NoError(t, err)
	assert.Equal(t, expected, infos)
}

func TestXDXParseEmptyData(t *testing.T) {
	infos, err := New().parse(nil)
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestXDXParseInvalidNumericValue(t *testing.T) {
	_, err := New().parse([]byte("GPU 0\nTemperature : invalid C\n"))
	assert.Error(t, err)
}
