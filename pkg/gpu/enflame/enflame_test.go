package enflame

import (
	"fmt"
	"strconv"
	"testing"

	_ "embed"
)

//go:embed testdata/efs.txt
var efs []byte

func TestEnflameParse(t *testing.T) {
	enflame := &enflameSMICommand{}
	gpuInfoList, err := enflame.parse(efs)
	if err != nil {
		t.Fatalf("failed to parse efsmi output: %v", err)
	}

	// fmt.Printf("gpuInfoList: %+v\n", gpuInfoList)
	for i, gpuInfo := range gpuInfoList.GPUInfos {
		fmt.Printf("gpuInfo: %+v\n", gpuInfo)
		if i == 0 {
			if gpuInfo.DeviceID != "0" {
				t.Fatalf("expected device ID 0, got %s", gpuInfo.DeviceID)
			}
			if gpuInfo.CardVendor != "Enflame" {
				t.Fatalf("expected card vendor Enflame, got %s", gpuInfo.CardVendor)
			}
			if gpuInfo.CardModel != "Enflame GCU" {
				t.Fatalf("expected card model Enflame GCU, got %s", gpuInfo.CardModel)
			}
			memBytes, err := strconv.ParseUint(gpuInfo.VRAMTotalMemory, 10, 64)
			if err != nil {
				t.Fatalf("failed to parse VRAM total memory: %v", err)
			}
			if memBytes != 42976*1024*1024 {
				t.Fatalf("expected VRAM total memory 42976 MiB, got %s", gpuInfo.VRAMTotalMemory)
			}
			memUsedBytes, err := strconv.ParseUint(gpuInfo.VRAMTotalUsedMemory, 10, 64)
			if err != nil {
				t.Fatalf("failed to parse VRAM total used memory: %v", err)
			}
			if memUsedBytes != 1129*1024*1024 {
				t.Fatalf("expected VRAM total used memory 1129 MiB, got %s", gpuInfo.VRAMTotalUsedMemory)
			}
			if gpuInfo.TemperatureMemory != "39" {
				t.Fatalf("expected temperature memory 39 C, got %s", gpuInfo.TemperatureMemory)
			}
			if gpuInfo.Num != 0 {
				t.Fatalf("expected num 0, got %d", gpuInfo.Num)
			}
			if gpuInfo.PCIBus != "0000:0c:00.0" {
				t.Fatalf("expected PCI bus 0000:0c:00.0, got %s", gpuInfo.PCIBus)
			}
		}
		if i == 1 {
			if gpuInfo.PCIBus != "0000:0f:00.0" {
				t.Fatalf("expected PCI bus 0000:0f:00.0, got %s", gpuInfo.PCIBus)
			}
		}
		i++
	}
}
