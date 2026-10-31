package amdriscv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

var (
	cardNamePattern = regexp.MustCompile(`^card[0-9]+$`)
	pciBusPattern   = regexp.MustCompile(`^(?:[0-9a-fA-F]{4}:)?[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-7]$`)
)

func New() *amdRISCVGPU {
	return &amdRISCVGPU{drmRoot: "/sys/class/drm", sensorsCommand: "sensors"}
}

type amdRISCVGPU struct {
	drmRoot        string
	sensorsCommand string
}

func (a *amdRISCVGPU) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "riscv64" {
		return nil, &gpu.OpError{
			Vendor: a.Vendor(),
			Op:     "load",
			Err:    fmt.Errorf("unsupported platform %s/%s: %w", runtime.GOOS, runtime.GOARCH, gpu.ErrUnavailable),
		}
	}
	infos, err := a.load(ctx, a.drmRoot, a.sensorsCommand)
	if err != nil {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "load", Err: err}
	}
	return infos, nil
}

func (a *amdRISCVGPU) load(ctx context.Context, drmRoot, sensorsCommand string) ([]gpu.GPUInfo, error) {
	sensorOutput, err := a.querySensors(ctx, sensorsCommand)
	if err != nil {
		return nil, err
	}
	sensors, err := parseSensors(sensorOutput)
	if err != nil {
		return nil, fmt.Errorf("parse sensors output: %w", err)
	}
	entries, err := os.ReadDir(drmRoot)
	if err != nil {
		return nil, fmt.Errorf("read DRM devices: %w", err)
	}

	infos := make([]gpu.GPUInfo, 0)
	for _, entry := range entries {
		if !cardNamePattern.MatchString(entry.Name()) {
			continue
		}
		num, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), "card"))
		if err != nil {
			return nil, fmt.Errorf("parse AMD-riscv device number %q: %w", entry.Name(), err)
		}
		devicePath := filepath.Join(drmRoot, entry.Name(), "device")
		deviceID, err := os.ReadFile(filepath.Join(devicePath, "device"))
		if err != nil {
			return nil, fmt.Errorf("read AMD-riscv %s device ID: %w", entry.Name(), err)
		}
		pciBus, err := devicePCIBus(devicePath)
		if err != nil {
			return nil, fmt.Errorf("read AMD-riscv %s PCI address: %w", entry.Name(), err)
		}
		gpuUse, err := readUintMetric(filepath.Join(devicePath, "gpu_busy_percent"))
		if err != nil {
			return nil, fmt.Errorf("read AMD-riscv %s utilization: %w", entry.Name(), err)
		}
		vramTotal, err := readUintMetric(filepath.Join(devicePath, "mem_info_vram_total"))
		if err != nil {
			return nil, fmt.Errorf("read AMD-riscv %s total memory: %w", entry.Name(), err)
		}
		vramUsed, err := readUintMetric(filepath.Join(devicePath, "mem_info_vram_used"))
		if err != nil {
			return nil, fmt.Errorf("read AMD-riscv %s used memory: %w", entry.Name(), err)
		}

		info := gpu.GPUInfo{
			Num:                 num,
			DeviceID:            strings.TrimSpace(string(deviceID)),
			CardVendor:          "AMD",
			PCIBus:              pciBus,
			VRAMTotalMemory:     vramTotal,
			VRAMTotalUsedMemory: vramUsed,
		}
		if gpuUse != nil {
			info.GPUUse = gpu.Float64(float64(*gpuUse))
		}
		if sensor, ok := sensors[sensorName(pciBus)]; ok {
			info.TemperatureEdge = sensor.Edge.Temp1Input
			info.TemperatureJunction = sensor.Junction.Temp2Input
			info.TemperatureMemory = sensor.Mem.Temp3Input
			info.AverageGraphicsPackagePower = sensor.PPT.Power1Average
			if info.AverageGraphicsPackagePower == nil {
				info.AverageGraphicsPackagePower = sensor.PPT.Power1Input
			}
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Num < infos[j].Num })
	return infos, nil
}

func (a *amdRISCVGPU) querySensors(ctx context.Context, command string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, "-j")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "query sensors", Command: cmd.String(), Err: ctxErr}
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if message := strings.TrimSpace(stderr.String()); message != "" {
		err = fmt.Errorf("%w: %s", err, message)
	}
	return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "query sensors", Command: cmd.String(), Err: err}
}

func readUintMetric(path string) (*uint64, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	value, err := gpu.ParseUint(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return value, nil
}

func devicePCIBus(devicePath string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(devicePath); err == nil {
		name := filepath.Base(resolved)
		if pciBusPattern.MatchString(name) {
			return strings.ToLower(name), nil
		}
	}
	data, err := os.ReadFile(filepath.Join(devicePath, "uevent"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if value, ok := strings.CutPrefix(line, "PCI_SLOT_NAME="); ok {
			value = strings.TrimSpace(value)
			if !pciBusPattern.MatchString(value) {
				return "", fmt.Errorf("invalid PCI_SLOT_NAME %q", value)
			}
			return strings.ToLower(value), nil
		}
	}
	return "", nil
}

func sensorName(pciBus string) string {
	if pciBus == "" {
		return ""
	}
	parts := strings.Split(pciBus, ":")
	if len(parts) < 2 {
		return ""
	}
	busDevice := parts[len(parts)-2] + parts[len(parts)-1]
	busDevice, _, _ = strings.Cut(busDevice, ".")
	return "amdgpu-pci-" + strings.ToLower(busDevice)
}

type sensorTemperature1 struct {
	Temp1Input *float64 `json:"temp1_input"`
}

type sensorTemperature2 struct {
	Temp2Input *float64 `json:"temp2_input"`
}

type sensorTemperature3 struct {
	Temp3Input *float64 `json:"temp3_input"`
}

type sensorPower struct {
	Power1Average *float64 `json:"power1_average"`
	Power1Input   *float64 `json:"power1_input"`
}

type amdSensor struct {
	Edge     sensorTemperature1 `json:"edge"`
	Junction sensorTemperature2 `json:"junction"`
	Mem      sensorTemperature3 `json:"mem"`
	PPT      sensorPower        `json:"PPT"`
}

func parseSensors(data []byte) (map[string]amdSensor, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var sensors map[string]amdSensor
	if err := json.Unmarshal(data, &sensors); err != nil {
		return nil, err
	}
	return sensors, nil
}

func (a *amdRISCVGPU) Vendor() string {
	return "AMD-riscv"
}
