package mx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

const mxPath = "/usr/bin/mx-smi"

func New() *mxCommand {
	return &mxCommand{path: mxPath}
}

type mxCommand struct {
	path string
}

func (m *mxCommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	output, err := mxCmd(ctx, m.path)
	if err != nil {
		return nil, err
	}
	return parseMxOutput(output)
}

func (m *mxCommand) Vendor() string {
	return "mx"
}

func mxCmd(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, path, "--show-temperature", "--show-usage", "--show-memory")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", &gpu.OpError{Vendor: "mx", Op: "query", Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		opErr := &gpu.OpError{Vendor: "mx", Op: "query", Command: cmd.String(), Err: err}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %w", gpu.ErrUnavailable, opErr)
		}
		return "", opErr
	}
	return string(output), nil
}

func parseMxOutput(output string) ([]gpu.GPUInfo, error) {
	lines := strings.Split(output, "\n")
	infos := make([]gpu.GPUInfo, 0)
	var current *gpu.GPUInfo
	gpuCount := -1

	for _, line := range lines {
		if strings.Contains(line, "Attached GPUs") {
			_, value, ok := strings.Cut(line, ":")
			if !ok {
				return nil, fmt.Errorf("parse mx-smi attached GPU count %q", line)
			}
			count, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("parse mx-smi attached GPU count %q: %w", value, err)
			}
			if count < 0 {
				return nil, fmt.Errorf("parse mx-smi attached GPU count %q: negative count", value)
			}
			gpuCount = count
			continue
		}

		if strings.Contains(line, "GPU#") && strings.Contains(line, "MX") {
			if current != nil {
				infos = append(infos, *current)
			}
			info := gpu.GPUInfo{}
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasPrefix(field, "GPU#") {
					num, err := strconv.Atoi(strings.TrimPrefix(field, "GPU#"))
					if err != nil || num < 0 {
						return nil, fmt.Errorf("parse mx-smi GPU number %q", field)
					}
					info.Num = num
				} else if field == "MXN260" {
					info.CardModel = "MXN260"
				}
			}
			if len(fields) > 2 {
				info.PCIBus = fields[len(fields)-1]
			}
			current = &info
			continue
		}
		if current == nil {
			continue
		}

		var err error
		if strings.Contains(line, "hotspot") && strings.Contains(line, ":") {
			_, value, ok := strings.Cut(line, ":")
			if ok {
				current.TemperatureJunction, err = parseMXFloat(value, "°C", "C")
			}
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "vram total") && strings.Contains(line, ":") {
			_, value, ok := strings.Cut(line, ":")
			if ok {
				current.VRAMTotalMemory, err = parseKBToBytes(value)
			}
		}
		if strings.HasPrefix(trimmed, "vram used") && strings.Contains(line, ":") {
			_, value, ok := strings.Cut(line, ":")
			if ok {
				current.VRAMTotalUsedMemory, err = parseKBToBytes(value)
			}
		}
		if strings.Contains(line, "GPU") && strings.Contains(line, ":") && strings.Contains(line, "%") &&
			!strings.Contains(line, "VPUE") && !strings.Contains(line, "VPUD") {
			_, value, ok := strings.Cut(line, ":")
			if ok {
				current.GPUUse, err = parseMXFloat(value, "%")
			}
		}
		if err != nil {
			return nil, fmt.Errorf("parse mx-smi gpu %d field %q: %w", current.Num, line, err)
		}
	}
	if current != nil {
		infos = append(infos, *current)
	}
	if gpuCount >= 0 && len(infos) != gpuCount {
		return nil, fmt.Errorf("parse mx-smi output: attached GPU count %d does not match %d parsed devices", gpuCount, len(infos))
	}
	if len(infos) == 0 && gpuCount < 0 && strings.TrimSpace(output) != "" {
		return nil, fmt.Errorf("parse mx-smi output: no GPU devices found")
	}
	return infos, nil
}

func parseMXFloat(value string, expectedUnits ...string) (*float64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	fields := strings.Fields(value)
	if len(fields) != 2 || !slices.Contains(expectedUnits, fields[1]) {
		return nil, fmt.Errorf("invalid numeric value %q", value)
	}
	return gpu.ParseFloat(fields[0])
}

func parseKBToBytes(value string) (*uint64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	fields := strings.Fields(value)
	if len(fields) != 2 || fields[1] != "KB" {
		return nil, fmt.Errorf("invalid KB value %q", value)
	}
	kb, err := gpu.ParseUint(fields[0])
	if err != nil || kb == nil {
		return nil, err
	}
	if *kb > math.MaxUint64/1024 {
		return nil, fmt.Errorf("memory size out of range %q", value)
	}
	return gpu.Uint64(*kb * 1024), nil
}
