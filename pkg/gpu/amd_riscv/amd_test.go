package amdriscv

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeDeviceFile(t *testing.T, root, card, name, value string) {
	t.Helper()
	dir := filepath.Join(root, card, "device")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(value), 0o644))
}

func sensorsCommand(t *testing.T, output string) string {
	t.Helper()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "sensors.json")
	require.NoError(t, os.WriteFile(fixture, []byte(output), 0o644))
	t.Setenv("AMD_SENSORS_FIXTURE", fixture)
	path := filepath.Join(dir, "sensors")
	script := "#!/bin/sh\nif [ \"$#\" -ne 1 ] || [ \"$1\" != \"-j\" ]; then exit 2; fi\ncat \"$AMD_SENSORS_FIXTURE\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestLoadMatchesSensorsByPCIAddress(t *testing.T) {
	root := t.TempDir()
	writeDeviceFile(t, root, "card7", "device", "0x744a\n")
	writeDeviceFile(t, root, "card7", "gpu_busy_percent", "2\n")
	writeDeviceFile(t, root, "card7", "mem_info_vram_total", "48301604864\n")
	writeDeviceFile(t, root, "card7", "mem_info_vram_used", "27910144\n")
	writeDeviceFile(t, root, "card7", "uevent", "DRIVER=amdgpu\nPCI_SLOT_NAME=0000:0e:00.0\n")
	sensors := sensorsCommand(t, `{
		"amdgpu-pci-0e00": {
			"edge": {"temp1_input": 32},
			"mem": {"temp3_input": 0},
			"PPT": {"power1_average": 30.25}
		},
		"amdgpu-pci-0300": {
			"edge": {"temp1_input": 99},
			"junction": {"temp2_input": 99}
		}
	}`)

	infos, err := New().load(context.Background(), root, sensors)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	info := infos[0]
	assert.Equal(t, 7, info.Num)
	assert.Equal(t, "0x744a", info.DeviceID)
	assert.Equal(t, "AMD", info.CardVendor)
	assert.Equal(t, "0000:0e:00.0", info.PCIBus)
	assert.Equal(t, gpu.Float64(2), info.GPUUse)
	assert.Equal(t, gpu.Uint64(48301604864), info.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(27910144), info.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(32), info.TemperatureEdge)
	assert.Nil(t, info.TemperatureJunction)
	assert.Equal(t, gpu.Float64(0), info.TemperatureMemory)
	assert.Equal(t, gpu.Float64(30.25), info.AverageGraphicsPackagePower)
}

func TestLoadUsesSysfsSymlinkForPCIAddress(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(t.TempDir(), "0000:03:00.0")
	require.NoError(t, os.MkdirAll(device, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(device, "device"), []byte("0x744a\n"), 0o644))
	card := filepath.Join(root, "card0")
	require.NoError(t, os.MkdirAll(card, 0o755))
	require.NoError(t, os.Symlink(device, filepath.Join(card, "device")))

	infos, err := New().load(context.Background(), root, sensorsCommand(t, `{
		"amdgpu-pci-0300": {"PPT": {"power1_input": 12.5}}
	}`))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "0000:03:00.0", infos[0].PCIBus)
	assert.Equal(t, gpu.Float64(12.5), infos[0].AverageGraphicsPackagePower)
}

func TestLoadAllowsMissingSensorsAndMetrics(t *testing.T) {
	root := t.TempDir()
	writeDeviceFile(t, root, "card0", "device", "0x744a\n")
	writeDeviceFile(t, root, "card0", "gpu_busy_percent", "4\n")

	infos, err := New().load(context.Background(), root, filepath.Join(t.TempDir(), "missing-sensors"))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, gpu.Float64(4), infos[0].GPUUse)
	assert.Nil(t, infos[0].TemperatureEdge)
	assert.Nil(t, infos[0].TemperatureJunction)
	assert.Nil(t, infos[0].TemperatureMemory)
	assert.Nil(t, infos[0].AverageGraphicsPackagePower)
	assert.Nil(t, infos[0].VRAMTotalMemory)
	assert.Nil(t, infos[0].VRAMTotalUsedMemory)
}

func TestLoadRejectsInvalidSysfsMetric(t *testing.T) {
	root := t.TempDir()
	writeDeviceFile(t, root, "card0", "device", "0x744a\n")
	writeDeviceFile(t, root, "card0", "gpu_busy_percent", "invalid\n")
	_, err := New().load(context.Background(), root, sensorsCommand(t, `{}`))
	assert.Error(t, err)
}

func TestSensorName(t *testing.T) {
	assert.Equal(t, "amdgpu-pci-0e00", sensorName("0000:0e:00.0"))
	assert.Equal(t, "amdgpu-pci-0300", sensorName("03:00.0"))
	assert.Empty(t, sensorName(""))
}
