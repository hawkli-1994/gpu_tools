package enflame

import (
	"fmt"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	_ "embed"
)

//go:embed testdata/efs.txt
var efs []byte

//go:embed testdata/efs-15.txt
var efs15 []byte

//go:embed testdata/efs17.txt
var efs17 []byte

func TestEnflameParse(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []gpu.GPUInfo
	}{
		{
			name:  "efs_old_format",
			input: efs,
			expected: []gpu.GPUInfo{
				{
					Num: 0, DeviceID: "0", CardVendor: "Enflame", CardModel: "Enflame GCU",
					TemperatureMemory: "39", TemperatureEdge: "39", TemperatureJunction: "39",
					VRAMTotalMemory:     fmt.Sprintf("%d", 42976*1024*1024),
					VRAMTotalUsedMemory: fmt.Sprintf("%d", 1129*1024*1024),
					GPUUse:              "0.0",
					PCIBus:              "0000:0c:00.0",
				},
				{
					Num: 1, DeviceID: "1", CardVendor: "Enflame", CardModel: "Enflame GCU",
					TemperatureMemory: "39", TemperatureEdge: "39", TemperatureJunction: "39",
					VRAMTotalMemory:     fmt.Sprintf("%d", 42976*1024*1024),
					VRAMTotalUsedMemory: fmt.Sprintf("%d", 1129*1024*1024),
					GPUUse:              "0.0",
					PCIBus:              "0000:0f:00.0",
				},
			},
		},
		{
			name:  "efs15_total_size_no_bar",
			input: efs15,
			expected: []gpu.GPUInfo{
				{
					Num: 0, DeviceID: "0", CardVendor: "Enflame", CardModel: "Enflame GCU",
					TemperatureMemory: "58", TemperatureEdge: "58", TemperatureJunction: "58",
					VRAMTotalMemory:     fmt.Sprintf("%d", 42976*1024*1024),
					VRAMTotalUsedMemory: fmt.Sprintf("%d", 24028*1024*1024),
					GPUUse:              "0.0",
					PCIBus:              "0000:0c:00.0",
				},
				{
					Num: 1, DeviceID: "1", CardVendor: "Enflame", CardModel: "Enflame GCU",
					TemperatureMemory: "59", TemperatureEdge: "59", TemperatureJunction: "59",
					VRAMTotalMemory:     fmt.Sprintf("%d", 42976*1024*1024),
					VRAMTotalUsedMemory: fmt.Sprintf("%d", 24028*1024*1024),
					GPUUse:              "0.0",
					PCIBus:              "0000:0f:00.0",
				},
			},
		},
		{
			name:  "efs17_total_size_with_bar",
			input: efs17,
			expected: []gpu.GPUInfo{
				{
					Num: 0, DeviceID: "0", CardVendor: "Enflame", CardModel: "Enflame GCU",
					TemperatureMemory: "76", TemperatureEdge: "76", TemperatureJunction: "76",
					VRAMTotalMemory:     fmt.Sprintf("%d", 42976*1024*1024),
					VRAMTotalUsedMemory: fmt.Sprintf("%d", 8684*1024*1024),
					GPUUse:              "0.0",
					PCIBus:              "0000:0c:00.0",
				},
				{
					Num: 1, DeviceID: "1", CardVendor: "Enflame", CardModel: "Enflame GCU",
					TemperatureMemory: "0", TemperatureEdge: "0", TemperatureJunction: "0",
					VRAMTotalMemory:     "0",
					VRAMTotalUsedMemory: "0",
					GPUUse:              "0",
					PCIBus:              "0000:0f:00.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &enflameSMICommand{}
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
				if got.PCIBus != exp.PCIBus {
					t.Errorf("GPU %d PCIBus: expected %s, got %s", i, exp.PCIBus, got.PCIBus)
				}
			}
		})
	}
}
