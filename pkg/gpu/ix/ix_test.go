package ix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envValue(t *testing.T, env []string, key string) string {
	t.Helper()
	prefix := key + "="
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, prefix); ok {
			return v
		}
	}
	return ""
}

func TestIxsmmiEnv(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	ixsmiPath := filepath.Join(binDir, "ixsmi")
	if err := os.WriteFile(ixsmiPath, []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatalf("failed to create ixsmi: %v", err)
	}

	env := ixsmiEnv(ixsmiPath)

	// 验证 PATH 前置了 corex bin 目录，且保留原值
	newPath := envValue(t, env, "PATH")
	if !strings.HasPrefix(newPath, binDir+":") {
		t.Errorf("PATH should start with %s, got %s", binDir, newPath)
	}
	if !strings.Contains(newPath, os.Getenv("PATH")) {
		t.Errorf("PATH should preserve old value %s, got %s", os.Getenv("PATH"), newPath)
	}

	// 验证 LD_LIBRARY_PATH 包含 corex lib 目录，且保留原值
	newLibPath := envValue(t, env, "LD_LIBRARY_PATH")
	for _, dir := range []string{filepath.Join(tmpDir, "lib"), filepath.Join(tmpDir, "lib64")} {
		if !strings.Contains(newLibPath, dir) {
			t.Errorf("LD_LIBRARY_PATH should contain %s, got %s", dir, newLibPath)
		}
	}
	if old := os.Getenv("LD_LIBRARY_PATH"); old != "" && !strings.Contains(newLibPath, old) {
		t.Errorf("LD_LIBRARY_PATH should preserve old value %s, got %s", old, newLibPath)
	}
}

func TestIxsmmiEnvDoesNotMutateGlobalEnv(t *testing.T) {
	origPath := os.Getenv("PATH")
	origLibPath := os.Getenv("LD_LIBRARY_PATH")

	ixsmiEnv("/usr/local/corex-4.4.0/bin/ixsmi")

	if os.Getenv("PATH") != origPath {
		t.Errorf("global PATH mutated: got %s, want %s", os.Getenv("PATH"), origPath)
	}
	if os.Getenv("LD_LIBRARY_PATH") != origLibPath {
		t.Errorf("global LD_LIBRARY_PATH mutated: got %s, want %s", os.Getenv("LD_LIBRARY_PATH"), origLibPath)
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
