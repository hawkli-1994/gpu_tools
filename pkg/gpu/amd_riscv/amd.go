package amdriscv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"log/slog"

	"github.com/hawkli-1994/go-radeontop/pkg/monitor"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&amdRISCVGPU{})
}

type amdRISCVGPU struct {
}

func (a *amdRISCVGPU) Load() (*gpu.GPUInfoList, error) {

	mon, err := monitor.New(slog.Default())
	if err != nil {
		return nil, err
	}
	gpuinfo, err := mon.GetDeviceInfoList()
	if err != nil {
		return nil, err
	}
	gpuInfoList := &gpu.GPUInfoList{}
	for _, gpuInfo := range gpuinfo.Items {
		num := strings.TrimPrefix(gpuInfo.Name, "card")
		numInt, err := strconv.Atoi(num)
		if err != nil {
			return nil, err
		}
		gpuInfoList.GPUInfos = append(gpuInfoList.GPUInfos, gpu.GPUInfo{
			Num:                 numInt,
			DeviceID:            gpuInfo.DeviceID,
			TemperatureEdge:     fmt.Sprintf("%.2f", gpuInfo.Stats.GpuTempEdge),
			TemperatureJunction: fmt.Sprintf("%.2f", gpuInfo.Stats.GpuTempJunction),
			TemperatureMemory:   fmt.Sprintf("%.2f", gpuInfo.Stats.GpuTempMem),
			GPUUse:              fmt.Sprintf("%.2f", gpuInfo.Stats.GPUUsage),
			SerialNumber:        gpuInfo.DeviceID,
			VRAMTotalMemory:     fmt.Sprintf("%d", gpuInfo.Stats.VRAMTotal),
			VRAMTotalUsedMemory: fmt.Sprintf("%d", gpuInfo.Stats.VRAMUsed),
			CardSeries:          gpuInfo.DeviceID,
		})
	}

	return gpuInfoList, nil
}

func (a *amdRISCVGPU) Available() bool {
	// set slog print to console
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// logger.Info("amdRISCVGPU Available", "os", runtime.GOOS, "arch", runtime.GOARCH)
	if runtime.GOOS != "linux" || runtime.GOARCH != "riscv64" {
		logger.Info("amdRISCVGPU Available", "os", runtime.GOOS, "arch", runtime.GOARCH, "return", false)
		return false
	}

	lsClassDrm := "/sys/class/drm/"
	cmd := exec.Command("ls", lsClassDrm)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Info("amdRISCVGPU Available", "lsClassDrm", lsClassDrm, "error", err, "return", false)
		return false
	}
	if !strings.Contains(string(output), "card0") {
		logger.Info("amdRISCVGPU Available", "lsClassDrm", lsClassDrm, "output", string(output), "return", false)
		return false
	}
	devicePath := filepath.Join(lsClassDrm, "card0", "device", "device")
	cmd = exec.Command("cat", devicePath)

	output, err = cmd.CombinedOutput()
	if err != nil {
		logger.Info("amdRISCVGPU Available", "devicePath", devicePath, "error", err, "return", false)
		return false
	}
	if string(output) != "" {
		logger.Info("amdRISCVGPU Available", "devicePath", devicePath, "output", string(output), "return", true)
		return true
	}
	logger.Info("amdRISCVGPU Available", "devicePath", devicePath, "output", string(output), "return", false)
	return false
}

func (a *amdRISCVGPU) Vendor() string {
	return "AMD-riscv"
}
