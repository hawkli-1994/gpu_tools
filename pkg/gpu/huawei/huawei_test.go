package huawei

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*.txt
var testdataFS embed.FS

func TestExtractNPUIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "single_npu", input: "| 2944    310P3 |\n| 0       0     |", expected: []string{"2944"}},
		{name: "dual_npu_dedup", input: "| 2944    310P3 |\n| 2944    310P3 |\n| 5678    910B  |", expected: []string{"2944", "5678"}},
		{name: "header_should_not_match", input: "| NPU Name |\n| Chip Device |\n| 2944 310P3 |", expected: []string{"2944"}},
		{name: "empty_output", input: "", expected: nil},
		{name: "no_number", input: "| abc def |", expected: nil},
		{name: "whitespace", input: "| 1234 310P3 |\n|\t5678\t910B\t|", expected: []string{"1234", "5678"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := extractNPUIDs([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, ids)
		})
	}
}

func TestParseChipSectionsForward(t *testing.T) {
	t.Run("single_chip", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2

Chip ID                        : 0
Memory Usage Rate(%)           : 2
Aicore Usage Rate(%)           : 0
Temperature(C)                 : 45`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 1)
		assert.Equal(t, "45", got["0"]["Temperature(C)"])
		assert.Equal(t, "0", got["0"]["Aicore Usage Rate(%)"])
	})

	t.Run("dual_chip_with_mcu", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2

Chip ID                        : 0
Memory Usage Rate(%)           : 2
Aicore Usage Rate(%)           : 0
Temperature(C)                 : 45

Chip ID                        : 1
Memory Usage Rate(%)           : 3
Aicore Usage Rate(%)           : 0
Temperature(C)                 : 47

Chip Name                      : mcu
Temperature(C)                 : 48
NPU Real-time Power(W)         : 42.8`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 2)
		assert.Equal(t, "45", got["0"]["Temperature(C)"])
		assert.Equal(t, "47", got["1"]["Temperature(C)"])
		_, ok := got["1"]["NPU Real-time Power(W)"]
		assert.False(t, ok, "MCU power should not leak into chip 1")
		_, ok = got["1"]["Chip Name"]
		assert.False(t, ok, "MCU chip name should not leak into chip 1")
	})

	t.Run("empty_after_filter", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2`
		got := parseChipSections([]byte(input))
		assert.Nil(t, got)
	})

	t.Run("no_chip_id", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2
Power Dissipation(W)           : 42.9`
		got := parseChipSections([]byte(input))
		assert.Nil(t, got)
	})
}

