package huawei

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&npuSMICommand{})
}

func New() *npuSMICommand {
	return &npuSMICommand{}
}

type npuSMICommand struct {
	smiPath string
}

func (h *npuSMICommand) Available() bool {
	if h.smiPath != "" {
		return true
	}
	if _, err := exec.LookPath("npu-smi"); err == nil {
		h.smiPath = "npu-smi"
		return true
	}
	if _, err := exec.LookPath("/usr/local/sbin/npu-smi"); err == nil {
		h.smiPath = "/usr/local/sbin/npu-smi"
		return true
	}
	return false
}

func (h *npuSMICommand) Vendor() string {
	return "Huawei"
}

func (h *npuSMICommand) Load() (*gpu.GPUInfoList, error) {
	if h.smiPath == "" {
		h.Available()
	}
	if h.smiPath == "" {
		return nil, fmt.Errorf("npu-smi command not found")
	}
	cmd := exec.Command(h.smiPath, "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute npu-smi info: %v", err)
	}

	npuIDs, err := extractNPUIDs(output)
	if err != nil {
		return nil, fmt.Errorf("failed to extract NPU IDs: %v", err)
	}

	result := &gpu.GPUInfoList{GPUInfos: []gpu.GPUInfo{}}
	globalNum := 0

	for _, npuID := range npuIDs {
		boardInfo, _ := h.getBoardInfo(npuID)
		commonInfo, _ := h.getCommonInfo(npuID)
		usagesInfo, _ := h.getUsagesInfo(npuID)
		productInfo, _ := h.getProductInfo(npuID)
		powerInfo, _ := h.getPowerInfo(npuID)

		infos, nextNum := buildGPUInfoList(npuID, boardInfo, commonInfo, usagesInfo, productInfo, powerInfo, globalNum)
		result.GPUInfos = append(result.GPUInfos, infos...)
		globalNum = nextNum
	}

	return result, nil
}

func buildGPUInfoList(
	npuID string,
	board map[string]string,
	common map[string]map[string]string,
	usages map[string]map[string]string,
	product map[string]map[string]string,
	power map[string]string,
	globalNum int,
) ([]gpu.GPUInfo, int) {
	powerVal := "0"
	if power != nil {
		if v, ok := power["Power Dissipation(W)"]; ok && v != "" && v != "NA" {
			powerVal = v
		}
	}

	serialNumber := ""
	pciBus := ""
	if board != nil {
		serialNumber = board["Serial Number"]
		pciBus = board["PCIe Bus Info"]
	}

	// Collect all known chip IDs
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
		return nil, globalNum
	}

	var chipIDs []string
	for chipID := range chipIDSet {
		chipIDs = append(chipIDs, chipID)
	}
	sort.Strings(chipIDs)

	var infos []gpu.GPUInfo
	for _, chipID := range chipIDs {
		chipCommon := common[chipID]
		chipUsages := usages[chipID]
		chipProduct := product[chipID]

		// Temperature
		temp := "0"
		if chipCommon != nil {
			if v, ok := chipCommon["Temperature(C)"]; ok && v != "" && v != "NA" {
				temp = v
			}
		}

		// GPU Use: prefer usages Aicore, fallback to common Aicore
		gpuUse := "0"
		if chipUsages != nil {
			if v, ok := chipUsages["Aicore Usage Rate(%)"]; ok && v != "" && v != "NA" {
				gpuUse = v
			}
		}
		if gpuUse == "0" && chipCommon != nil {
			if v, ok := chipCommon["Aicore Usage Rate(%)"]; ok && v != "" && v != "NA" {
				gpuUse = v
			}
		}

		// VRAM: total from usages DDR Capacity, used computed from DDR Usage Rate
		var vramTotalBytes int64
		var vramUsedBytes int64
		if chipUsages != nil {
			if capStr, ok := chipUsages["DDR Capacity(MB)"]; ok && capStr != "" {
				if capMB, err := strconv.ParseFloat(capStr, 64); err == nil {
					vramTotalBytes = int64(capMB * 1024 * 1024)
				}
			}
			if rateStr, ok := chipUsages["DDR Usage Rate(%)"]; ok && rateStr != "" {
				if rate, err := strconv.ParseFloat(rateStr, 64); err == nil && vramTotalBytes > 0 {
					vramUsedBytes = int64(float64(vramTotalBytes) * rate / 100.0)
				}
			}
		}

		// Model
		model := ""
		if chipProduct != nil {
			if v, ok := chipProduct["Product Type"]; ok && v != "" && v != "NA" {
				model = v
			}
		}

		infos = append(infos, gpu.GPUInfo{
			Num:                         globalNum,
			DeviceID:                    chipID,
			CardVendor:                  "Huawei",
			CardSeries:                  "Ascend",
			CardModel:                   model,
			TemperatureEdge:             temp,
			TemperatureJunction:         temp,
			TemperatureMemory:           temp,
			GPUUse:                      gpuUse,
			VRAMTotalMemory:             fmt.Sprintf("%d", vramTotalBytes),
			VRAMTotalUsedMemory:         fmt.Sprintf("%d", vramUsedBytes),
			AverageGraphicsPackagePower: powerVal,
			SerialNumber:                serialNumber,
			PCIBus:                      pciBus,
		})
		globalNum++
	}

	return infos, globalNum
}

