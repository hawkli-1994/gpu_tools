package ix

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/ixsmi.xml
var ixdata string

func TestParseIXSMI(t *testing.T) {
	path := filepath.Join("testdata", "ixsmi.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read xml: %v", err)
	}
	info, err := ParseIXSMI(string(data))
	if err != nil {
		t.Fatalf("parseIXSMI error: %v", err)
	}
	if len(info.GPUInfos) != 2 {
		t.Fatalf("expected 2 gpus, got %d", len(info.GPUInfos))
	}
	gpu0 := info.GPUInfos[0]
	if gpu0.DeviceID != "00000000:0C:00.0" {
		t.Errorf("gpu0.DeviceID = %s", gpu0.DeviceID)
	}
	if gpu0.SerialNumber != "23490256585496" {
		t.Errorf("gpu0.SerialNumber = %s", gpu0.SerialNumber)
	}
	if gpu0.VRAMTotalMemory != "34359738368" {
		t.Errorf("gpu0.VRAMTotalMemory = %s", gpu0.VRAMTotalMemory)
	}
	if gpu0.VRAMTotalUsedMemory != "29129441280" {
		t.Errorf("gpu0.VRAMTotalUsedMemory = %s", gpu0.VRAMTotalUsedMemory)
	}
	if gpu0.TemperatureEdge != "44" {
		t.Errorf("gpu0.TemperatureEdge = %s", gpu0.TemperatureEdge)
	}
	if gpu0.GPUUse != "0" {
		t.Errorf("gpu0.GPUUse = %s", gpu0.GPUUse)
	}
	if gpu0.PCIBus != "0000:0C:00.0" {
		t.Errorf("gpu0.PCIBus = %s", gpu0.PCIBus)
	}
	gpu1 := info.GPUInfos[1]
	if gpu1.DeviceID != "00000000:0F:00.0" {
		t.Errorf("gpu1.DeviceID = %s", gpu1.DeviceID)
	}
	if gpu1.SerialNumber != "23490258505496" {
		t.Errorf("gpu1.SerialNumber = %s", gpu1.SerialNumber)
	}
	if gpu1.VRAMTotalMemory != "34359738368" {
		t.Errorf("gpu1.VRAMTotalMemory = %s", gpu1.VRAMTotalMemory)
	}
	if gpu1.VRAMTotalUsedMemory != "29062332416" {
		t.Errorf("gpu1.VRAMTotalUsedMemory = %s", gpu1.VRAMTotalUsedMemory)
	}
	if gpu1.TemperatureEdge != "44" {
		t.Errorf("gpu1.TemperatureEdge = %s", gpu1.TemperatureEdge)
	}
	if gpu1.GPUUse != "0" {
		t.Errorf("gpu1.GPUUse = %s", gpu1.GPUUse)
	}
	if gpu1.PCIBus != "0000:0F:00.0" {
		t.Errorf("gpu1.PCIBus = %s", gpu1.PCIBus)
	}
}
