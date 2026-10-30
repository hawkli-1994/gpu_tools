package huawei

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *npuSMICommand {
	return &npuSMICommand{}
}

type npuSMICommand struct{}

func (h *npuSMICommand) Vendor() string {
	return "Huawei"
}

func (h *npuSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	smiPath, err := findSMIPath()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, smiPath, "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: h.Vendor(), Op: "list NPUs", Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		return nil, &gpu.OpError{Vendor: h.Vendor(), Op: "list NPUs", Command: cmd.String(), Err: err}
	}

	npuIDs, err := extractNPUIDs(output)
	if err != nil {
		return nil, fmt.Errorf("extract NPU IDs: %w", err)
	}

	result := make([]gpu.GPUInfo, 0)
	globalNum := 0
	for _, npuID := range npuIDs {
		boardInfo, err := h.getBoardInfo(ctx, smiPath, npuID)
		if err != nil {
			return nil, err
		}
		commonInfo, err := h.getCommonInfo(ctx, smiPath, npuID)
		if err != nil {
			return nil, err
		}
		usagesInfo, err := h.getUsagesInfo(ctx, smiPath, npuID)
		if err != nil {
			return nil, err
		}
		productInfo, err := h.getProductInfo(ctx, smiPath, npuID)
		if err != nil {
			return nil, err
		}
		powerInfo, err := h.getPowerInfo(ctx, smiPath, npuID)
		if err != nil {
			return nil, err
		}

		infos, nextNum, err := buildGPUInfoList(npuID, boardInfo, commonInfo, usagesInfo, productInfo, powerInfo, globalNum)
		if err != nil {
			return nil, err
		}
		result = append(result, infos...)
		globalNum = nextNum
	}
	return result, nil
}

func findSMIPath() (string, error) {
	for _, path := range []string{"npu-smi", "/usr/local/sbin/npu-smi"} {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}
	return "", &gpu.OpError{Vendor: "Huawei", Op: "find npu-smi", Err: fmt.Errorf("npu-smi command not found: %w", gpu.ErrUnavailable)}
}

func buildGPUInfoList(
	npuID string,
	board map[string]string,
	common map[string]map[string]string,
	usages map[string]map[string]string,
	product map[string]map[string]string,
	power map[string]string,
	globalNum int,
) ([]gpu.GPUInfo, int, error) {
	powerValue, err := parseHuaweiFloat(power["Power Dissipation(W)"])
	if err != nil {
		return nil, globalNum, fmt.Errorf("parse NPU %s Power Dissipation(W): %w", npuID, err)
	}

	chipIDSet := make(map[string]struct{})
	for chipID := range common {
		chipIDSet[chipID] = struct{}{}
	}
	for chipID := range usages {
		chipIDSet[chipID] = struct{}{}
	}
	for chipID := range product {
		chipIDSet[chipID] = struct{}{}
	}
	if len(chipIDSet) == 0 {
		return nil, globalNum, nil
	}

	chipIDs := make([]string, 0, len(chipIDSet))
	for chipID := range chipIDSet {
		chipIDs = append(chipIDs, chipID)
	}
	sort.Strings(chipIDs)

	infos := make([]gpu.GPUInfo, 0, len(chipIDs))
	for _, chipID := range chipIDs {
		chipCommon := common[chipID]
		chipUsages := usages[chipID]
		chipProduct := product[chipID]

		temp, err := parseHuaweiFloat(chipCommon["Temperature(C)"])
		if err != nil {
			return nil, globalNum, fmt.Errorf("parse NPU %s chip %s Temperature(C): %w", npuID, chipID, err)
		}
		gpuUse, err := parseHuaweiFloat(chipUsages["Aicore Usage Rate(%)"])
		if err != nil {
			return nil, globalNum, fmt.Errorf("parse NPU %s chip %s usages Aicore Usage Rate(%%): %w", npuID, chipID, err)
		}
		if gpuUse == nil {
			gpuUse, err = parseHuaweiFloat(chipCommon["Aicore Usage Rate(%)"])
			if err != nil {
				return nil, globalNum, fmt.Errorf("parse NPU %s chip %s common Aicore Usage Rate(%%): %w", npuID, chipID, err)
			}
		}

		capacityMB, err := parseHuaweiFloat(chipUsages["DDR Capacity(MB)"])
		if err != nil {
			return nil, globalNum, fmt.Errorf("parse NPU %s chip %s DDR Capacity(MB): %w", npuID, chipID, err)
		}
		var totalBytes *uint64
		if capacityMB != nil {
			bytes := *capacityMB * 1024 * 1024
			if *capacityMB < 0 || bytes >= math.Exp2(64) {
				return nil, globalNum, fmt.Errorf("parse NPU %s chip %s DDR Capacity(MB): value out of range", npuID, chipID)
			}
			totalBytes = gpu.Uint64(uint64(bytes))
		}
		usageRate, err := parseHuaweiFloat(chipUsages["DDR Usage Rate(%)"])
		if err != nil {
			return nil, globalNum, fmt.Errorf("parse NPU %s chip %s DDR Usage Rate(%%): %w", npuID, chipID, err)
		}
		var usedBytes *uint64
		if totalBytes != nil && usageRate != nil {
			used := float64(*totalBytes) * *usageRate / 100
			if *usageRate < 0 || used >= math.Exp2(64) {
				return nil, globalNum, fmt.Errorf("parse NPU %s chip %s DDR Usage Rate(%%): value out of range", npuID, chipID)
			}
			usedBytes = gpu.Uint64(uint64(used))
		}

		model := ""
		if value := chipProduct["Product Type"]; value != "" && value != "NA" {
			model = value
		}
		infos = append(infos, gpu.GPUInfo{
			Num:                         globalNum,
			DeviceID:                    chipID,
			CardVendor:                  "Huawei",
			CardSeries:                  "Ascend",
			CardModel:                   model,
			TemperatureEdge:             temp,
			GPUUse:                      gpuUse,
			VRAMTotalMemory:             totalBytes,
			VRAMTotalUsedMemory:         usedBytes,
			AverageGraphicsPackagePower: powerValue,
			SerialNumber:                board["Serial Number"],
			PCIBus:                      board["PCIe Bus Info"],
		})
		globalNum++
	}
	return infos, globalNum, nil
}

