package huawei

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata/*.txt
var testdataFS embed.FS

func TestExtractNPUIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "single_npu",
			input: `+=======+
| NPU     Name |
| 2944    310P3 |
| 0       0     |
+=======+`,
			expected: []string{"2944"},
		},
		{
			name: "dual_npu_dedup",
			input: `+=======+
| 2944    310P3 |
| 0       0     |
+-------+
| 2944    310P3 |
| 1       1     |
+=======+
| 5678    910B  |
| 0       0     |
+=======+`,
			expected: []string{"2944", "5678"},
		},
		{
			name: "header_should_not_match",
			input: `| NPU     Name                  | Health          |
| Chip    Device                | Bus-Id          |
| 2944    310P3                 | OK              |`,
			expected: []string{"2944"},
		},
		{
			name:     "empty_output",
			input:    "",
			expected: nil,
		},
		{
			name: "no_number_in_first_field",
			input: `| abc     def   |
| ghi     jkl   |`,
			expected: nil,
		},
		{
			name:     "whitespace_variants",
			input:    "| 1234    310P3 |\n|\t5678\t910B\t|\n",
			expected: []string{"1234", "5678"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractNPUIDs([]byte(tt.input))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseChipSectionsForward(t *testing.T) {
	// common-style: Chip ID at the start of each section
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
		// MCU data should not be mixed into chip 1
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
	// usages/memory/product/health-style: Chip ID at the end of each section
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
		// MCU data is ignored in reverse mode (after last Chip ID)
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
	t.Run("full_board", func(t *testing.T) {
		input := `NPU ID                         : 2944
Product Name                   : IT21PD2G10
Model                          : NA
Manufacturer                   : Huawei
Serial Number                  : 2106030737ZERC003572
Software Version               : 25.5.1
Firmware Version               : 7.8.0.6.201
Compatibility                  : OK
Board ID                       : 0xb1
PCB ID                         : E
BOM ID                         : 3
PCIe Bus Info                  : 0000:0C:00.0
Slot ID                        : NA
Class ID                       : NA
PCI Vendor ID                  : 0x19E5
PCI Device ID                  : 0xD500
Subsystem Vendor ID            : 0x0200
Subsystem Device ID            : 0x0110
Chip Count                     : 2
Chip Fault                     : 0`
		got := parseBoardOutput([]byte(input))
		assert.Equal(t, "2106030737ZERC003572", got["Serial Number"])
		assert.Equal(t, "0000:0C:00.0", got["PCIe Bus Info"])
		assert.Equal(t, "Huawei", got["Manufacturer"])
	})

	t.Run("empty", func(t *testing.T) {
		got := parseBoardOutput([]byte(""))
		assert.Empty(t, got)
	})

	t.Run("no_colon_lines", func(t *testing.T) {
		got := parseBoardOutput([]byte("some random text\nwithout colons"))
		assert.Empty(t, got)
	})
}

func TestBuildGPUInfoList(t *testing.T) {
	t.Run("full_data_dual_chip", func(t *testing.T) {
		board := map[string]string{
			"Serial Number": "SN12345",
			"PCIe Bus Info": "0000:0C:00.0",
		}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "45", "Aicore Usage Rate(%)": "10"},
			"1": {"Temperature(C)": "47", "Aicore Usage Rate(%)": "20"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "44278", "DDR Usage Rate(%)": "2", "Aicore Usage Rate(%)": "10"},
			"1": {"DDR Capacity(MB)": "43693", "DDR Usage Rate(%)": "3", "Aicore Usage Rate(%)": "20"},
		}
		product := map[string]map[string]string{
			"0": {"Product Type": "Atlas 300I Duo"},
			"1": {"Product Type": "Atlas 300I Duo"},
		}
		power := map[string]string{"Power Dissipation(W)": "43.0"}

		infos, nextNum := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Len(t, infos, 2)
		assert.Equal(t, 2, nextNum)

		// Chip 0
		assert.Equal(t, 0, infos[0].Num)
		assert.Equal(t, "0", infos[0].DeviceID)
		assert.Equal(t, "Huawei", infos[0].CardVendor)
		assert.Equal(t, "Ascend", infos[0].CardSeries)
		assert.Equal(t, "Atlas 300I Duo", infos[0].CardModel)
		assert.Equal(t, "45", infos[0].TemperatureEdge)
		assert.Equal(t, "45", infos[0].TemperatureJunction)
		assert.Equal(t, "45", infos[0].TemperatureMemory)
		assert.Equal(t, "10", infos[0].GPUUse)
		assert.Equal(t, "SN12345", infos[0].SerialNumber)
		assert.Equal(t, "0000:0C:00.0", infos[0].PCIBus)
		assert.Equal(t, "43.0", infos[0].AverageGraphicsPackagePower)
		// VRAM: 44278 * 1024 * 1024 = 46428848128
		assert.Equal(t, "46428848128", infos[0].VRAMTotalMemory)
		// Used: 46428848128 * 2 / 100 = 928576962.04 -> 928576962
		assert.Equal(t, "928576962", infos[0].VRAMTotalUsedMemory)

		// Chip 1
		assert.Equal(t, 1, infos[1].Num)
		assert.Equal(t, "1", infos[1].DeviceID)
		assert.Equal(t, "47", infos[1].TemperatureEdge)
		assert.Equal(t, "20", infos[1].GPUUse)
		assert.Equal(t, "Atlas 300I Duo", infos[1].CardModel)
	})

	t.Run("missing_common_temperature_fallback_to_zero", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000", "DDR Usage Rate(%)": "50"},
		}
		product := map[string]map[string]string{}
		power := map[string]string{}

		infos, _ := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Len(t, infos, 1)
		assert.Equal(t, "0", infos[0].TemperatureEdge)
		assert.Equal(t, "0", infos[0].GPUUse)
	})

	t.Run("missing_usages_vram_zero", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "50", "Aicore Usage Rate(%)": "30"},
		}
		usages := map[string]map[string]string{}
		product := map[string]map[string]string{}
		power := map[string]string{}

		infos, _ := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Len(t, infos, 1)
		assert.Equal(t, "50", infos[0].TemperatureEdge)
		assert.Equal(t, "30", infos[0].GPUUse)
		assert.Equal(t, "0", infos[0].VRAMTotalMemory)
		assert.Equal(t, "0", infos[0].VRAMTotalUsedMemory)
	})

	t.Run("gpu_use_fallback_from_common", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{
			"0": {"Aicore Usage Rate(%)": "25"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000"}, // no Aicore here
		}
		product := map[string]map[string]string{}
		power := map[string]string{}

		infos, _ := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Equal(t, "25", infos[0].GPUUse)
	})

	t.Run("power_na_becomes_zero", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "40"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000"},
		}
		product := map[string]map[string]string{}
		power := map[string]string{"Power Dissipation(W)": "NA"}

		infos, _ := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Equal(t, "0", infos[0].AverageGraphicsPackagePower)
	})

	t.Run("power_empty_becomes_zero", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "40"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000"},
		}
		product := map[string]map[string]string{}
		power := map[string]string{}

		infos, _ := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Equal(t, "0", infos[0].AverageGraphicsPackagePower)
	})

	t.Run("missing_product_model_empty", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "40"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000"},
		}
		product := map[string]map[string]string{}
		power := map[string]string{}

		infos, _ := buildGPUInfoList("2944", board, common, usages, product, power, 0)
		assert.Equal(t, "", infos[0].CardModel)
	})

	t.Run("single_chip", func(t *testing.T) {
		board := map[string]string{"PCIe Bus Info": "0000:01:00.0"}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "55"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "22000", "DDR Usage Rate(%)": "10"},
		}
		product := map[string]map[string]string{
			"0": {"Product Type": "Atlas 300I Pro"},
		}
		power := map[string]string{"Power Dissipation(W)": "65.5"}

		infos, nextNum := buildGPUInfoList("1234", board, common, usages, product, power, 0)
		assert.Len(t, infos, 1)
		assert.Equal(t, 1, nextNum)
		assert.Equal(t, "0", infos[0].DeviceID)
		assert.Equal(t, "55", infos[0].TemperatureEdge)
		assert.Equal(t, "65.5", infos[0].AverageGraphicsPackagePower)
	})

	t.Run("no_chips_returns_empty", func(t *testing.T) {
		infos, nextNum := buildGPUInfoList("2944", nil, nil, nil, nil, nil, 0)
		assert.Empty(t, infos)
		assert.Equal(t, 0, nextNum)
	})

	t.Run("global_num_increments", func(t *testing.T) {
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "40"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000"},
		}

		infos1, num1 := buildGPUInfoList("2944", nil, common, usages, nil, nil, 5)
		assert.Len(t, infos1, 1)
		assert.Equal(t, 5, infos1[0].Num)
		assert.Equal(t, 6, num1)
	})

	t.Run("missing_board_fields", func(t *testing.T) {
		board := map[string]string{}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "40"},
		}
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000"},
		}

		infos, _ := buildGPUInfoList("2944", board, common, usages, nil, nil, 0)
		assert.Equal(t, "", infos[0].SerialNumber)
		assert.Equal(t, "", infos[0].PCIBus)
	})

	t.Run("vram_calculation_rounding", func(t *testing.T) {
		usages := map[string]map[string]string{
			"0": {"DDR Capacity(MB)": "10000", "DDR Usage Rate(%)": "33.3"},
		}
		common := map[string]map[string]string{
			"0": {"Temperature(C)": "40"},
		}

		infos, _ := buildGPUInfoList("2944", nil, common, usages, nil, nil, 0)
		// 10000 * 1024 * 1024 = 10485760000
		assert.Equal(t, "10485760000", infos[0].VRAMTotalMemory)
		// 10485760000 * 33.3 / 100 = 3491758080
		assert.Equal(t, "3491758080", infos[0].VRAMTotalUsedMemory)
	})
}

func TestNpuSMICommandVendor(t *testing.T) {
	cmd := &npuSMICommand{}
	assert.Equal(t, "Huawei", cmd.Vendor())
}

func TestAvailableWithCachedPath(t *testing.T) {
	cmd := &npuSMICommand{smiPath: "/some/path/npu-smi"}
	assert.True(t, cmd.Available())
}

func TestNewReturnsCommand(t *testing.T) {
	cmd := New()
	assert.NotNil(t, cmd)
	assert.Equal(t, "Huawei", cmd.Vendor())
}

func TestIntegrationDualChipFromFixtures(t *testing.T) {
	fixtureNPUInfo, _ := testdataFS.ReadFile("testdata/npu_info.txt")
	fixtureNPUBoard, _ := testdataFS.ReadFile("testdata/npu_board.txt")
	fixtureNPUCommon, _ := testdataFS.ReadFile("testdata/npu_common.txt")
	fixtureNPUUsages, _ := testdataFS.ReadFile("testdata/npu_usages.txt")
	fixtureNPUProduct, _ := testdataFS.ReadFile("testdata/npu_product.txt")
	fixtureNPUPower, _ := testdataFS.ReadFile("testdata/npu_power.txt")

	ids, err := extractNPUIDs(fixtureNPUInfo)
	assert.NoError(t, err)
	assert.Equal(t, []string{"2944"}, ids)

	board := parseBoardOutput(fixtureNPUBoard)
	common := parseChipSections(fixtureNPUCommon)
	usages := parseChipSections(fixtureNPUUsages)
	product := parseChipSections(fixtureNPUProduct)
	power := parseBoardOutput(fixtureNPUPower)

	infos, nextNum := buildGPUInfoList("2944", board, common, usages, product, power, 0)
	assert.Equal(t, 2, nextNum)
	assert.Len(t, infos, 2)

	// Chip 0
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "0", infos[0].DeviceID)
	assert.Equal(t, "Huawei", infos[0].CardVendor)
	assert.Equal(t, "Ascend", infos[0].CardSeries)
	assert.Equal(t, "Atlas 300I Duo", infos[0].CardModel)
	assert.Equal(t, "45", infos[0].TemperatureEdge)
	assert.Equal(t, "0", infos[0].GPUUse)
	assert.Equal(t, "2106030737ZERC003572", infos[0].SerialNumber)
	assert.Equal(t, "0000:0C:00.0", infos[0].PCIBus)
	assert.Equal(t, "42.9", infos[0].AverageGraphicsPackagePower)
	assert.Equal(t, "46428848128", infos[0].VRAMTotalMemory)
	assert.Equal(t, "928576962", infos[0].VRAMTotalUsedMemory)

	// Chip 1
	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, "1", infos[1].DeviceID)
	assert.Equal(t, "47", infos[1].TemperatureEdge)
	assert.Equal(t, "0", infos[1].GPUUse)
	assert.Equal(t, "45815431168", infos[1].VRAMTotalMemory)
	assert.Equal(t, "1374462935", infos[1].VRAMTotalUsedMemory)
}

func TestIntegrationSingleChipFromFixtures(t *testing.T) {
	fixtureSingleInfo, _ := testdataFS.ReadFile("testdata/npu_single_chip.txt")
	fixtureSingleBoard, _ := testdataFS.ReadFile("testdata/npu_single_board.txt")
	fixtureSingleCommon, _ := testdataFS.ReadFile("testdata/npu_single_common.txt")
	fixtureSingleUsages, _ := testdataFS.ReadFile("testdata/npu_single_usages.txt")
	fixtureSingleProduct, _ := testdataFS.ReadFile("testdata/npu_single_product.txt")
	fixtureSinglePower, _ := testdataFS.ReadFile("testdata/npu_single_power.txt")

	ids, err := extractNPUIDs(fixtureSingleInfo)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1234"}, ids)

	board := parseBoardOutput(fixtureSingleBoard)
	common := parseChipSections(fixtureSingleCommon)
	usages := parseChipSections(fixtureSingleUsages)
	product := parseChipSections(fixtureSingleProduct)
	power := parseBoardOutput(fixtureSinglePower)

	infos, nextNum := buildGPUInfoList("1234", board, common, usages, product, power, 0)
	assert.Equal(t, 1, nextNum)
	assert.Len(t, infos, 1)

	assert.Equal(t, "0", infos[0].DeviceID)
	assert.Equal(t, "Atlas 300I Pro", infos[0].CardModel)
	assert.Equal(t, "55", infos[0].TemperatureEdge)
	assert.Equal(t, "10", infos[0].GPUUse)
	assert.Equal(t, "SN987654321", infos[0].SerialNumber)
	assert.Equal(t, "0000:01:00.0", infos[0].PCIBus)
	assert.Equal(t, "65.5", infos[0].AverageGraphicsPackagePower)
	// 22000 * 1024 * 1024 = 23068672000
	assert.Equal(t, "23068672000", infos[0].VRAMTotalMemory)
	// 23068672000 * 10 / 100 = 2306867200
	assert.Equal(t, "2306867200", infos[0].VRAMTotalUsedMemory)
}
