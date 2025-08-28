package amd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	jsonData := `{"card0":{"Device ID":"0x747e","Device Rev":"0xc8","Temperature (Sensor edge) (C)":"36.0","Temperature (Sensor junction) (C)":"41.0","Temperature (Sensor memory) (C)":"44.0","Average Graphics Package Power (W)":"4.0","GPU use (%)":"0","Serial Number":"5c88007d760374f3","VRAM Total Memory (B)":"17163091968","VRAM Total Used Memory (B)":"283090944","Card series":"0x747e","Card model":"0x7801","Card vendor":"Advanced Micro Devices, Inc. [AMD/ATI]","Card SKU":"EXT94393"}}`
	amd := &rocmSMICommand{}

	gpuInfoList, err := amd.parse([]byte(jsonData))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(gpuInfoList.GPUInfos))
	assert.Equal(t, "0x747e", gpuInfoList.GPUInfos[0].DeviceID)
	assert.Equal(t, "0xc8", gpuInfoList.GPUInfos[0].DeviceRev)
	assert.Equal(t, "36.0", gpuInfoList.GPUInfos[0].TemperatureEdge)
	assert.Equal(t, "41.0", gpuInfoList.GPUInfos[0].TemperatureJunction)
	assert.Equal(t, "44.0", gpuInfoList.GPUInfos[0].TemperatureMemory)
	assert.Equal(t, "4.0", gpuInfoList.GPUInfos[0].AverageGraphicsPackagePower)
	assert.Equal(t, "0", gpuInfoList.GPUInfos[0].GPUUse)
	assert.Equal(t, "5c88007d760374f3", gpuInfoList.GPUInfos[0].SerialNumber)
	assert.Equal(t, "17163091968", gpuInfoList.GPUInfos[0].VRAMTotalMemory)
	assert.Equal(t, "283090944", gpuInfoList.GPUInfos[0].VRAMTotalUsedMemory)
	assert.Equal(t, "0x747e", gpuInfoList.GPUInfos[0].CardSeries)
	assert.Equal(t, "0x7801", gpuInfoList.GPUInfos[0].CardModel)
	assert.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI]", gpuInfoList.GPUInfos[0].CardVendor)
	assert.Equal(t, "EXT94393", gpuInfoList.GPUInfos[0].CardSKU)
	assert.Equal(t, 0, gpuInfoList.GPUInfos[0].Num)
}
