package nvidia

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&nvidiaSMICommand{})
}

func New() *nvidiaSMICommand {
	return &nvidiaSMICommand{}
}

type nvidiaSMICommand struct {
}

func (n *nvidiaSMICommand) Load() (*gpu.GPUInfoList, error) {
	cmd := exec.Command("nvidia-smi", "--format=csv,noheader", "--query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu,pci.bus_id")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute nvidia-smi command: %v", err)
	}
	return n.parse(output)
}

func (n *nvidiaSMICommand) Available() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

func (n *nvidiaSMICommand) parse(output []byte) (*gpu.GPUInfoList, error) {
	/*
	   Parse nvidia-smi output example:
	   0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 1309 MiB, 0 %, 41
	   1, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 13625 MiB, 0 %, 39
	*/

	result := &gpu.GPUInfoList{
		GPUInfos: []gpu.GPUInfo{},
	}

	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV output: %v", err)
	}

	for _, row := range records {
		if len(row) < 6 {
			continue
		}

		index := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		memoryTotal := strings.TrimSpace(row[2])
		memoryUsed := strings.TrimSpace(row[3])
		utilizationGPU := strings.TrimSpace(row[4])
		temperatureGPU := strings.TrimSpace(row[5])

		// Parse index
		indexInt, err := strconv.Atoi(index)
		if err != nil {
			continue
		}

		// Parse memory total (convert MiB to bytes)
		memoryTotalParts := strings.Fields(memoryTotal)
		var memoryTotalBytes int64 = 0
		if len(memoryTotalParts) >= 1 {
			if memTotal, err := strconv.ParseInt(memoryTotalParts[0], 10, 64); err == nil {
				memoryTotalBytes = memTotal * 1024 * 1024 // Convert MiB to bytes
			}
		}

		// Parse memory used (convert MiB to bytes)
		memoryUsedParts := strings.Fields(memoryUsed)
		var memoryUsedBytes int64 = 0
		if len(memoryUsedParts) >= 1 {
			if memUsed, err := strconv.ParseInt(memoryUsedParts[0], 10, 64); err == nil {
				memoryUsedBytes = memUsed * 1024 * 1024 // Convert MiB to bytes
			}
		}

		// Parse GPU utilization (remove % sign)
		utilizationGPUParts := strings.Fields(utilizationGPU)
		var utilizationGPUFloat float64 = 0
		if len(utilizationGPUParts) >= 1 {
			if util, err := strconv.ParseFloat(utilizationGPUParts[0], 64); err == nil {
				utilizationGPUFloat = util
			}
		}

		// Parse temperature
		var temperatureFloat float64 = 0
		if temp, err := strconv.ParseFloat(temperatureGPU, 64); err == nil {
			temperatureFloat = temp
		}

		pciBusID := strings.TrimPrefix(strings.TrimSpace(row[6]), "0000")

		device := gpu.GPUInfo{
			Num:                         indexInt,
			DeviceID:                    fmt.Sprintf("%d", indexInt),
			CardModel:                   name,
			CardVendor:                  "NVIDIA",
			CardSeries:                  "NVIDIA",
			VRAMTotalMemory:             fmt.Sprintf("%d", memoryTotalBytes),
			VRAMTotalUsedMemory:         fmt.Sprintf("%d", memoryUsedBytes),
			GPUUse:                      fmt.Sprintf("%.1f", utilizationGPUFloat),
			TemperatureEdge:             fmt.Sprintf("%.1f", temperatureFloat),
			TemperatureJunction:         fmt.Sprintf("%.1f", temperatureFloat),
			TemperatureMemory:           fmt.Sprintf("%.1f", temperatureFloat),
			AverageGraphicsPackagePower: "0",      // Not provided by basic nvidia-smi query
			SerialNumber:                "",       // Not provided by basic nvidia-smi query
			DeviceRev:                   "",       // Not provided by basic nvidia-smi query
			CardSKU:                     "",       // Not provided by basic nvidia-smi query
			PCIBus:                      pciBusID, // Not provided by basic nvidia-smi query
		}

		result.GPUInfos = append(result.GPUInfos, device)
	}

	return result, nil
}

func (n *nvidiaSMICommand) Vendor() string {
	return "NVIDIA"
}

func (n *nvidiaSMICommand) DriverInfo() (gpu.GPUDriverInfo, error) {
	cmd := exec.Command("nvidia-smi", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return gpu.GPUDriverInfo{}, fmt.Errorf("failed to execute nvidia-smi command: %v", err)
	}
	info, err := ParseVersion(string(output))
	if err != nil {
		return gpu.GPUDriverInfo{}, fmt.Errorf("failed to parse version info: %v", err)
	}
	info.Vendor = n.Vendor()
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