// extractNPUIDs extracts unique NPU IDs from npu-smi info table output.
// It distinguishes NPU rows (e.g. "2944    310P3") from Chip rows (e.g. "0       0")
// by checking whether the second field contains non-digit characters.
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
		firstField := strings.TrimSpace(parts[1])
		if firstField == "" {
			continue
		}
		tokens := strings.Fields(firstField)
		if len(tokens) < 2 {
			continue
		}
		// First token must be all digits (NPU ID)
		if !isAllDigits(tokens[0]) {
			continue
		}
		// Second token must contain at least one non-digit character
		// (e.g. "310P3", "910B") to distinguish from Chip rows like "0       0"
		if isAllDigits(tokens[1]) {
			continue
		}
		id := tokens[0]
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
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

// parseChipSections parses key:value output grouped by Chip ID.
// It auto-detects whether Chip ID appears at the start or end of each section.
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
	isForward := chipIndices[0] == 0

	if isForward {
		for i, idx := range chipIndices {
			chipID := lines[idx].value
			m := make(map[string]string)
			end := len(lines)
			if i+1 < len(chipIndices) {
				end = chipIndices[i+1]
			}
			for j := idx + 1; j < end; j++ {
				if lines[j].key == "Chip Name" && strings.EqualFold(lines[j].value, "mcu") {
					break
				}
				m[lines[j].key] = lines[j].value
			}
			result[chipID] = m
		}
	} else {
		start := 0
		for _, idx := range chipIndices {
			chipID := lines[idx].value
			m := make(map[string]string)
			for j := start; j < idx; j++ {
				m[lines[j].key] = lines[j].value
			}
			result[chipID] = m
			start = idx + 1
		}
	}

	return result
}

// parseBoardOutput parses board-level key:value output (no Chip ID grouping).
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
		value := strings.TrimSpace(parts[1])
		if key == "NPU ID" || key == "Chip Count" {
			continue
		}
		result[key] = value
	}
	return result
}

func (h *npuSMICommand) getBoardInfo(npuID string) (map[string]string, error) {
	cmd := exec.Command(h.smiPath, "info", "-t", "board", "-i", npuID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get board info for NPU %s: %v", npuID, err)
	}
	return parseBoardOutput(output), nil
}

func (h *npuSMICommand) getCommonInfo(npuID string) (map[string]map[string]string, error) {
	cmd := exec.Command(h.smiPath, "info", "-t", "common", "-i", npuID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get common info for NPU %s: %v", npuID, err)
	}
	return parseChipSections(output), nil
}

func (h *npuSMICommand) getUsagesInfo(npuID string) (map[string]map[string]string, error) {
	cmd := exec.Command(h.smiPath, "info", "-t", "usages", "-i", npuID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get usages info for NPU %s: %v", npuID, err)
	}
	return parseChipSections(output), nil
}

func (h *npuSMICommand) getProductInfo(npuID string) (map[string]map[string]string, error) {
	cmd := exec.Command(h.smiPath, "info", "-t", "product", "-i", npuID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get product info for NPU %s: %v", npuID, err)
	}
	return parseChipSections(output), nil
}

func (h *npuSMICommand) getPowerInfo(npuID string) (map[string]string, error) {
	cmd := exec.Command(h.smiPath, "info", "-t", "power", "-i", npuID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get power info for NPU %s: %v", npuID, err)
	}
	return parseBoardOutput(output), nil
}
