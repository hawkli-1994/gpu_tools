package amd

import (
	"encoding/json"
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

func TestAMDSMIParseLegacyTopLevelArrays(t *testing.T) {
	staticOutput, err := os.ReadFile("testdata/amd_smi_static.json")
	require.NoError(t, err)
	metricOutput, err := os.ReadFile("testdata/amd_smi_metric.json")
	require.NoError(t, err)

	legacyGPUData := func(output []byte) []byte {
		var wrapped struct {
			GPUData json.RawMessage `json:"gpu_data"`
		}
		require.NoError(t, json.Unmarshal(output, &wrapped))
		require.NotEmpty(t, wrapped.GPUData)
		return wrapped.GPUData
	}
	expected, err := (&amdSMICommand{}).parse(staticOutput, metricOutput)
	require.NoError(t, err)
	actual, err := (&amdSMICommand{}).parse(legacyGPUData(staticOutput), legacyGPUData(metricOutput))
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestAMDSMIParseUnavailableMetricGroups(t *testing.T) {
	staticOutput, err := os.ReadFile("testdata/amd_smi_static.json")
	require.NoError(t, err)
	metricOutput, err := os.ReadFile("testdata/amd_smi_metric.json")
	require.NoError(t, err)
	var metric struct {
		GPUData []map[string]json.RawMessage `json:"gpu_data"`
	}
	require.NoError(t, json.Unmarshal(metricOutput, &metric))
	require.Len(t, metric.GPUData, 2)
	for _, group := range []string{"usage", "power", "temperature", "mem_usage"} {
		metric.GPUData[1][group] = json.RawMessage(`"N/A"`)
	}
	metricOutput, err = json.Marshal(metric)
	require.NoError(t, err)

	infos, err := (&amdSMICommand{}).parse(staticOutput, metricOutput)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, gpu.Float64(12), infos[0].GPUUse)
	second := infos[1]
	assert.Nil(t, second.TemperatureEdge)
	assert.Nil(t, second.TemperatureJunction)
	assert.Nil(t, second.TemperatureMemory)
	assert.Nil(t, second.AverageGraphicsPackagePower)
	assert.Nil(t, second.GPUUse)
	assert.Nil(t, second.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Uint64(48301604864), second.VRAMTotalMemory)
	assert.Equal(t, "0xDBA7EBE13A88DDBD", second.SerialNumber)
}

func TestAMDSMIParseLegacyScalarValues(t *testing.T) {
	staticOutput, err := os.ReadFile("testdata/amd_smi_legacy_static.json")
	require.NoError(t, err)
	metricOutput, err := os.ReadFile("testdata/amd_smi_legacy_metric.json")
	require.NoError(t, err)

	infos, err := (&amdSMICommand{}).parse(staticOutput, metricOutput)
	require.NoError(t, err)
	require.Len(t, infos, 1)

	info := infos[0]
	assert.Equal(t, "0x740f", info.DeviceID)
	assert.Equal(t, gpu.Float64(35), info.TemperatureEdge)
	assert.Equal(t, gpu.Float64(42), info.TemperatureJunction)
	assert.Nil(t, info.TemperatureMemory)
	assert.Equal(t, gpu.Float64(31.25), info.AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Float64(7.5), info.GPUUse)
	assert.Equal(t, gpu.Uint64(51539607552), info.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(1073741824), info.VRAMTotalUsedMemory)
}

func TestAMDSMIParseCurrentPowerWithoutSocketPower(t *testing.T) {
	staticOutput, err := os.ReadFile("testdata/amd_smi_legacy_static.json")
	require.NoError(t, err)
	metricOutput := []byte(`[{"gpu":0,"power":{"current_power":{"value":18.5,"unit":"W"}}}]`)

	infos, err := (&amdSMICommand{}).parse(staticOutput, metricOutput)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, gpu.Float64(18.5), infos[0].AverageGraphicsPackagePower)
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
