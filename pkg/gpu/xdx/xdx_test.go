package xdx

import (
	"fmt"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	_ "embed"
)

//go:embed testdata/output.txt
var output []byte

func TestXDXParse(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []gpu.GPUInfo
	}{
		{
			name:  "xdxsmi_output",
			input: output,
			expected: []gpu.GPUInfo{
				{
					Num:                 0,
					DeviceID:            "0",
					CardVendor:          "XDX",
					CardModel:           "R1900",
					CardSeries:          "PANGU_A0",
					TemperatureMemory:   "35.0",
					TemperatureEdge:     "35.0",
					TemperatureJunction: "35.0",
					VRAMTotalMemory:     fmt.Sprintf("%d", 16384*1024*1024),
					VRAMTotalUsedMemory: "0",
					GPUUse:              "0",
					AverageGraphicsPackagePower: "41.0",
					PCIBus:              "0000:19:00.0",
				},
				{
					Num:                 1,
					DeviceID:            "1",
					CardVendor:          "XDX",
					CardModel:           "R1900",
					CardSeries:          "PANGU_A0",
					TemperatureMemory:   "35.0",
					TemperatureEdge:     "35.0",
					TemperatureJunction: "35.0",
					VRAMTotalMemory:     fmt.Sprintf("%d", 16384*1024*1024),
					VRAMTotalUsedMemory: "0",
					GPUUse:              "0",
					AverageGraphicsPackagePower: "41.0",
					PCIBus:              "0000:1a:00.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &xdxSMICommand{}
			gpuInfoList, err := e.parse(tt.input)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if len(gpuInfoList.GPUInfos) != len(tt.expected) {
				t.Fatalf("expected %d GPUs, got %d", len(tt.expected), len(gpuInfoList.GPUInfos))
			}
			for i, exp := range tt.expected {
				got := gpuInfoList.GPUInfos[i]
				if got.Num != exp.Num {
					t.Errorf("GPU %d Num: expected %d, got %d", i, exp.Num, got.Num)
				}
				if got.DeviceID != exp.DeviceID {
					t.Errorf("GPU %d DeviceID: expected %s, got %s", i, exp.DeviceID, got.DeviceID)
				}
				if got.CardVendor != exp.CardVendor {
					t.Errorf("GPU %d CardVendor: expected %s, got %s", i, exp.CardVendor, got.CardVendor)
				}
				if got.CardModel != exp.CardModel {
					t.Errorf("GPU %d CardModel: expected %s, got %s", i, exp.CardModel, got.CardModel)
				}
				if got.CardSeries != exp.CardSeries {
					t.Errorf("GPU %d CardSeries: expected %s, got %s", i, exp.CardSeries, got.CardSeries)
				}
				if got.TemperatureMemory != exp.TemperatureMemory {
					t.Errorf("GPU %d TemperatureMemory: expected %s, got %s", i, exp.TemperatureMemory, got.TemperatureMemory)
				}
				if got.TemperatureEdge != exp.TemperatureEdge {
					t.Errorf("GPU %d TemperatureEdge: expected %s, got %s", i, exp.TemperatureEdge, got.TemperatureEdge)
				}
				if got.TemperatureJunction != exp.TemperatureJunction {
					t.Errorf("GPU %d TemperatureJunction: expected %s, got %s", i, exp.TemperatureJunction, got.TemperatureJunction)
				}
				if got.VRAMTotalMemory != exp.VRAMTotalMemory {
					t.Errorf("GPU %d VRAMTotalMemory: expected %s, got %s", i, exp.VRAMTotalMemory, got.VRAMTotalMemory)
				}
				if got.VRAMTotalUsedMemory != exp.VRAMTotalUsedMemory {
					t.Errorf("GPU %d VRAMTotalUsedMemory: expected %s, got %s", i, exp.VRAMTotalUsedMemory, got.VRAMTotalUsedMemory)
				}
				if got.GPUUse != exp.GPUUse {
					t.Errorf("GPU %d GPUUse: expected %s, got %s", i, exp.GPUUse, got.GPUUse)
				}
				if got.AverageGraphicsPackagePower != exp.AverageGraphicsPackagePower {
					t.Errorf("GPU %d AverageGraphicsPackagePower: expected %s, got %s", i, exp.AverageGraphicsPackagePower, got.AverageGraphicsPackagePower)
				}
				if got.PCIBus != exp.PCIBus {
					t.Errorf("GPU %d PCIBus: expected %s, got %s", i, exp.PCIBus, got.PCIBus)
				}
			}
		})
	}
}

func TestXDXParseEmptyData(t *testing.T) {
	e := &xdxSMICommand{}
	gpuInfoList, err := e.parse([]byte(""))
	if err != nil {
		t.Fatalf("failed to parse empty data: %v", err)
	}
	if len(gpuInfoList.GPUInfos) != 0 {
		t.Errorf("expected 0 GPUs, got %d", len(gpuInfoList.GPUInfos))
	}
}
