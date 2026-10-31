package houmoai

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

var (
	hmDeviceRegex = regexp.MustCompile(`^\s*device(\d+)\s+detail\s+infos\s*$`)
	hmKVRegex     = regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*:\s*(.*)$`)
	hmFloatRegex  = regexp.MustCompile(`^\s*([-+]?(?:\d+(?:\.\d*)?|\.\d+))\s*(?i:(%|C|W))\s*$`)
	hmMBRegex     = regexp.MustCompile(`^\s*([-+]?(?:\d+(?:\.\d*)?|\.\d+))\s*(?i:MB)\s*$`)
)

func New() *houmoaiSMICommand {
	return &houmoaiSMICommand{}
}

type houmoaiSMICommand struct{}

func (h *houmoaiSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	cmd := exec.CommandContext(ctx, "hm_smi", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: h.Vendor(), Op: "query", Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		opErr := &gpu.OpError{Vendor: h.Vendor(), Op: "query", Command: cmd.String(), Err: err}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %w", gpu.ErrUnavailable, opErr)
		}
		return nil, opErr
	}
	return h.parse(output)
}

func (h *houmoaiSMICommand) Vendor() string {
	return "Houmo"
}

func (h *houmoaiSMICommand) parse(output []byte) ([]gpu.GPUInfo, error) {
	result := make([]gpu.GPUInfo, 0)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var current map[string]string
	var currentNum int
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if matches := hmDeviceRegex.FindStringSubmatch(line); matches != nil {
			if current != nil {
				info, err := h.buildGPUInfo(currentNum, current)
				if err != nil {
					return nil, err
				}
				result = append(result, info)
			}
			num, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, fmt.Errorf("parse hm_smi device number %q: %w", matches[1], err)
			}
			currentNum = num
			current = make(map[string]string)
			continue
		}
		if current == nil {
			continue
		}
		if matches := hmKVRegex.FindStringSubmatch(line); matches != nil {
			current[strings.TrimSpace(matches[1])] = strings.TrimSpace(matches[2])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan hm_smi output: %w", err)
	}
	if current != nil {
		info, err := h.buildGPUInfo(currentNum, current)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	if len(result) == 0 && strings.TrimSpace(string(output)) != "" {
		return nil, fmt.Errorf("parse hm_smi output: no GPU devices found")
	}
	return result, nil
}

func (h *houmoaiSMICommand) buildGPUInfo(num int, fields map[string]string) (gpu.GPUInfo, error) {
	totalMB, err := parseFloatMB(fields["DDR_Memory_Total"])
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d DDR_Memory_Total: %w", num, err)
	}
	freeMB, err := parseFloatMB(fields["DDR_Memory_Free"])
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d DDR_Memory_Free: %w", num, err)
	}
	var usedMB *float64
	if totalMB != nil && freeMB != nil {
		used := *totalMB - *freeMB
		if used < 0 {
			return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d memory: free memory exceeds total", num)
		}
		usedMB = &used
	}
	totalBytes, err := mbToBytes(totalMB)
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d total memory: %w", num, err)
	}
	usedBytes, err := mbToBytes(usedMB)
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d used memory: %w", num, err)
	}
	avgTemp, err := averageDDRTemp(fields)
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d DDR temperature: %w", num, err)
	}
	gpuUse, err := firstFloat(fields["Average_Util"], "%")
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d Average_Util: %w", num, err)
	}
	edge, err := firstFloat(fields["Core0"], "C")
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d Core0: %w", num, err)
	}
	junction, err := firstFloat(fields["Core1"], "C")
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d Core1: %w", num, err)
	}
	power, err := firstFloat(fields["Board_Power"], "W")
	if err != nil {
		return gpu.GPUInfo{}, fmt.Errorf("parse hm_smi gpu %d Board_Power: %w", num, err)
	}

	return gpu.GPUInfo{
		Num:                         num,
		DeviceID:                    fields["Dev"],
		CardVendor:                  fields["Vendor"],
		CardModel:                   fields["Model"],
		CardSKU:                     fields["PN"],
		SerialNumber:                fields["SN"],
		DeviceRev:                   fields["Firmware_Version"],
		PCIBus:                      fields["BDF"],
		GPUUse:                      gpuUse,
		VRAMTotalMemory:             totalBytes,
		VRAMTotalUsedMemory:         usedBytes,
		TemperatureEdge:             edge,
		TemperatureJunction:         junction,
		TemperatureMemory:           avgTemp,
		AverageGraphicsPackagePower: power,
	}, nil
}

func parseFloatMB(value string) (*float64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	matches := hmMBRegex.FindStringSubmatch(value)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid MB value %q", value)
	}
	parsed, err := gpu.ParseFloat(matches[1])
	if err != nil {
		return nil, err
	}
	if parsed != nil && *parsed < 0 {
		return nil, fmt.Errorf("negative memory value %q", value)
	}
	return parsed, nil
}

func mbToBytes(mb *float64) (*uint64, error) {
	if mb == nil {
		return nil, nil
	}
	bytes := *mb * 1024 * 1024
	if math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes < 0 || bytes >= math.Exp2(64) {
		return nil, fmt.Errorf("memory size out of range")
	}
	return gpu.Uint64(uint64(bytes + 0.5)), nil
}

func firstFloat(value, expectedUnit string) (*float64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	matches := hmFloatRegex.FindStringSubmatch(value)
	if len(matches) != 3 || !strings.EqualFold(matches[2], expectedUnit) {
		return nil, fmt.Errorf("invalid %s value %q", expectedUnit, value)
	}
	return gpu.ParseFloat(matches[1])
}

func averageDDRTemp(fields map[string]string) (*float64, error) {
	keys := []string{"DDR0", "DDR2", "DDR4", "DDR5"}
	var sum float64
	var count int
	for _, key := range keys {
		value, err := firstFloat(fields[key], "C")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		if value != nil {
			sum += *value
			count++
		}
	}
	if count == 0 {
		return nil, nil
	}
	average := sum / float64(count)
	if math.IsNaN(average) || math.IsInf(average, 0) {
		return nil, fmt.Errorf("DDR temperature average is not finite")
	}
	return gpu.Float64(average), nil
}
