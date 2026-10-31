package nvidia

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *nvidiaSMICommand {
	return &nvidiaSMICommand{}
}

type nvidiaSMICommand struct {
}

func (n *nvidiaSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	const command = "nvidia-smi"
	if _, err := exec.LookPath(command); err != nil {
		return nil, &gpu.OpError{Vendor: n.Vendor(), Op: "load", Command: command, Err: fmt.Errorf("%w: %w", gpu.ErrUnavailable, err)}
	}
	cmd := exec.CommandContext(ctx, command, "--format=csv,noheader", "--query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu,pci.bus_id")
	output, err := cmd.Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			if message := strings.TrimSpace(string(exitErr.Stderr)); message != "" {
				err = fmt.Errorf("%w: %s", err, message)
			}
		}
		return nil, &gpu.OpError{Vendor: n.Vendor(), Op: "load", Command: command, Err: err}
	}
	return n.parse(output)
}

func (n *nvidiaSMICommand) parse(output []byte) ([]gpu.GPUInfo, error) {
	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse nvidia-smi CSV output: %w", err)
	}

	infos := make([]gpu.GPUInfo, 0, len(records))
	for _, row := range records {
		if len(row) != 7 {
			return nil, fmt.Errorf("parse nvidia-smi row: expected 7 columns, got %d", len(row))
		}
		index, err := strconv.Atoi(strings.TrimSpace(row[0]))
		if err != nil {
			return nil, fmt.Errorf("parse nvidia-smi GPU index %q: %w", row[0], err)
		}
		if index < 0 {
			return nil, fmt.Errorf("parse nvidia-smi GPU index %q: must be non-negative", row[0])
		}
		totalMemory, err := parseMemory(row[2])
		if err != nil {
			return nil, fmt.Errorf("parse nvidia-smi GPU %d total memory: %w", index, err)
		}
		usedMemory, err := parseMemory(row[3])
		if err != nil {
			return nil, fmt.Errorf("parse nvidia-smi GPU %d used memory: %w", index, err)
		}
		utilization, err := parseMetric(row[4])
		if err != nil {
			return nil, fmt.Errorf("parse nvidia-smi GPU %d utilization: %w", index, err)
		}
		temperature, err := gpu.ParseFloat(strings.TrimSpace(row[5]))
		if err != nil {
			return nil, fmt.Errorf("parse nvidia-smi GPU %d temperature: %w", index, err)
		}
		infos = append(infos, gpu.GPUInfo{
			Num:                 index,
			DeviceID:            strconv.Itoa(index),
			CardModel:           strings.TrimSpace(row[1]),
			CardVendor:          n.Vendor(),
			CardSeries:          n.Vendor(),
			VRAMTotalMemory:     totalMemory,
			VRAMTotalUsedMemory: usedMemory,
			GPUUse:              utilization,
			TemperatureEdge:     temperature,
			PCIBus:              strings.TrimSpace(row[6]),
		})
	}
	return infos, nil
}

func parseMemory(value string) (*uint64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 || strings.EqualFold(fields[0], "N/A") {
		return nil, nil
	}
	if len(fields) != 2 || !strings.EqualFold(fields[1], "MiB") {
		return nil, fmt.Errorf("expected MiB quantity, got %q", value)
	}
	memory, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return nil, err
	}
	if memory > math.MaxUint64/(1024*1024) {
		return nil, fmt.Errorf("memory quantity %q overflows bytes", value)
	}
	return gpu.Uint64(memory * 1024 * 1024), nil
}

func parseMetric(value string) (*float64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 || len(fields) == 1 && strings.EqualFold(fields[0], "N/A") {
		return nil, nil
	}
	if len(fields) != 2 || fields[1] != "%" || strings.EqualFold(fields[0], "N/A") {
		return nil, fmt.Errorf("expected percentage or N/A, got %q", value)
	}
	return gpu.ParseFloat(fields[0])
}

func (n *nvidiaSMICommand) Vendor() string {
	return "NVIDIA"
}

func (n *nvidiaSMICommand) DriverInfo(ctx context.Context) (gpu.GPUDriverInfo, error) {
	const command = "nvidia-smi"
	path, err := exec.LookPath(command)
	if err != nil {
		return gpu.GPUDriverInfo{}, &gpu.OpError{Vendor: n.Vendor(), Op: "driver info", Command: command, Err: fmt.Errorf("%w: %w", gpu.ErrUnavailable, err)}
	}
	output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
		} else if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		return gpu.GPUDriverInfo{}, &gpu.OpError{Vendor: n.Vendor(), Op: "driver info", Command: command, Err: err}
	}
	info, err := ParseVersion(string(output))
	if err != nil {
		return gpu.GPUDriverInfo{}, fmt.Errorf("parse nvidia-smi version info: %w", err)
	}
	info.Vendor = n.Vendor()
	info.Installed = true
	info.InstallPath = path
	return info, nil
}

func ParseVersion(versionInfo string) (gpu.GPUDriverInfo, error) {
	info := gpu.GPUDriverInfo{}
	lines := strings.Split(versionInfo, "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "NVIDIA-SMI version":
			info.ClientVersion = value
		case "DRIVER version":
			info.Version = value
		case "CUDA Version":
			info.LibVersion = value
		}
	}

	if info.ClientVersion == "" || info.Version == "" || info.LibVersion == "" {
		return info, fmt.Errorf("failed to parse version info: missing required fields")
	}

	return info, nil
}
