package amd

import (
	"os"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAMDSMIParse(t *testing.T) {
	staticOutput, err := os.ReadFile("testdata/amd_smi_static.json")
	require.NoError(t, err)
	metricOutput, err := os.ReadFile("testdata/amd_smi_metric.json")
	require.NoError(t, err)

	infos, err := (&amdSMICommand{}).parse(staticOutput, metricOutput)
	require.NoError(t, err)
	require.Len(t, infos, 2)

	first := infos[0]
	assert.Equal(t, 0, first.Num)
	assert.Equal(t, "0x744a", first.DeviceID)
	assert.Equal(t, "0x00", first.DeviceRev)
	assert.Equal(t, gpu.Float64(33), first.TemperatureEdge)
	assert.Equal(t, gpu.Float64(38), first.TemperatureJunction)
	assert.Equal(t, gpu.Float64(36), first.TemperatureMemory)
	assert.Equal(t, gpu.Float64(23), first.AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Float64(12), first.GPUUse)
	assert.Equal(t, "0xDC5D0656268643EC", first.SerialNumber)
	assert.Equal(t, gpu.Uint64(48301604864), first.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(12961447936), first.VRAMTotalUsedMemory)
	assert.Equal(t, "AMD Radeon PRO W7900 Dual Slot", first.CardSeries)
	assert.Equal(t, "N/A", first.CardModel)
	assert.Equal(t, "Advanced Micro Devices Inc. [AMD/ATI]", first.CardVendor)
	assert.Equal(t, "N/A", first.CardSKU)
	assert.Equal(t, "0000:0e:00.0", first.PCIBus)

	second := infos[1]
	assert.Equal(t, 1, second.Num)
	assert.Equal(t, gpu.Float64(0.5), second.GPUUse)
	assert.Equal(t, gpu.Uint64(524288), second.VRAMTotalUsedMemory)
	assert.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI]", second.CardVendor)
}

func TestAMDSMIParseInvalidJSON(t *testing.T) {
	validStatic, err := os.ReadFile("testdata/amd_smi_static.json")
	require.NoError(t, err)

	_, err = (&amdSMICommand{}).parse([]byte("invalid"), []byte("{}"))
	assert.ErrorContains(t, err, "static")
	_, err = (&amdSMICommand{}).parse(validStatic, []byte("invalid"))
	assert.ErrorContains(t, err, "metric")
}

func TestSMIValueBytesRejectsUnknownUnit(t *testing.T) {
	value := smiValue{Value: []byte("1"), Unit: "blocks"}
	_, err := value.bytes()
	assert.ErrorContains(t, err, "unsupported memory unit")
}
