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
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "find ixsmi", Err: fmt.Errorf("%w: %w", gpu.ErrUnavailable, err)}
	}
	if smiPath == "" {
		return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "find ixsmi", Err: fmt.Errorf("ixsmi not found: %w", gpu.ErrUnavailable)}
	}

	cmd := exec.CommandContext(ctx, smiPath, "-q", "-x")
	cmd.Env = ixsmiEnv(smiPath)
	data, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: a.Vendor(), Op: "query", Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(data)); message != "" {
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
	type PCI struct {
		Bus         string `xml:"pci_bus"`
		Device      string `xml:"pci_device"`
		Domain      string `xml:"pci_domain"`
		DeviceID    string `xml:"pci_device_id"`
		BusID       string `xml:"pci_bus_id"`
		SubSystemID string `xml:"pci_sub_system_id"`
	}
	type GPU struct {
		ID      string      `xml:"id,attr"`
		Product string      `xml:"product_name"`
		Serial  string      `xml:"serial"`
		Minor   string      `xml:"minor_number"`
		Memory  MemoryUsage `xml:"memory_usage"`
		Util    Utilization `xml:"utilization"`
		Temp    Temperature `xml:"temperature"`
		PCI     PCI         `xml:"pci"`
	}
	type IXSMILog struct {
		XMLName xml.Name `xml:"ixsmi_log"`
		GPUs    []GPU    `xml:"gpu"`
	}

	var log IXSMILog
	if err := xml.Unmarshal([]byte(data), &log); err != nil {
		return nil, fmt.Errorf("parse ixsmi XML: %w", err)
	}
	infos := make([]gpu.GPUInfo, 0, len(log.GPUs))
	for i, device := range log.GPUs {
		memTotal, err := parseMiBToBytes(device.Memory.Total)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d total memory: %w", i, err)
		}
		memUsed, err := parseMiBToBytes(device.Memory.Used)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d used memory: %w", i, err)
		}
		temp, err := parseTempC(device.Temp.GPUTemp)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d temperature: %w", i, err)
		}
		gpuUse, err := parsePercent(device.Util.GPUUtil)
		if err != nil {
			return nil, fmt.Errorf("parse ixsmi gpu %d utilization: %w", i, err)
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
			Num:                 i,
			DeviceID:            device.ID,
			SerialNumber:        device.Serial,
			VRAMTotalMemory:     memTotal,
			VRAMTotalUsedMemory: memUsed,
			TemperatureEdge:     temp,
			GPUUse:              gpuUse,
			CardSeries:          device.Product,
			CardModel:           device.Product,
			CardVendor:          "Iluvatar",
			PCIBus:              pciBus,
		})
	}
	return infos, nil
}

func parseIXFloat(value string) (*float64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil, nil
	}
	parsed, err := gpu.ParseFloat(fields[0])
	if err != nil {
		return nil, err
	}
	if parsed != nil && (math.IsNaN(*parsed) || math.IsInf(*parsed, 0)) {
		return nil, fmt.Errorf("non-finite value %q", value)
	}
	return parsed, nil
}

func parseMiBToBytes(value string) (*uint64, error) {
	parsed, err := parseIXFloat(value)
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
	return parseIXFloat(value)
}

func parsePercent(value string) (*float64, error) {
	return parseIXFloat(value)
}
