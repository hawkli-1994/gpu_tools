package mx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

const mxPath = "/usr/bin/mx-smi"

func New() *mxCommand {
	return &mxCommand{}
}

type mxCommand struct{}

func (m *mxCommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	output, err := mxCmd(ctx)
	if err != nil {
		return nil, err
	}
	return parseMxOutput(output)
}

func (m *mxCommand) Vendor() string {
	return "mx"
}

func mxCmd(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, mxPath, "--show-temperature", "--show-usage", "--show-memory")
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
	var gpuCount int

	for _, line := range lines {
		if strings.Contains(line, "Attached GPUs") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err != nil {
					return nil, fmt.Errorf("parse mx-smi attached GPU count %q: %w", parts[1], err)
				}
				if count < 0 {
					return nil, fmt.Errorf("parse mx-smi attached GPU count %q: negative count", parts[1])
				}
				gpuCount = count
			}
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
					if err != nil {
						return nil, fmt.Errorf("parse mx-smi GPU number %q: %w", field, err)
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
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				value = strings.TrimSuffix(value, "°C")
				value = strings.TrimSuffix(value, " C")
				current.TemperatureJunction, err = gpu.ParseFloat(value)
			}
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "vram total") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				current.VRAMTotalMemory, err = parseKBToBytes(parts[1])
			}
		}
		if strings.HasPrefix(trimmed, "vram used") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				current.VRAMTotalUsedMemory, err = parseKBToBytes(parts[1])
			}
		}
		if strings.Contains(line, "GPU") && strings.Contains(line, ":") && strings.Contains(line, "%") &&
			!strings.Contains(line, "VPUE") && !strings.Contains(line, "VPUD") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "%"))
				current.GPUUse, err = gpu.ParseFloat(value)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("parse mx-smi gpu %d field %q: %w", current.Num, line, err)
		}
	}
	if current != nil {
		infos = append(infos, *current)
	}
	if gpuCount > 0 && len(infos) > gpuCount {
		infos = infos[:gpuCount]
	}
	return infos, nil
}

func parseKBToBytes(value string) (*uint64, error) {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "KB"))
	kb, err := gpu.ParseUint(value)
	if err != nil || kb == nil {
		return nil, err
	}
	if *kb > math.MaxUint64/1024 {
		return nil, fmt.Errorf("memory size out of range %q", value)
	}
	return gpu.Uint64(*kb * 1024), nil
}