func TestParseChipSectionsReverse(t *testing.T) {
	t.Run("usages_dual_chip", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2

DDR Capacity(MB)               : 44278
DDR Usage Rate(%)              : 2
Aicore Usage Rate(%)           : 0
Chip ID                        : 0

DDR Capacity(MB)               : 43693
DDR Usage Rate(%)              : 3
Aicore Usage Rate(%)           : 0
Chip ID                        : 1`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 2)
		assert.Equal(t, "44278", got["0"]["DDR Capacity(MB)"])
		assert.Equal(t, "2", got["0"]["DDR Usage Rate(%)"])
		assert.Equal(t, "43693", got["1"]["DDR Capacity(MB)"])
		assert.Equal(t, "3", got["1"]["DDR Usage Rate(%)"])
	})

	t.Run("memory_dual_chip", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2

Total DDR Capacity(MB)         : 49152
DDR Capacity(MB)               : 44278
DDR Clock Speed(MHz)           : 451
Chip ID                        : 0

Total DDR Capacity(MB)         : 49152
DDR Capacity(MB)               : 43693
DDR Clock Speed(MHz)           : 451
Chip ID                        : 1`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 2)
		assert.Equal(t, "44278", got["0"]["DDR Capacity(MB)"])
		assert.Equal(t, "451", got["0"]["DDR Clock Speed(MHz)"])
		assert.Equal(t, "43693", got["1"]["DDR Capacity(MB)"])
	})

	t.Run("product_dual_chip", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2

Product Type                   : Atlas 300I Duo
Chip ID                        : 0

Product Type                   : Atlas 300I Duo
Chip ID                        : 1`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 2)
		assert.Equal(t, "Atlas 300I Duo", got["0"]["Product Type"])
		assert.Equal(t, "Atlas 300I Duo", got["1"]["Product Type"])
	})

	t.Run("health_dual_chip_with_mcu", func(t *testing.T) {
		input := `NPU ID                         : 2944
Chip Count                     : 2

Health                         : OK
Chip ID                        : 0

Health                         : OK
Chip ID                        : 1

Health                         : OK
Chip Name                      : MCU`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 2)
		assert.Equal(t, "OK", got["0"]["Health"])
		assert.Equal(t, "OK", got["1"]["Health"])
		_, ok := got["mcu"]
		assert.False(t, ok, "MCU should not appear as a chip")
	})

	t.Run("single_chip", func(t *testing.T) {
		input := `NPU ID                         : 1234
Chip Count                     : 1

DDR Capacity(MB)               : 22000
Chip ID                        : 0`
		got := parseChipSections([]byte(input))
		assert.NotNil(t, got)
		assert.Len(t, got, 1)
		assert.Equal(t, "22000", got["0"]["DDR Capacity(MB)"])
	})
}

func TestParseBoardOutput(t *testing.T) {
	got := parseBoardOutput([]byte(`NPU ID : 2944
Serial Number : SN123
PCIe Bus Info : 0000:0C:00.0
Chip Count : 2`))
	assert.Equal(t, "SN123", got["Serial Number"])
	assert.Equal(t, "0000:0C:00.0", got["PCIe Bus Info"])
	assert.NotContains(t, got, "NPU ID")
	assert.Empty(t, parseBoardOutput(nil))
}

func TestBuildGPUInfoList(t *testing.T) {
	board := map[string]string{"Serial Number": "SN12345", "PCIe Bus Info": "0000:0C:00.0"}
	common := map[string]map[string]string{
		"0": {"Temperature(C)": "45", "Aicore Usage Rate(%)": "10"},
		"1": {"Temperature(C)": "47", "Aicore Usage Rate(%)": "20"},
	}
	usages := map[string]map[string]string{
		"0": {"DDR Capacity(MB)": "44278", "DDR Usage Rate(%)": "2", "Aicore Usage Rate(%)": "10"},
		"1": {"DDR Capacity(MB)": "43693", "DDR Usage Rate(%)": "3", "Aicore Usage Rate(%)": "20"},
	}
	product := map[string]map[string]string{"0": {"Product Type": "Atlas 300I Duo"}, "1": {"Product Type": "Atlas 300I Duo"}}
	power := map[string]string{"Power Dissipation(W)": "43.0"}

	infos, nextNum, err := buildGPUInfoList("2944", board, common, usages, product, power, 0)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 2, nextNum)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "0", infos[0].DeviceID)
	assert.Equal(t, "Huawei", infos[0].CardVendor)
	assert.Equal(t, "Ascend", infos[0].CardSeries)
	assert.Equal(t, "Atlas 300I Duo", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(45), infos[0].TemperatureEdge)
	assert.Nil(t, infos[0].TemperatureJunction)
	assert.Nil(t, infos[0].TemperatureMemory)
	assert.Equal(t, gpu.Float64(10), infos[0].GPUUse)
	assert.Equal(t, "SN12345", infos[0].SerialNumber)
	assert.Equal(t, "0000:0C:00.0", infos[0].PCIBus)
	assert.Equal(t, gpu.Float64(43), infos[0].AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Uint64(46428848128), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(928576962), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, gpu.Float64(47), infos[1].TemperatureEdge)
	assert.Equal(t, gpu.Float64(20), infos[1].GPUUse)
}

func TestBuildGPUInfoListMissingValues(t *testing.T) {
	infos, nextNum, err := buildGPUInfoList("2944", nil, nil, map[string]map[string]string{
		"0": {"DDR Capacity(MB)": "10000", "DDR Usage Rate(%)": "50", "Aicore Usage Rate(%)": "NA"},
	}, nil, map[string]string{"Power Dissipation(W)": "NA"}, 5)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 6, nextNum)
	assert.Equal(t, 5, infos[0].Num)
	assert.Nil(t, infos[0].TemperatureEdge)
	assert.Nil(t, infos[0].GPUUse)
	assert.Nil(t, infos[0].AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Uint64(10485760000), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(5242880000), infos[0].VRAMTotalUsedMemory)

	infos, _, err = buildGPUInfoList("2944", nil, map[string]map[string]string{
		"0": {"Temperature(C)": "50", "Aicore Usage Rate(%)": "30"},
	}, nil, nil, nil, 0)
	require.NoError(t, err)
	assert.Nil(t, infos[0].VRAMTotalMemory)
	assert.Nil(t, infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(30), infos[0].GPUUse)
}

func TestBuildGPUInfoListInvalidNumericValue(t *testing.T) {
	_, _, err := buildGPUInfoList("2944", nil, map[string]map[string]string{
		"0": {"Temperature(C)": "invalid"},
	}, nil, nil, nil, 0)
	assert.Error(t, err)
}

func TestBuildGPUInfoListNoChips(t *testing.T) {
	infos, nextNum, err := buildGPUInfoList("2944", nil, nil, nil, nil, nil, 3)
	require.NoError(t, err)
	assert.Empty(t, infos)
	assert.Equal(t, 3, nextNum)
}

func TestNew(t *testing.T) {
	loader := New()
	assert.Equal(t, "Huawei", loader.Vendor())
	var _ gpu.Loader = loader
}

func TestLoadReturnsSubcommandError(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
if [ "$#" -eq 1 ] && [ "$1" = "info" ]; then
	echo '| 2944 310P3 |'
	exit 0
fi
echo 'board query failed' >&2
exit 1
`
	path := filepath.Join(dir, "npu-smi")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)

	infos, err := New().Load(context.Background())
	assert.Error(t, err)
	assert.Nil(t, infos)
	assert.Contains(t, err.Error(), "board")
	assert.Contains(t, err.Error(), "board query failed")
}

