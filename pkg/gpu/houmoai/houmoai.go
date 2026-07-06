package houmoai

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&houmoaiSMICommand{})
}

func New() *houmoaiSMICommand {
	return &houmoaiSMICommand{}
}

type houmoaiSMICommand struct{}

func (h *houmoaiSMICommand) Load() (*gpu.GPUInfoList, error) {
	cmd := exec.Command("hm_smi", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute hm_smi command: %v", err)
	}
	return h.parse(output)
}

func (h *houmoaiSMICommand) Available() bool {
	_, err := exec.LookPath("hm_smi")
	return err == nil
}

func (h *houmoaiSMICommand) Vendor() string {
	return "Houmo"
}

func (h *houmoaiSMICommand) parse(output []byte) (*gpu.GPUInfoList, error) {
	result := &gpu.GPUInfoList{
		GPUInfos: []gpu.GPUInfo{},
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	deviceRegex := regexp.MustCompile(`^\s*device(\d+)\s+detail\s+infos\s*$`)
	kvRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*:\s*(.*)$`)

	var current map[string]string
	num := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if matches := deviceRegex.FindStringSubmatch(line); matches != nil {
			if current != nil {
				result.GPUInfos = append(result.GPUInfos, h.buildGPUInfo(num, current))
				num++
			}
			current = make(map[string]string)
			continue
		}

		if current == nil {
			continue
		}

		if matches := kvRegex.FindStringSubmatch(line); matches != nil {
			key := strings.TrimSpace(matches[1])
			value := strings.TrimSpace(matches[2])
			current[key] = value
		}
	}

	if current != nil {
		result.GPUInfos = append(result.GPUInfos, h.buildGPUInfo(num, current))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan hm_smi output: %v", err)
	}

	return result, nil
}

func (h *houmoaiSMICommand) buildGPUInfo(num int, fields map[string]string) gpu.GPUInfo {
	totalMB := parseFloatMB(fields["DDR_Memory_Total"])
	freeMB := parseFloatMB(fields["DDR_Memory_Free"])
	usedMB := totalMB - freeMB
	if usedMB < 0 {
		usedMB = 0
	}

	avgTemp := averageDDRTemp(fields)

	return gpu.GPUInfo{
		Num:                         num,
		DeviceID:                    fields["Dev"],
		CardVendor:                  fields["Vendor"],
		CardSeries:                  fields["Vendor"],
		CardModel:                   fields["Model"],
		CardSKU:                     fields["PN"],
		SerialNumber:                fields["SN"],
		DeviceRev:                   fields["Firmware_Version"],
		PCIBus:                      fields["BDF"],
		GPUUse:                      firstFloat(fields["Average_Util"]),
		VRAMTotalMemory:             mbToBytesString(totalMB),
		VRAMTotalUsedMemory:         mbToBytesString(usedMB),
		TemperatureEdge:             firstFloat(fields["Core0"]),
		TemperatureJunction:         firstFloat(fields["Core1"]),
		TemperatureMemory:           avgTemp,
		AverageGraphicsPackagePower: firstFloat(fields["Board_Power"]),
	}
}

func parseFloatMB(s string) float64 {
	re := regexp.MustCompile(`([\d.]+)\s*MB`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0
	}
	v, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	return v
}

func mbToBytesString(mb float64) string {
	return fmt.Sprintf("%.0f", mb*1024*1024)
}

func firstFloat(s string) string {
	re := regexp.MustCompile(`([\d.]+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "0"
	}
	v, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return "0"
	}
	return fmt.Sprintf("%.1f", v)
}

func averageDDRTemp(fields map[string]string) string {
	keys := []string{"DDR0", "DDR2", "DDR4", "DDR5"}
	var sum, count float64
	re := regexp.MustCompile(`([\d.]+)`)
	for _, k := range keys {
		matches := re.FindStringSubmatch(fields[k])
		if len(matches) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			continue
		}
		sum += v
		count++
	}
	if count == 0 {
		return "0"
	}
	return fmt.Sprintf("%.1f", sum/count)
}
