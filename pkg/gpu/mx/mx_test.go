package mx

import (
	"testing"
	_ "embed"
)

//go:embed testdata/output.txt
var output string

//go:embed testdata/meta_2.2.12.txt
var output212 string

//go:embed testdata/debug-mx.txt
var output234 string

func TestParseMxOutput(t *testing.T) {
	gpuList, err := parseMxOutput(output)
	if err != nil {
		t.Fatalf("parseMxOutput failed: %v", err)
	}

	if len(gpuList.GPUInfos) != 1 {
		t.Errorf("Expected 1 GPU, got %d", len(gpuList.GPUInfos))
		return
	}

	if gpuList.GPUInfos[0].Num != 0 {
		t.Errorf("Expected GPU number 0, got %d", gpuList.GPUInfos[0].Num)
	}

	if gpuList.GPUInfos[0].CardModel != "MXN260" {
		t.Errorf("Expected CardModel MXN260, got %s", gpuList.GPUInfos[0].CardModel)
	}

	if gpuList.GPUInfos[0].TemperatureEdge != "44.00" {
		t.Errorf("Expected TemperatureEdge 44.00, got %s", gpuList.GPUInfos[0].TemperatureEdge)
	}

	if gpuList.GPUInfos[0].VRAMTotalMemory != "68719476736" {
		t.Errorf("Expected VRAMTotalMemory 68719476736, got %s", gpuList.GPUInfos[0].VRAMTotalMemory)
	}

	if gpuList.GPUInfos[0].VRAMTotalUsedMemory != "62684897280" {
		t.Errorf("Expected VRAMTotalUsedMemory 62684897280, got %s", gpuList.GPUInfos[0].VRAMTotalUsedMemory)
	}

	if gpuList.GPUInfos[0].GPUUse != "0" {
		t.Errorf("Expected GPUUse 0, got %s", gpuList.GPUInfos[0].GPUUse)
	}

	if gpuList.GPUInfos[0].PCIBus != "0000:0f:00.0" {
		t.Errorf("Expected PCIBus 0000:0f:00.0, got %s", gpuList.GPUInfos[0].PCIBus)
	}
}

func TestParseMxOutput212(t *testing.T) {
	gpuList, err := parseMxOutput(output212)
	if err != nil {
		t.Fatalf("parseMxOutput failed: %v", err)
	}

	if len(gpuList.GPUInfos) != 2 {
		t.Errorf("Expected 2 GPUs, got %d", len(gpuList.GPUInfos))
		return
	}

	// GPU#0
	if gpuList.GPUInfos[0].Num != 0 {
		t.Errorf("Expected GPU#0 Num 0, got %d", gpuList.GPUInfos[0].Num)
	}
	if gpuList.GPUInfos[0].CardModel != "MXN260" {
		t.Errorf("Expected GPU#0 CardModel MXN260, got %s", gpuList.GPUInfos[0].CardModel)
	}
	if gpuList.GPUInfos[0].TemperatureEdge != "38.00" {
		t.Errorf("Expected GPU#0 TemperatureEdge 38.00, got %s", gpuList.GPUInfos[0].TemperatureEdge)
	}
	if gpuList.GPUInfos[0].VRAMTotalMemory != "68719476736" {
		t.Errorf("Expected GPU#0 VRAMTotalMemory 68719476736, got %s", gpuList.GPUInfos[0].VRAMTotalMemory)
	}
	if gpuList.GPUInfos[0].VRAMTotalUsedMemory != "62379499520" {
		t.Errorf("Expected GPU#0 VRAMTotalUsedMemory 62379499520, got %s", gpuList.GPUInfos[0].VRAMTotalUsedMemory)
	}
	if gpuList.GPUInfos[0].GPUUse != "0" {
		t.Errorf("Expected GPU#0 GPUUse 0, got %s", gpuList.GPUInfos[0].GPUUse)
	}
	if gpuList.GPUInfos[0].PCIBus != "0000:19:00.0" {
		t.Errorf("Expected GPU#0 PCIBus 0000:19:00.0, got %s", gpuList.GPUInfos[0].PCIBus)
	}

	// GPU#1
	if gpuList.GPUInfos[1].Num != 1 {
		t.Errorf("Expected GPU#1 Num 1, got %d", gpuList.GPUInfos[1].Num)
	}
	if gpuList.GPUInfos[1].CardModel != "MXN260" {
		t.Errorf("Expected GPU#1 CardModel MXN260, got %s", gpuList.GPUInfos[1].CardModel)
	}
	if gpuList.GPUInfos[1].TemperatureEdge != "37.50" {
		t.Errorf("Expected GPU#1 TemperatureEdge 37.50, got %s", gpuList.GPUInfos[1].TemperatureEdge)
	}
	if gpuList.GPUInfos[1].VRAMTotalMemory != "68719476736" {
		t.Errorf("Expected GPU#1 VRAMTotalMemory 68719476736, got %s", gpuList.GPUInfos[1].VRAMTotalMemory)
	}
	if gpuList.GPUInfos[1].VRAMTotalUsedMemory != "2467725312" {
		t.Errorf("Expected GPU#1 VRAMTotalUsedMemory 2467725312, got %s", gpuList.GPUInfos[1].VRAMTotalUsedMemory)
	}
	if gpuList.GPUInfos[1].GPUUse != "0" {
		t.Errorf("Expected GPU#1 GPUUse 0, got %s", gpuList.GPUInfos[1].GPUUse)
	}
	if gpuList.GPUInfos[1].PCIBus != "0000:1a:00.0" {
		t.Errorf("Expected GPU#1 PCIBus 0000:1a:00.0, got %s", gpuList.GPUInfos[1].PCIBus)
	}
}