func TestIntegrationDualChipFromFixtures(t *testing.T) {
	read := func(name string) []byte {
		data, err := testdataFS.ReadFile("testdata/" + name)
		require.NoError(t, err)
		return data
	}
	ids, err := extractNPUIDs(read("npu_info.txt"))
	require.NoError(t, err)
	assert.Equal(t, []string{"2944"}, ids)
	infos, nextNum, err := buildGPUInfoList("2944", parseBoardOutput(read("npu_board.txt")), parseChipSections(read("npu_common.txt")), parseChipSections(read("npu_usages.txt")), parseChipSections(read("npu_product.txt")), parseBoardOutput(read("npu_power.txt")), 0)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 2, nextNum)
	assert.Equal(t, "Atlas 300I Duo", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(45), infos[0].TemperatureEdge)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "2106030737ZERC003572", infos[0].SerialNumber)
	assert.Equal(t, gpu.Float64(42.9), infos[0].AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Uint64(46428848128), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(928576962), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Uint64(45815431168), infos[1].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(1374462935), infos[1].VRAMTotalUsedMemory)
}

func TestIntegrationSingleChipFromFixtures(t *testing.T) {
	read := func(name string) []byte {
		data, err := testdataFS.ReadFile("testdata/" + name)
		require.NoError(t, err)
		return data
	}
	ids, err := extractNPUIDs(read("npu_single_chip.txt"))
	require.NoError(t, err)
	assert.Equal(t, []string{"1234"}, ids)
	infos, nextNum, err := buildGPUInfoList("1234", parseBoardOutput(read("npu_single_board.txt")), parseChipSections(read("npu_single_common.txt")), parseChipSections(read("npu_single_usages.txt")), parseChipSections(read("npu_single_product.txt")), parseBoardOutput(read("npu_single_power.txt")), 0)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 1, nextNum)
	assert.Equal(t, "Atlas 300I Pro", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(55), infos[0].TemperatureEdge)
	assert.Equal(t, gpu.Float64(10), infos[0].GPUUse)
	assert.Equal(t, "SN987654321", infos[0].SerialNumber)
	assert.Equal(t, gpu.Float64(65.5), infos[0].AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Uint64(23068672000), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(2306867200), infos[0].VRAMTotalUsedMemory)
}
