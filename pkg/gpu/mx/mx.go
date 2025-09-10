package mx

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&mxCommand{})
}

var mxPath = "/usr/bin/mx-smi"

type mxCommand struct {
}

func (m *mxCommand) Load() (*gpu.GPUInfoList, error) {
	output, err := mxCmd()
	if err != nil {
		return nil, err
	}

	return parseMxOutput(output)
}

func (m *mxCommand) Available() bool {
	cmd := exec.Command(mxPath)
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

func (m *mxCommand) Vendor() string {
	return "mx"
}

func mxCmd() (string, error) {
	mx := mxPath
	cmd := exec.Command(mx, "--show-temperature", "--show-usage", "--show-memory")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute mx-smi command: %v", err)
	}
	return string(output), nil

}

func parseMxOutput(output string) (*gpu.GPUInfoList, error) {
	lines := strings.Split(output, "\n")
	gpuList := &gpu.GPUInfoList{
		GPUInfos: make([]gpu.GPUInfo, 0),
	}

	var currentGPU *gpu.GPUInfo
	var gpuCount int

	for _, line := range lines {
		// 查找GPU数量
		if strings.Contains(line, "Attached GPUs") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					gpuCount = count
				}
			}
			continue
		}

		// 查找GPU标识行
		if strings.Contains(line, "GPU#") && strings.Contains(line, "MX") {
			// 如果已经有正在处理的GPU，先保存它
			if currentGPU != nil {
				gpuList.GPUInfos = append(gpuList.GPUInfos, *currentGPU)
				currentGPU = nil
			}

			// 创建新的GPU信息对象
			gpuInfo := gpu.GPUInfo{}

			// 解析GPU编号和型号
			// 格式: GPU#0  MXN260  0000:0f:00.0
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasPrefix(field, "GPU#") {
					// 提取GPU编号
					numStr := strings.TrimPrefix(field, "GPU#")
					if num, err := strconv.Atoi(numStr); err == nil {
						gpuInfo.Num = num
					}
				} else if field == "MXN260" {
					// 设置GPU型号
					gpuInfo.CardModel = "MXN260"
				}
			}

			// 提取PCI Bus信息（最后一个字段）
			if len(fields) > 2 {
				gpuInfo.PCIBus = fields[len(fields)-1]
			}

			currentGPU = &gpuInfo
			continue
		}

		// 如果当前有GPU对象在处理
		if currentGPU != nil {
			// 解析温度
			if strings.Contains(line, "hotspot") && strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					temp := strings.TrimSpace(parts[1])
					temp = strings.TrimSuffix(temp, "°C")
					currentGPU.TemperatureEdge = strings.TrimSpace(temp)
				}
			}

			// 解析显存总量和使用量 (使用vram而不是vis_vram)
			if strings.Contains(line, "vram total") && strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					mem := strings.TrimSpace(parts[1])
					mem = strings.TrimSuffix(mem, "KB")
					if kb, err := strconv.ParseInt(strings.TrimSpace(mem), 10, 64); err == nil {
						// 转换为字节
						currentGPU.VRAMTotalMemory = strconv.FormatInt(kb*1024, 10)
					}
				}
			}

			if strings.Contains(line, "vram used") && strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					mem := strings.TrimSpace(parts[1])
					mem = strings.TrimSuffix(mem, "KB")
					if kb, err := strconv.ParseInt(strings.TrimSpace(mem), 10, 64); err == nil {
						// 转换为字节
						currentGPU.VRAMTotalUsedMemory = strconv.FormatInt(kb*1024, 10)
					}
				}
			}

			// 解析GPU利用率
			if strings.Contains(line, "GPU") && strings.Contains(line, ":") &&
				strings.Contains(line, "%") && !strings.Contains(line, "VPUE") &&
				!strings.Contains(line, "VPUD") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					usage := strings.TrimSpace(parts[1])
					usage = strings.TrimSuffix(usage, "%")
					currentGPU.GPUUse = strings.TrimSpace(usage)
				}
			}
		}
	}

	// 添加最后一个GPU（如果有的话）
	if currentGPU != nil {
		gpuList.GPUInfos = append(gpuList.GPUInfos, *currentGPU)
	}

	// 确保GPU数量与报告的一致
	if gpuCount > 0 && len(gpuList.GPUInfos) > gpuCount {
		gpuList.GPUInfos = gpuList.GPUInfos[:gpuCount]
	}

	return gpuList, nil
}
