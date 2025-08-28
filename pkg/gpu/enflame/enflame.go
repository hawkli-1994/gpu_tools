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
	// root@node1:~# efsmi -q -d TEMP,MEMORY,USAGE
	// ----------------------------------------------------------------------------
	// ------------------- Enflame System Management Interface --------------------
	// ---------- Enflame Tech, All Rights Reserved. 2024 Copyright (C) -----------
	// ----------------------------------------------------------------------------

	// DEV ID 0
	// 	Device Mem Info
	// 		Mem Size                : 42976 MiB
	// 		Mem Usage               : 1129 MiB
	// 		Mem Ecc                 : enable
	// 	Temperature Info
	// 		GCU Temp                : 34 C
	// 	Device Usage Info
	// 		GCU Usage               : 0.0 %
	// DEV ID 1
	// 	Device Mem Info
	// 		Mem Size                : 42976 MiB
	// 		Mem Usage               : 1129 MiB
	// 		Mem Ecc                 : enable
	// 	Temperature Info
	// 		GCU Temp                : 34 C
	// 	Device Usage Info
	// 		GCU Usage               : 0.0 %

	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Split(bufio.ScanLines)

	result := &gpu.GPUInfoList{
		GPUInfos: []gpu.GPUInfo{},
	}

	var currentGPU *gpu.GPUInfo
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
				VRAMTotalMemory:     "0",
				VRAMTotalUsedMemory: "0",
				GPUUse:              "0",
				PCIBus:              "",
			}
			num++
		} else if currentGPU != nil {
			// 提取值
			matches := valueRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				value := matches[1]

				switch {
				case strings.Contains(line, "Mem Size") || strings.Contains(line, "Total Size"):
					// 从"42976 MiB"转换为字节数
					parts := strings.Split(value, " ")
					if len(parts) >= 2 && parts[1] == "MiB" {
						if size, err := strconv.ParseFloat(parts[0], 64); err == nil {
							// 转换MiB到B (1 MiB = 1024*1024 B)
							size = size * 1024 * 1024
							currentGPU.VRAMTotalMemory = fmt.Sprintf("%.0f", size)
						}
					}

				case strings.Contains(line, "Mem Usage") || strings.Contains(line, "Used Size"):
					// 从"1129 MiB"转换为字节数
					parts := strings.Split(value, " ")
					if len(parts) >= 2 && parts[1] == "MiB" {
						if size, err := strconv.ParseFloat(parts[0], 64); err == nil {
							// 转换MiB到B (1 MiB = 1024*1024 B)
							size = size * 1024 * 1024
							currentGPU.VRAMTotalUsedMemory = fmt.Sprintf("%.0f", size)
						}
					}

				case strings.Contains(line, "GCU Temp"):
					// 从"34 C"提取温度
					parts := strings.Split(value, " ")
					if len(parts) >= 1 {
						currentGPU.TemperatureMemory = parts[0]
						currentGPU.TemperatureEdge = parts[0]
						currentGPU.TemperatureJunction = parts[0]
					}

				case strings.Contains(line, "GCU Usage"):
					// 从"0.0 %"提取使用率
					parts := strings.Split(value, " ")
					if len(parts) >= 1 {
						currentGPU.GPUUse = parts[0]
					}
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