func TestParseMxOutput234(t *testing.T) {
	gpuList, err := parseMxOutput(output234)
	if err != nil {
		t.Fatalf("parseMxOutput failed: %v", err)
	}

	if len(gpuList.GPUInfos) != 2 {
		t.Errorf("Expected 2 GPUs, got %d", len(gpuList.GPUInfos))
		return
	}

	// GPU#0
	if gpuList.GPUInfos[0].Num != 0 {
		t.Errorf("Expected GPU#0 Num 0, got %d", gpuList.GPUInfos[0].Num)
	}
	if gpuList.GPUInfos[0].CardModel != "MXN260" {
		t.Errorf("Expected GPU#0 CardModel MXN260, got %s", gpuList.GPUInfos[0].CardModel)
	}
	if gpuList.GPUInfos[0].TemperatureEdge != "45.75" {
		t.Errorf("Expected GPU#0 TemperatureEdge 45.75, got %s", gpuList.GPUInfos[0].TemperatureEdge)
	}
	if gpuList.GPUInfos[0].VRAMTotalMemory != "68719476736" {
		t.Errorf("Expected GPU#0 VRAMTotalMemory 68719476736, got %s", gpuList.GPUInfos[0].VRAMTotalMemory)
	}
	if gpuList.GPUInfos[0].VRAMTotalUsedMemory != "700903424" {
		t.Errorf("Expected GPU#0 VRAMTotalUsedMemory 700903424, got %s", gpuList.GPUInfos[0].VRAMTotalUsedMemory)
	}
	if gpuList.GPUInfos[0].GPUUse != "0" {
		t.Errorf("Expected GPU#0 GPUUse 0, got %s", gpuList.GPUInfos[0].GPUUse)
	}
	if gpuList.GPUInfos[0].PCIBus != "0000:0c:00.0" {
		t.Errorf("Expected GPU#0 PCIBus 0000:0c:00.0, got %s", gpuList.GPUInfos[0].PCIBus)
	}

	// GPU#1
	if gpuList.GPUInfos[1].Num != 1 {
		t.Errorf("Expected GPU#1 Num 1, got %d", gpuList.GPUInfos[1].Num)
	}
	if gpuList.GPUInfos[1].TemperatureEdge != "44.25" {
		t.Errorf("Expected GPU#1 TemperatureEdge 44.25, got %s", gpuList.GPUInfos[1].TemperatureEdge)
	}
	if gpuList.GPUInfos[1].PCIBus != "0000:0f:00.0" {
		t.Errorf("Expected GPU#1 PCIBus 0000:0f:00.0, got %s", gpuList.GPUInfos[1].PCIBus)
	}
}

func TestParseMxOutputEmpty(t *testing.T) {
	// 测试空输出
	gpuList, err := parseMxOutput("")
	if err != nil {
		t.Fatalf("parseMxOutput failed with empty input: %v", err)
	}

	if len(gpuList.GPUInfos) != 0 {
		t.Errorf("Expected 0 GPUs for empty input, got %d", len(gpuList.GPUInfos))
	}
}

func TestParseMxOutputInvalid(t *testing.T) {
	// 测试无效输出
	invalidOutput := "invalid mx-smi output"
	gpuList, err := parseMxOutput(invalidOutput)
	if err != nil {
		t.Fatalf("parseMxOutput should not fail with invalid input: %v", err)
	}

	if len(gpuList.GPUInfos) != 0 {
		t.Errorf("Expected 0 GPUs for invalid input, got %d", len(gpuList.GPUInfos))
	}
}