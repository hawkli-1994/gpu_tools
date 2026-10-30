package amdriscv

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"

	"github.com/hawkli-1994/go-radeontop/pkg/monitor"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *amdRISCVGPU {
	return &amdRISCVGPU{}
}

type amdRISCVGPU struct{}

func (a *amdRISCVGPU) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "riscv64" {
		return nil, &gpu.OpError{
			Vendor: a.Vendor(),
			Op:     "load",
			Err:    fmt.Errorf("unsupported platform %s/%s: %w", runtime.GOOS, runtime.GOARCH, gpu.ErrUnavailable),
		}
	}

	mon, err := monitor.New(slog.Default())
	if err != nil {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "initialize monitor", Err: err}
	}
	deviceInfo, err := mon.GetDeviceInfoList()
	if err != nil {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "get device info", Err: err}
	}

	infos := make([]gpu.GPUInfo, 0, len(deviceInfo.Items))
	for _, device := range deviceInfo.Items {
		num, err := strconv.Atoi(strings.TrimPrefix(device.Name, "card"))
		if err != nil {
			return nil, fmt.Errorf("parse AMD-riscv device number %q: %w", device.Name, err)
		}
		infos = append(infos, gpu.GPUInfo{
			Num:                 num,
			DeviceID:            device.DeviceID,
			TemperatureEdge:     gpu.Float64(device.Stats.GpuTempEdge),
			TemperatureJunction: gpu.Float64(device.Stats.GpuTempJunction),
			TemperatureMemory:   gpu.Float64(device.Stats.GpuTempMem),
			GPUUse:              gpu.Float64(device.Stats.GPUUsage),
			SerialNumber:        device.DeviceID,
			VRAMTotalMemory:     gpu.Uint64(device.Stats.VRAMTotal),
			VRAMTotalUsedMemory: gpu.Uint64(device.Stats.VRAMUsed),
			CardSeries:          device.DeviceID,
		})
	}
	return infos, nil
}

func (a *amdRISCVGPU) Vendor() string {
	return "AMD-riscv"
}
