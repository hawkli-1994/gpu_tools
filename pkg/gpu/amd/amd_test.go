package amd

import (
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestROCMSMIParse(t *testing.T) {
	jsonData := `{"card1":{"Device ID":"0x747e","Device Rev":"0xc8","Temperature (Sensor edge) (C)":"36.0","Temperature (Sensor junction) (C)":"41.0","Temperature (Sensor memory) (C)":"44.0","Average Graphics Package Power (W)":"4.0","GPU use (%)":"0","Serial Number":"5c88007d760374f3","VRAM Total Memory (B)":"17163091968","VRAM Total Used Memory (B)":"283090944","Card series":"0x747e","Card model":"0x7801","Card vendor":"Advanced Micro Devices, Inc. [AMD/ATI]","Card SKU":"EXT94393","PCI Bus":"0000:0e:00.0"},"card0":{}}`
	infos, err := (&rocmSMICommand{}).parse([]byte(jsonData))
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 0, infos[0].Num)

	info := infos[1]
	assert.Equal(t, "0x747e", info.DeviceID)
	assert.Equal(t, "0xc8", info.DeviceRev)
	assert.Equal(t, gpu.Float64(36), info.TemperatureEdge)
	assert.Equal(t, gpu.Float64(41), info.TemperatureJunction)
	assert.Equal(t, gpu.Float64(44), info.TemperatureMemory)
	assert.Equal(t, gpu.Float64(4), info.AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Float64(0), info.GPUUse)
	assert.Equal(t, "5c88007d760374f3", info.SerialNumber)
	assert.Equal(t, gpu.Uint64(17163091968), info.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(283090944), info.VRAMTotalUsedMemory)
	assert.Equal(t, "0x747e", info.CardSeries)
	assert.Equal(t, "0x7801", info.CardModel)
	assert.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI]", info.CardVendor)
	assert.Equal(t, "EXT94393", info.CardSKU)
	assert.Equal(t, "0000:0e:00.0", info.PCIBus)
}

func TestROCMSMIParseRejectsInvalidMetric(t *testing.T) {
	_, err := (&rocmSMICommand{}).parse([]byte(`{"card0":{"GPU use (%)":"unknown"}}`))
	assert.ErrorContains(t, err, "utilization")
}
