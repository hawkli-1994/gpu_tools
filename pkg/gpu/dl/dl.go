package dl

import (
	"encoding/xml"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&dlsmiCommand{})
}

type dlsmiCommand struct {
}

func (d *dlsmiCommand) Load() (*gpu.GPUInfoList, error) {
	output, err := d.query()
	if err != nil {
		return nil, err
	}
	return parseDLSMIOutput(output)
}

func (d *dlsmiCommand) query() ([]byte, error) {
	candidates := [][]string{
		{"dlsmi", "query", "--xml-format"},
		{"/usr/bin/dlsmi", "query", "--xml-format"},
		{"/usr/local/bin/dlsmi", "query", "--xml-format"},
	}

	var lastErr error
	for _, args := range candidates {
		cmd := exec.Command(args[0], args[1:]...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			return output, nil
		}
		lastErr = fmt.Errorf("dlsmi (%s) failed: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("dlsmi command not found")
	}
	return nil, lastErr
}

func (d *dlsmiCommand) Available() bool {
	paths := []string{
		"dlsmi",
		"/usr/bin/dlsmi",
		"/usr/local/bin/dlsmi",
	}

	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return true
		}
	}
	return false
}

func (d *dlsmiCommand) Vendor() string {
	return "Denglin"
}

type dlsmiLog struct {
	GPUs []dlsmiGPU `xml:"gpu"`
}

type dlsmiGPU struct {
	ID                  string             `xml:"id,attr"`
	ProductName         string             `xml:"product_name"`
	ProductBrand        string             `xml:"product_brand"`
	ProductArchitecture string             `xml:"product_architecture"`
	SerialNumber        string             `xml:"serial_number"`
	FirmwareVersion     string             `xml:"fw_version"`
	BoardPartNumber     string             `xml:"board_part_number"`
	PCI                 dlsmiPCISection    `xml:"pci"`
	MemoryUsage         dlsmiMemorySection `xml:"memory_usage"`
	Utilization         dlsmiUtilization   `xml:"utilization"`
	Temperature         dlsmiTemperature   `xml:"temperature"`
	PowerReadings       dlsmiPowerReadings `xml:"power_readings"`
}

type dlsmiPCISection struct {
	Domain      string `xml:"domain"`
	Bus         string `xml:"bus"`
	Device      string `xml:"device"`
	BusID       string `xml:"bus_id"`
	DeviceID    string `xml:"device_id"`
	SubSystemID string `xml:"sub_system_id"`
}

type dlsmiMemorySection struct {
	Total string `xml:"total"`
	Used  string `xml:"used"`
	Free  string `xml:"free"`
}

type dlsmiUtilization struct {
	GPU string `xml:"gpu"`
}

type dlsmiTemperature struct {
	GPUCurrent    string `xml:"gpu_current_temp"`
	GPUSlowdown   string `xml:"gpu_slowdown_temp"`
	GPUShutdown   string `xml:"gpu_shutdown_temp"`
	GPUMaxOper    string `xml:"gpu_max_operating_temp"`
	MemoryCurrent string `xml:"memory_current_temp"`
	MemoryMax     string `xml:"memory_max_operating_temp"`
}

type dlsmiPowerReadings struct {
	PowerDraw string `xml:"power_draw"`
}

func parseDLSMIOutput(output []byte) (*gpu.GPUInfoList, error) {
	var log dlsmiLog
	if err := xml.Unmarshal(output, &log); err != nil {
		return nil, fmt.Errorf("parse dlsmi output: %w", err)
	}

	gpuInfos := make([]gpu.GPUInfo, 0, len(log.GPUs))
	for idx, gpuNode := range log.GPUs {
		info := gpu.GPUInfo{
			Num:                         idx,
			DeviceID:                    strings.TrimSpace(gpuNode.PCI.DeviceID),
			DeviceRev:                   strings.TrimSpace(gpuNode.FirmwareVersion),
			SerialNumber:                strings.TrimSpace(gpuNode.SerialNumber),
			CardSeries:                  strings.TrimSpace(gpuNode.ProductArchitecture),
			CardModel:                   strings.TrimSpace(gpuNode.ProductName),
			CardVendor:                  resolveVendor(gpuNode.ProductBrand),
			CardSKU:                     strings.TrimSpace(gpuNode.BoardPartNumber),
			PCIBus:                      resolveBusID(gpuNode),
			VRAMTotalMemory:             convertSizeToBytes(gpuNode.MemoryUsage.Total),
			VRAMTotalUsedMemory:         convertSizeToBytes(gpuNode.MemoryUsage.Used),
			GPUUse:                      parseNumericField(gpuNode.Utilization.GPU),
			TemperatureEdge:             parseNumericField(gpuNode.Temperature.GPUCurrent),
			TemperatureJunction:         parseNumericField(gpuNode.Temperature.GPUSlowdown),
			TemperatureMemory:           parseNumericField(gpuNode.Temperature.MemoryCurrent),
			AverageGraphicsPackagePower: parseNumericField(gpuNode.PowerReadings.PowerDraw),
		}

		if info.DeviceID == "" {
			info.DeviceID = strings.TrimSpace(gpuNode.ID)
		}

		gpuInfos = append(gpuInfos, info)
	}

	return &gpu.GPUInfoList{GPUInfos: gpuInfos}, nil
}

func resolveVendor(productBrand string) string {
	if trimmed := strings.TrimSpace(productBrand); trimmed != "" && !strings.EqualFold(trimmed, "N/A") {
		return trimmed
	}
	return "Denglin"
}

func resolveBusID(g dlsmiGPU) string {
	if bid := strings.TrimSpace(g.PCI.BusID); bid != "" {
		return bid
	}
	return strings.TrimSpace(g.ID)
}

func parseNumericField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return "0"
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "0"
	}

	numStr := strings.Trim(fields[0], "+")
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return "0"
	}

	if math.Abs(num-math.Round(num)) < 1e-6 {
		return strconv.FormatInt(int64(math.Round(num)), 10)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", num), "0"), ".")
}

func convertSizeToBytes(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return "0"
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "0"
	}

	num, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "0"
	}

	unit := "B"
	if len(fields) > 1 {
		unit = strings.ToUpper(strings.TrimSuffix(fields[1], "s"))
	}

	switch unit {
	case "B":
	case "KB", "KIB":
		num *= 1024
	case "MB", "MIB":
		num *= 1024 * 1024
	case "GB", "GIB":
		num *= 1024 * 1024 * 1024
	case "TB", "TIB":
		num *= 1024 * 1024 * 1024 * 1024
	default:
		// best effort for strings like "MiB"
		if strings.Contains(unit, "IB") {
			if strings.HasPrefix(unit, "M") {
				num *= 1024 * 1024
			} else if strings.HasPrefix(unit, "G") {
				num *= 1024 * 1024 * 1024
			} else if strings.HasPrefix(unit, "K") {
				num *= 1024
			}
		}
	}

	return strconv.FormatInt(int64(math.Round(num)), 10)
}
