package ix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func envValue(t *testing.T, env []string, key string) string {
	t.Helper()
	prefix := key + "="
	for _, kv := range env {
		if value, ok := strings.CutPrefix(kv, prefix); ok {
			return value
		}
	}
	return ""
}

func TestIxsmiEnv(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	ixsmiPath := filepath.Join(binDir, "ixsmi")
	require.NoError(t, os.WriteFile(ixsmiPath, []byte("#!/bin/sh\n"), 0o755))

	env := ixsmiEnv(ixsmiPath)
	newPath := envValue(t, env, "PATH")
	assert.True(t, strings.HasPrefix(newPath, binDir+":"))
	assert.Contains(t, newPath, os.Getenv("PATH"))
	newLibPath := envValue(t, env, "LD_LIBRARY_PATH")
	assert.Contains(t, newLibPath, filepath.Join(tmpDir, "lib"))
	assert.Contains(t, newLibPath, filepath.Join(tmpDir, "lib64"))
	if old := os.Getenv("LD_LIBRARY_PATH"); old != "" {
		assert.Contains(t, newLibPath, old)
	}
}

func TestIxsmiEnvDoesNotMutateGlobalEnv(t *testing.T) {
	origPath := os.Getenv("PATH")
	origLibPath := os.Getenv("LD_LIBRARY_PATH")
	ixsmiEnv("/usr/local/corex-4.4.0/bin/ixsmi")
	assert.Equal(t, origPath, os.Getenv("PATH"))
	assert.Equal(t, origLibPath, os.Getenv("LD_LIBRARY_PATH"))
}

func TestLoadKeepsStderrOutOfXML(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "ixsmi.xml"))
	require.NoError(t, err)
	t.Setenv("IXSMI_FIXTURE", fixture)
	binDir := filepath.Join(t.TempDir(), "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	smiPath := filepath.Join(binDir, "ixsmi")
	script := `#!/bin/sh
if [ "$#" -ne 2 ] || [ "$1" != "-q" ] || [ "$2" != "-x" ]; then
	exit 2
fi
cat "$IXSMI_FIXTURE"
echo 'driver warning' >&2
`
	require.NoError(t, os.WriteFile(smiPath, []byte(script), 0o755))

	infos, err := New().load(context.Background(), smiPath)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, "23490256585496", infos[0].SerialNumber)
}

func TestParseIXSMI(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "ixsmi.xml"))
	require.NoError(t, err)
	infos, err := ParseIXSMI(string(data))
	require.NoError(t, err)
	require.Len(t, infos, 2)

	assert.Equal(t, "00000000:0C:00.0", infos[0].DeviceID)
	assert.Equal(t, "23490256585496", infos[0].SerialNumber)
	assert.Equal(t, gpu.Uint64(34359738368), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(29129441280), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(44), infos[0].TemperatureEdge)
	assert.Nil(t, infos[0].TemperatureJunction)
	assert.Nil(t, infos[0].TemperatureMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:0C:00.0", infos[0].PCIBus)
	assert.Equal(t, "00000000:0F:00.0", infos[1].DeviceID)
	assert.Equal(t, "23490258505496", infos[1].SerialNumber)
	assert.Equal(t, gpu.Uint64(34359738368), infos[1].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(29062332416), infos[1].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(44), infos[1].TemperatureEdge)
	assert.Equal(t, gpu.Float64(0), infos[1].GPUUse)
	assert.Equal(t, "0000:0F:00.0", infos[1].PCIBus)
}

func TestParseIXSMIMissingNumericValues(t *testing.T) {
	infos, err := ParseIXSMI(`<ixsmi_log><gpu id="0"><memory_usage><total>N/A</total></memory_usage></gpu></ixsmi_log>`)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Nil(t, infos[0].VRAMTotalMemory)
	assert.Nil(t, infos[0].VRAMTotalUsedMemory)
	assert.Nil(t, infos[0].TemperatureEdge)
	assert.Nil(t, infos[0].GPUUse)
}

func TestParseIXSMIInvalidNumericValue(t *testing.T) {
	_, err := ParseIXSMI(`<ixsmi_log><gpu id="0"><temperature><gpu_temp>invalid C</gpu_temp></temperature></gpu></ixsmi_log>`)
	assert.Error(t, err)
}

func TestParseIXSMIDeviceIdentityPowerAndUnits(t *testing.T) {
	infos, err := ParseIXSMI(`<ixsmi_log><attached_gpus>1</attached_gpus><gpu id="gpu7"><minor_number>7</minor_number><power_readings><gpu_power_draw>37.25 W</gpu_power_draw></power_readings></gpu></ixsmi_log>`)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 7, infos[0].Num)
	assert.Equal(t, gpu.Float64(37.25), infos[0].AverageGraphicsPackagePower)

	data, err := os.ReadFile(filepath.Join("testdata", "ixsmi.xml"))
	require.NoError(t, err)
	infos, err = ParseIXSMI(string(data))
	require.NoError(t, err)
	assert.Equal(t, gpu.Float64(37), infos[0].AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Float64(39), infos[1].AverageGraphicsPackagePower)
}

func TestParseIXSMIRejectsInvalidUnitsAndDeviceCount(t *testing.T) {
	for _, input := range []string{
		`<ixsmi_log><gpu id="0"><memory_usage><total>1 GB</total></memory_usage></gpu></ixsmi_log>`,
		`<ixsmi_log><gpu id="0"><temperature><gpu_temp>53 W</gpu_temp></temperature></gpu></ixsmi_log>`,
		`<ixsmi_log><gpu id="0"><utilization><gpu_util>10 C</gpu_util></utilization></gpu></ixsmi_log>`,
		`<ixsmi_log><gpu id="0"><power_readings><gpu_power_draw>10 C</gpu_power_draw></power_readings></gpu></ixsmi_log>`,
		`<ixsmi_log><attached_gpus>2</attached_gpus><gpu id="0"/></ixsmi_log>`,
	} {
		_, err := ParseIXSMI(input)
		assert.Error(t, err, input)
	}
}
