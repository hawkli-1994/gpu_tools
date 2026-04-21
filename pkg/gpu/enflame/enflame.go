package enflame

import (
	"bufio"
	"bytes"
	"fmt"

	// "os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&enflameSMICommand{})
}

type enflameSMICommand struct {
}

func (e *enflameSMICommand) Load() (*gpu.GPUInfoList, error) {
	// 执行efsmi命令获取GPU信息

	cmd := exec.Command("efsmi", "-q", "-d", "TEMP,MEMORY,USAGE,PCIE")
	output, err := cmd.CombinedOutput()
	if err != nil {

		cmd = exec.Command("/usr/bin/efsmi", "-q", "-d", "TEMP,MEMORY,USAGE,PCIE")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to execute efsmi command: %v", err)
		} else {
			return e.parse(output)
		}
	}
	return e.parse(output)
}

func (e *enflameSMICommand) Available() bool {
	// 检查efsmi命令是否可用
	_, err := exec.LookPath("efsmi")
	if err != nil {
		_, err = exec.LookPath("/usr/bin/efsmi")
		return err == nil
	} else {
		return true
	}
}

func (e *enflameSMICommand) parse(output []byte) (*gpu.GPUInfoList, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Split(bufio.ScanLines)

	result := &gpu.GPUInfoList{
		GPUInfos: []gpu.GPUInfo{},
	}

	var currentGPU *gpu.GPUInfo
	var currentSection string
	valueRegex := regexp.MustCompile(`:\s+(.+)`)
	num := 0
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// 检测新的GPU设备
		if strings.HasPrefix(line, "DEV ID") {
			if currentGPU != nil {
				result.GPUInfos = append(result.GPUInfos, *currentGPU)
			}

			deviceID := strings.TrimSpace(line[len("DEV ID"):])
			currentGPU = &gpu.GPUInfo{
				Num:                 num,
				DeviceID:            deviceID,
				CardVendor:          "Enflame",
				CardModel:           "Enflame GCU",
				TemperatureMemory:   "0",
				TemperatureEdge:     "0",
				TemperatureJunction: "0",
				VRAMTotalMemory:     "0",
				VRAMTotalUsedMemory: "0",
				GPUUse:              "0",
				PCIBus:              "",
			}
			currentSection = ""
			num++
			continue
		}

		if currentGPU == nil {
			continue
		}

		// 检测 section 切换
		switch {
		case strings.HasPrefix(line, "PCIe Info"):
			currentSection = "pcie"
			continue
		case strings.HasPrefix(line, "Device Mem Info"):
			currentSection = "device_mem"
			continue
		case strings.HasPrefix(line, "BAR1 Mem Info"):
			currentSection = "bar1"
			continue
		case strings.HasPrefix(line, "BAR2 Mem Info"):
			currentSection = "bar2"
			continue
		case strings.HasPrefix(line, "Temperature Info"):
			currentSection = "temperature"
			continue
		case strings.HasPrefix(line, "Device Usage Info"):
			currentSection = "usage"
			continue
		case strings.HasPrefix(line, "Link Info"):
			currentSection = "link"
			continue
		}

		matches := valueRegex.FindStringSubmatch(line)
		if len(matches) <= 1 {
			continue
		}
		value := matches[1]

		switch currentSection {
		case "device_mem":
			switch {
			case strings.Contains(line, "Mem Size") || strings.Contains(line, "Total Size"):
				parts := strings.Split(value, " ")
				if len(parts) >= 2 && parts[1] == "MiB" {
					if size, err := strconv.ParseFloat(parts[0], 64); err == nil {
						size = size * 1024 * 1024
						currentGPU.VRAMTotalMemory = fmt.Sprintf("%.0f", size)
					}
				}
			case strings.Contains(line, "Mem Usage") || strings.Contains(line, "Used Size"):
				parts := strings.Split(value, " ")
				if len(parts) >= 2 && parts[1] == "MiB" {
					if size, err := strconv.ParseFloat(parts[0], 64); err == nil {
						size = size * 1024 * 1024
						currentGPU.VRAMTotalUsedMemory = fmt.Sprintf("%.0f", size)
					}
				}
			}
		case "temperature":
			if strings.Contains(line, "GCU Temp") {
				parts := strings.Split(value, " ")
				if len(parts) >= 1 {
					currentGPU.TemperatureMemory = parts[0]
					currentGPU.TemperatureEdge = parts[0]
					currentGPU.TemperatureJunction = parts[0]
				}
			}
		case "usage":
			if strings.Contains(line, "GCU Usage") {
				parts := strings.Split(value, " ")
				if len(parts) >= 1 {
					currentGPU.GPUUse = parts[0]
				}
			}
		case "pcie":
			switch {
			case strings.Contains(line, "Domain"):
				currentGPU.PCIBus = strings.TrimSpace(value)
			case strings.Contains(line, "Bus"):
				currentGPU.PCIBus = fmt.Sprintf("%s:%s", currentGPU.PCIBus, strings.TrimSpace(value))
			case strings.Contains(line, "Dev  "):
				currentGPU.PCIBus = fmt.Sprintf("%s:%s", currentGPU.PCIBus, strings.TrimSpace(value))
			case strings.Contains(line, "Func"):
				currentGPU.PCIBus = fmt.Sprintf("%s.%s", currentGPU.PCIBus, strings.TrimSpace(value))
			}
		}
	}

	// 添加最后一个处理的GPU
	if currentGPU != nil {
		result.GPUInfos = append(result.GPUInfos, *currentGPU)
	}

	return result, nil
}

func (e *enflameSMICommand) Vendor() string {
	return "Enflame"
}
