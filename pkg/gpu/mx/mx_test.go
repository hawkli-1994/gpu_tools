package mx

import (
	"testing"
	_ "embed"
)

//go:embed testdata/output.txt
var output string

func TestMX(t *testing.T) {
	// TODO: write tests
}

func TestParseMxOutput(t *testing.T) {
	// 测试解析函数
	gpuList, err := parseMxOutput(output)
	if err != nil {
		t.Fatalf("parseMxOutput failed: %v", err)
	}

	// 验证解析结果
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

	if gpuList.GPUInfos[0].VRAMTotalMemory != "68719476736" { // 67108864 KB = 68719476736 bytes
		t.Errorf("Expected VRAMTotalMemory 68719476736, got %s", gpuList.GPUInfos[0].VRAMTotalMemory)
	}

	if gpuList.GPUInfos[0].VRAMTotalUsedMemory != "62684897280" { // 61215720 KB = 62684897280 bytes
		t.Errorf("Expected VRAMTotalUsedMemory 62684897280, got %s", gpuList.GPUInfos[0].VRAMTotalUsedMemory)
	}

	if gpuList.GPUInfos[0].GPUUse != "0" {
		t.Errorf("Expected GPUUse 0, got %s", gpuList.GPUInfos[0].GPUUse)
	}

	if gpuList.GPUInfos[0].PCIBus != "0000:0f:00.0" {
		t.Errorf("Expected PCIBus 0000:0f:00.0, got %s", gpuList.GPUInfos[0].PCIBus)
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