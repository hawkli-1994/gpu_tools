package ix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupIxsmmiEnv(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	// 创建假的 ixsmi 文件
	ixsmiPath := filepath.Join(binDir, "ixsmi")
	if err := os.WriteFile(ixsmiPath, []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatalf("failed to create ixsmi: %v", err)
	}

	// 保存原始环境变量
	origPath := os.Getenv("PATH")
	origLibPath := os.Getenv("LD_LIBRARY_PATH")
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("LD_LIBRARY_PATH", origLibPath)
	}()

	// 调用 setupIxsmmiEnv
	setupIxsmmiEnv(ixsmiPath)

	// 验证 PATH
	newPath := os.Getenv("PATH")
	if !strings.Contains(newPath, binDir) {
		t.Errorf("PATH should contain %s, got %s", binDir, newPath)
	}
	if !strings.HasPrefix(newPath, binDir) {
		t.Errorf("PATH should start with %s, got %s", binDir, newPath)
	}

	// 验证 LD_LIBRARY_PATH
	newLibPath := os.Getenv("LD_LIBRARY_PATH")
	expectedLibDir := filepath.Join(tmpDir, "lib")
	expectedLib64Dir := filepath.Join(tmpDir, "lib64")
	if !strings.Contains(newLibPath, expectedLibDir) {
		t.Errorf("LD_LIBRARY_PATH should contain %s, got %s", expectedLibDir, newLibPath)
	}
	if !strings.Contains(newLibPath, expectedLib64Dir) {
		t.Errorf("LD_LIBRARY_PATH should contain %s, got %s", expectedLib64Dir, newLibPath)
	}
}

func TestSetupIxsmmiEnvPreservesOldValues(t *testing.T) {
	// 设置已有的环境变量
	os.Setenv("PATH", "/usr/bin")
	os.Setenv("LD_LIBRARY_PATH", "/usr/lib")

	ixsmiPath := "/usr/local/corex-4.4.0/bin/ixsmi"

	setupIxsmmiEnv(ixsmiPath)

	// 验证原有值被保留
	newPath := os.Getenv("PATH")
	if !strings.Contains(newPath, "/usr/bin") {
		t.Errorf("PATH should preserve old value /usr/bin, got %s", newPath)
	}

	newLibPath := os.Getenv("LD_LIBRARY_PATH")
	if !strings.Contains(newLibPath, "/usr/lib") {
		t.Errorf("LD_LIBRARY_PATH should preserve old value /usr/lib, got %s", newLibPath)
	}
}

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
