package ix

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *ixGPU {
	return &ixGPU{}
}

func scanCorexSmiPaths(rootDir string) ([]string, error) {
	dirs, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, dir := range dirs {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), "corex") {
			smiPath := filepath.Join(rootDir, dir.Name(), "bin", "ixsmi")
			if _, err := os.Stat(smiPath); err == nil {
				paths = append(paths, smiPath)
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	slices.Sort(paths)
	return paths, nil
}

func autoFindSmiPath() (string, error) {
	paths, err := scanCorexSmiPaths("/usr/local")
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}
	return paths[0], nil
}

func ixsmiEnv(smiPath string) []string {
	smiDir := strings.TrimSuffix(smiPath, "/bin/ixsmi")
	overrides := map[string]string{
		"PATH":            smiDir + "/bin:" + os.Getenv("PATH"),
		"LD_LIBRARY_PATH": fmt.Sprintf("%s/lib:%s/lib64:%s", smiDir, smiDir, os.Getenv("LD_LIBRARY_PATH")),
	}
	env := make([]string, 0, len(os.Environ())+len(overrides))
	seen := make(map[string]bool, len(overrides))
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if value, ok := overrides[key]; ok {
			env = append(env, key+"="+value)
			seen[key] = true
		} else {
			env = append(env, kv)
		}
	}
	for key, value := range overrides {
		if !seen[key] {
			env = append(env, key+"="+value)
		}
	}
	return env
}

type ixGPU struct{}

func (a *ixGPU) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "load", Err: fmt.Errorf("unsupported platform %s: %w", runtime.GOOS, gpu.ErrUnavailable)}
	}
	smiPath, err := autoFindSmiPath()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("%w: %w", gpu.ErrUnavailable, err)
		}
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "find ixsmi", Err: err}
	}
	if smiPath == "" {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "find ixsmi", Err: fmt.Errorf("ixsmi not found: %w", gpu.ErrUnavailable)}
	}
	return a.load(ctx, smiPath)
}

func (a *ixGPU) load(ctx context.Context, smiPath string) ([]gpu.GPUInfo, error) {
	cmd := exec.CommandContext(ctx, smiPath, "-q", "-x")
	cmd.Env = ixsmiEnv(smiPath)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "query", Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(stderr.String()); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "query", Command: cmd.String(), Err: err}
	}
	result, err := ParseIXSMI(string(data))
	if err != nil {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "parse output", Command: cmd.String(), Err: err}
	}
	return result, nil
}

func (a *ixGPU) Vendor() string {
	return "Iluvatar"
}

func ParseIXSMI(data string) ([]gpu.GPUInfo, error) {
	type MemoryUsage struct {
		Total string `xml:"total"`
		Used  string `xml:"used"`
	}
	type Utilization struct {
		GPUUtil    string `xml:"gpu_util"`
		MemoryUtil string `xml:"memory_util"`
	}
	type Temperature struct {
		GPUTemp string `xml:"gpu_temp"`
	}
	type PowerReadings struct {
		GPUPowerDraw string `xml:"gpu_power_draw"`
	}
	type PCI struct {
		Bus         string `xml:"pci_bus"`
		Device      string `xml:"pci_device"`
		Domain      string `xml:"pci_domain"`
		DeviceID    string `xml:"pci_device_id"`
		BusID       string `xml:"pci_bus_id"`
		SubSystemID string `xml:"pci_sub_system_id"`
	}
	type GPU struct {
		ID            string        `xml:"id,attr"`
		Product       string        `xml:"product_name"`
		Serial        string        `xml:"serial"`
		Minor         string        `xml:"minor_number"`
		Memory        MemoryUsage   `xml:"memory_usage"`
		Util          Utilization   `xml:"utilization"`
		Temp          Temperature   `xml:"temperature"`
		PowerReadings PowerReadings `xml:"power_readings"`
		PCI           PCI           `xml:"pci"`
	}
	type IXSMILog struct {
		XMLName      xml.Name `xml:"ixsmi_log"`
		AttachedGPUs *int     `xml:"attached_gpus"`
		GPUs         []GPU    `xml:"gpu"`
	}

	var log IXSMILog
	if err := xml.Unmarshal([]byte(data), &log); err != nil {
		return nil, fmt.Errorf("parse ixsmi XML: %w", err)
	}
	if log.AttachedGPUs != nil && (*log.AttachedGPUs < 0 || *log.AttachedGPUs != len(log.GPUs)) {
		return nil, fmt.Errorf("parse ixsmi XML: attached GPU count %d does not match %d parsed devices", *log.AttachedGPUs, len(log.GPUs))
	}
	infos := make([]gpu.GPUInfo, 0, len(log.GPUs))
	for i, device := range log.GPUs {
		num := i
		if minor := strings.TrimSpace(device.Minor); minor != "" && !strings.EqualFold(minor, "N/A") {
			parsed, err := strconv.Atoi(minor)
			if err != nil || parsed < 0 {
				return nil, fmt.Errorf("parse ixsmi gpu %d minor number %q", i, device.Minor)
			}
			num = parsed
		}
		memTotal, err := parseMiBToBytes(device.Memory.Total)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d total memory: %w", num, err)
		}
		memUsed, err := parseMiBToBytes(device.Memory.Used)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d used memory: %w", num, err)
		}
		temp, err := parseTempC(device.Temp.GPUTemp)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d temperature: %w", num, err)
		}
		gpuUse, err := parsePercent(device.Util.GPUUtil)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d utilization: %w", num, err)
		}
		power, err := parsePowerW(device.PowerReadings.GPUPowerDraw)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d power: %w", num, err)
		}
		var pciBus string
		if device.PCI.BusID != "" {
			parts := strings.Split(device.PCI.BusID, ":")
			if len(parts) > 0 {
				parts[0] = device.PCI.Domain
			}
			pciBus = strings.Join(parts, ":")
		}
		infos = append(infos, gpu.GPUInfo{
			Num:                         num,
			DeviceID:                    device.ID,
			SerialNumber:                device.Serial,
			VRAMTotalMemory:             memTotal,
			VRAMTotalUsedMemory:         memUsed,
			TemperatureEdge:             temp,
			AverageGraphicsPackagePower: power,
			GPUUse:                      gpuUse,
			CardSeries:                  device.Product,
			CardModel:                   device.Product,
			CardVendor:                  "Iluvatar",
			PCIBus:                      pciBus,
		})
	}
	return infos, nil
}

func parseIXFloat(value, expectedUnit string) (*float64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	fields := strings.Fields(value)
	if len(fields) != 2 || fields[1] != expectedUnit {
		return nil, fmt.Errorf("invalid %s value %q", expectedUnit, value)
	}
	return gpu.ParseFloat(fields[0])
}

func parseMiBToBytes(value string) (*uint64, error) {
	parsed, err := parseIXFloat(value, "MiB")
	if err != nil || parsed == nil {
		return nil, err
	}
	bytes := *parsed * 1024 * 1024
	if *parsed < 0 || bytes >= math.Exp2(64) {
		return nil, fmt.Errorf("memory size out of range %q", value)
	}
	return gpu.Uint64(uint64(bytes + 0.5)), nil
}

func parseTempC(value string) (*float64, error) {
	return parseIXFloat(value, "C")
}

func parsePercent(value string) (*float64, error) {
	return parseIXFloat(value, "%")
}

func parsePowerW(value string) (*float64, error) {
	return parseIXFloat(value, "W")
}
