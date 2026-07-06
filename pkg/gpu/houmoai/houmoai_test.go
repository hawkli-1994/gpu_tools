package houmoai

import (
	_ "embed"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/hm_smi.txt
var hmSMIOutput []byte

func TestParse(t *testing.T) {
	h := &houmoaiSMICommand{}
	gpuInfoList, err := h.parse(hmSMIOutput)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(gpuInfoList.GPUInfos))

	// device1 appears first in the fixture
	gpu0 := gpuInfoList.GPUInfos[0]
	assert.Equal(t, 0, gpu0.Num)
	assert.Equal(t, "1", gpu0.DeviceID)
	assert.Equal(t, "Houmo", gpu0.CardVendor)
	assert.Equal(t, "Houmo", gpu0.CardSeries)
	assert.Equal(t, "LQ50-18GB", gpu0.CardModel)
	assert.Equal(t, "100I2010", gpu0.CardSKU)
	assert.Equal(t, "0103020100002025003800000003", gpu0.SerialNumber)
	assert.Equal(t, "V1.1.1", gpu0.DeviceRev)
	assert.Equal(t, "0000:18:00.0", gpu0.PCIBus)
	assert.Equal(t, "0.0", gpu0.GPUUse)
	assert.Equal(t, "19193135104", gpu0.VRAMTotalMemory)
	assert.Equal(t, "0", gpu0.VRAMTotalUsedMemory)
	assert.Equal(t, "46.8", gpu0.TemperatureEdge)
	assert.Equal(t, "45.9", gpu0.TemperatureJunction)
	assert.Equal(t, "46.5", gpu0.TemperatureMemory)
	assert.Equal(t, "6.7", gpu0.AverageGraphicsPackagePower)

	gpu1 := gpuInfoList.GPUInfos[1]
	assert.Equal(t, 1, gpu1.Num)
	assert.Equal(t, "0", gpu1.DeviceID)
	assert.Equal(t, "LQ50-24GB", gpu1.CardModel)
	assert.Equal(t, "100O2010", gpu1.CardSKU)
	assert.Equal(t, "0000:15:00.0", gpu1.PCIBus)
	assert.Equal(t, "25635586048", gpu1.VRAMTotalMemory)
	assert.Equal(t, "0", gpu1.VRAMTotalUsedMemory)
	assert.Equal(t, "0.0", gpu1.GPUUse)
	assert.Equal(t, "44.9", gpu1.TemperatureEdge)
	assert.Equal(t, "41.5", gpu1.TemperatureJunction)
	assert.Equal(t, "40.9", gpu1.TemperatureMemory)
	assert.Equal(t, "6.6", gpu1.AverageGraphicsPackagePower)
}

func TestParseEmpty(t *testing.T) {
	h := &houmoaiSMICommand{}
	gpuInfoList, err := h.parse([]byte(""))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(gpuInfoList.GPUInfos))
}

func TestVendorAndInterface(t *testing.T) {
	h := New()
	assert.Equal(t, "Houmo", h.Vendor())
	var _ gpu.GPUInfoLoader = h
}