func parseHuaweiFloat(value string) (*float64, error) {
	if strings.EqualFold(strings.TrimSpace(value), "NA") {
		return nil, nil
	}
	parsed, err := gpu.ParseFloat(value)
	if err != nil {
		return nil, err
	}
	if parsed != nil && (math.IsNaN(*parsed) || math.IsInf(*parsed, 0)) {
		return nil, fmt.Errorf("non-finite value %q", value)
	}
	return parsed, nil
}

func extractNPUIDs(output []byte) ([]string, error) {
	seen := make(map[string]bool)
	var ids []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		tokens := strings.Fields(strings.TrimSpace(parts[1]))
		if len(tokens) < 2 || !isAllDigits(tokens[0]) || isAllDigits(tokens[1]) {
			continue
		}
		if !seen[tokens[0]] {
			seen[tokens[0]] = true
			ids = append(ids, tokens[0])
		}
	}
	return ids, scanner.Err()
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func parseChipSections(output []byte) map[string]map[string]string {
	type kv struct {
		key   string
		value string
	}
	var lines []kv
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "NPU ID" || key == "Chip Count" {
			continue
		}
		lines = append(lines, kv{key, value})
	}

	var chipIndices []int
	for i, line := range lines {
		if line.key == "Chip ID" {
			chipIndices = append(chipIndices, i)
		}
	}
	if len(chipIndices) == 0 {
		return nil
	}

	result := make(map[string]map[string]string)
	if chipIndices[0] == 0 {
		for i, idx := range chipIndices {
			chipID := lines[idx].value
			values := make(map[string]string)
			end := len(lines)
			if i+1 < len(chipIndices) {
				end = chipIndices[i+1]
			}
			for j := idx + 1; j < end; j++ {
				if lines[j].key == "Chip Name" && strings.EqualFold(lines[j].value, "mcu") {
					break
				}
				values[lines[j].key] = lines[j].value
			}
			result[chipID] = values
		}
	} else {
		start := 0
		for _, idx := range chipIndices {
			chipID := lines[idx].value
			values := make(map[string]string)
			for j := start; j < idx; j++ {
				values[lines[j].key] = lines[j].value
			}
			result[chipID] = values
			start = idx + 1
		}
	}
	return result
}

func parseBoardOutput(output []byte) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "NPU ID" || key == "Chip Count" {
			continue
		}
		result[key] = strings.TrimSpace(parts[1])
	}
	return result
}

func (h *npuSMICommand) queryInfo(ctx context.Context, smiPath, npuID, infoType string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, smiPath, "info", "-t", infoType, "-i", npuID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: h.Vendor(), Op: "get " + infoType + " info for NPU " + npuID, Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		opErr := &gpu.OpError{Vendor: h.Vendor(), Op: "get " + infoType + " info for NPU " + npuID, Command: cmd.String(), Err: err}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %w", gpu.ErrUnavailable, opErr)
		}
		return nil, opErr
	}
	return output, nil
}

func (h *npuSMICommand) getBoardInfo(ctx context.Context, smiPath, npuID string) (map[string]string, error) {
	output, err := h.queryInfo(ctx, smiPath, npuID, "board")
	if err != nil {
		return nil, err
	}
	return parseBoardOutput(output), nil
}

func (h *npuSMICommand) getCommonInfo(ctx context.Context, smiPath, npuID string) (map[string]map[string]string, error) {
	output, err := h.queryInfo(ctx, smiPath, npuID, "common")
	if err != nil {
		return nil, err
	}
	return parseChipSections(output), nil
}

func (h *npuSMICommand) getUsagesInfo(ctx context.Context, smiPath, npuID string) (map[string]map[string]string, error) {
	output, err := h.queryInfo(ctx, smiPath, npuID, "usages")
	if err != nil {
		return nil, err
	}
	return parseChipSections(output), nil
}

func (h *npuSMICommand) getProductInfo(ctx context.Context, smiPath, npuID string) (map[string]map[string]string, error) {
	output, err := h.queryInfo(ctx, smiPath, npuID, "product")
	if err != nil {
		return nil, err
	}
	return parseChipSections(output), nil
}

func (h *npuSMICommand) getPowerInfo(ctx context.Context, smiPath, npuID string) (map[string]string, error) {
	output, err := h.queryInfo(ctx, smiPath, npuID, "power")
	if err != nil {
		return nil, err
	}
	return parseBoardOutput(output), nil
}
