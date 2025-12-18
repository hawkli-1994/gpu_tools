package dl

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDLSMIOutput(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "dlsmi_output.xml"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	infoList, err := parseDLSMIOutput(data)
	if err != nil {
		t.Fatalf("parseDLSMIOutput returned error: %v", err)
	}

	if len(infoList.GPUInfos) != 8 {
		t.Fatalf("expected 8 GPUs, got %d", len(infoList.GPUInfos))
	}

	first := infoList.GPUInfos[0]
	if first.Num != 0 {
		t.Fatalf("expected first GPU num 0, got %d", first.Num)
	}
	if first.DeviceID != "0x00061E27" {
		t.Fatalf("expected device ID 0x00061E27, got %s", first.DeviceID)
	}
	if first.DeviceRev != "0.3.16" {
		t.Fatalf("expected firmware version 0.3.16, got %s", first.DeviceRev)
	}
	if first.CardVendor != "Goldwasser" {
		t.Fatalf("expected vendor Goldwasser, got %s", first.CardVendor)
	}
	if first.CardModel != "KS38 QUAD-3" {
		t.Fatalf("expected card model KS38 QUAD-3, got %s", first.CardModel)
	}
	if first.CardSeries != "DLIv2" {
		t.Fatalf("expected card series DLIv2, got %s", first.CardSeries)
	}
	if first.CardSKU != "GDF00189C02" {
		t.Fatalf("expected SKU GDF00189C02, got %s", first.CardSKU)
	}
	if first.PCIBus != "00000000:1E:00.0" {
		t.Fatalf("expected PCI bus 00000000:1E:00.0, got %s", first.PCIBus)
	}
	if first.SerialNumber != "GDF00189C01DE25300141" {
		t.Fatalf("expected serial number GDF00189C01DE25300141, got %s", first.SerialNumber)
	}

	expectedTotal := fmt.Sprintf("%d", 32768*1024*1024)
	if first.VRAMTotalMemory != expectedTotal {
		t.Fatalf("expected total memory %s, got %s", expectedTotal, first.VRAMTotalMemory)
	}
	expectedUsed := fmt.Sprintf("%d", 293*1024*1024)
	if first.VRAMTotalUsedMemory != expectedUsed {
		t.Fatalf("expected used memory %s, got %s", expectedUsed, first.VRAMTotalUsedMemory)
	}

	if first.GPUUse != "0" {
		t.Fatalf("expected GPU utilization 0, got %s", first.GPUUse)
	}
	if first.AverageGraphicsPackagePower != "6.29" {
		t.Fatalf("expected power draw 6.29, got %s", first.AverageGraphicsPackagePower)
	}
	if first.TemperatureEdge != "53" {
		t.Fatalf("expected edge temperature 53, got %s", first.TemperatureEdge)
	}
	if first.TemperatureJunction != "106" {
		t.Fatalf("expected junction temperature 106, got %s", first.TemperatureJunction)
	}
	if first.TemperatureMemory != "53" {
		t.Fatalf("expected memory temperature 53, got %s", first.TemperatureMemory)
	}

	second := infoList.GPUInfos[1]
	if second.Num != 1 {
		t.Fatalf("expected second GPU num 1, got %d", second.Num)
	}
	if second.PCIBus != "00000000:1F:00.0" {
		t.Fatalf("expected second PCI bus 00000000:1F:00.0, got %s", second.PCIBus)
	}
	if second.CardModel != "KS38 QUAD-2" {
		t.Fatalf("expected second card model KS38 QUAD-2, got %s", second.CardModel)
	}

	last := infoList.GPUInfos[7]
	if last.Num != 7 {
		t.Fatalf("expected last GPU num 7, got %d", last.Num)
	}
	if last.PCIBus != "00000000:30:00.0" {
		t.Fatalf("expected last PCI bus 00000000:30:00.0, got %s", last.PCIBus)
	}
	if last.CardModel != "KS38 QUAD-1" {
		t.Fatalf("expected last card model KS38 QUAD-1, got %s", last.CardModel)
	}
}
