package amd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAMDSMILoadUsesExpectedArguments(t *testing.T) {
	tempDir := t.TempDir()
	testdataDir, err := filepath.Abs("testdata")
	require.NoError(t, err)
	commandPath := filepath.Join(tempDir, "amd-smi")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$AMD_SMI_ARGS_DIR/$1"
case "$1" in
static) /bin/cat "$AMD_SMI_TESTDATA/amd_smi_static.json" ;;
metric) /bin/cat "$AMD_SMI_TESTDATA/amd_smi_metric.json" ;;
esac
`
	require.NoError(t, os.WriteFile(commandPath, []byte(script), 0o755))
	t.Setenv("PATH", tempDir)
	t.Setenv("AMD_SMI_ARGS_DIR", tempDir)
	t.Setenv("AMD_SMI_TESTDATA", testdataDir)

	infos, err := (&amdSMICommand{}).Load(context.Background())
	require.NoError(t, err)
	require.Len(t, infos, 2)

	for _, test := range []struct {
		subcommand string
		expected   []string
	}{
		{subcommand: "static", expected: []string{"static", "-a", "-b", "-v", "-B", "--json"}},
		{subcommand: "metric", expected: []string{"metric", "-m", "-u", "-p", "-t", "--json"}},
	} {
		output, err := os.ReadFile(filepath.Join(tempDir, test.subcommand))
		require.NoError(t, err)
		assert.Equal(t, test.expected, strings.Split(strings.TrimSpace(string(output)), "\n"))
	}
}

func TestAMDSMILoadErrorsIncludeStderr(t *testing.T) {
	for _, subcommand := range []string{"static", "metric"} {
		t.Run(subcommand, func(t *testing.T) {
			tempDir := t.TempDir()
			testdataDir, err := filepath.Abs("testdata")
			require.NoError(t, err)
			commandPath := filepath.Join(tempDir, "amd-smi")
			script := `#!/bin/sh
if [ "$1" = "$AMD_SMI_FAIL" ]; then
	printf '%s failed\n' "$1" >&2
	exit 1
fi
/bin/cat "$AMD_SMI_TESTDATA/amd_smi_$1.json"
`
			require.NoError(t, os.WriteFile(commandPath, []byte(script), 0o755))
			t.Setenv("PATH", tempDir)
			t.Setenv("AMD_SMI_TESTDATA", testdataDir)
			t.Setenv("AMD_SMI_FAIL", subcommand)

			_, err = (&amdSMICommand{}).Load(context.Background())
			assert.ErrorContains(t, err, subcommand+" failed")
			var opErr *gpu.OpError
			require.ErrorAs(t, err, &opErr)
			assert.Equal(t, subcommand, opErr.Op)
		})
	}
}

func TestAMDSMILoadUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := (&amdSMICommand{}).Load(context.Background())
	assert.ErrorIs(t, err, gpu.ErrUnavailable)
}

func TestAMDSMILoadContextCanceled(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "amd-smi")
	require.NoError(t, os.WriteFile(commandPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", tempDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (&amdSMICommand{}).Load(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

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

func TestSMIValueBytesRejectsInvalidQuantities(t *testing.T) {
	for _, text := range []string{"NaN", "Inf", "-Inf", "18446744073709551616"} {
		t.Run(text, func(t *testing.T) {
			value := smiValue{Value: []byte(strconv.Quote(text)), Unit: "B"}
			parsed, err := value.bytes()
			assert.Nil(t, parsed)
			assert.ErrorContains(t, err, "invalid byte quantity")
		})
	}
}
