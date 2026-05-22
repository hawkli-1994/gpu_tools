package xdx

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
	gpu.Register(&xdxSMICommand{})
}

type xdxSMICommand struct {
}

func (x *xdxSMICommand) Load() (*gpu.GPUInfoList, error) {
	cmd := exec.Command("xdxsmi", "out")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute xdxsmi command: %v", err)
	}
	return x.parse(output)
}

func (x *xdxSMICommand) Available() bool {
	_, err := exec.LookPath("xdxsmi")
	return err == nil
}

func (x *xdxSMICommand) parse(output []byte) (*gpu.GPUInfoList, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Split(bufio.ScanLines)

	result := &gpu.GPUInfoList{
		GPUInfos: []gpu.GPUInfo{},
	}

	var currentGPU *gpu.GPUInfo
	var currentSection string
	kvRegex := regexp.MustCompile(`^\s*(.+?)\s*:\s+(.+)$`)
	gpuRegex := regexp.MustCompile(`^\s*GPU\s+(\d+)\s*$`)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "======") {
			continue
		}

		gpuMatches := gpuRegex.FindStringSubmatch(line)
		if gpuMatches != nil {
			if currentGPU != nil {
				result.GPUInfos = append(result.GPUInfos, *currentGPU)
			}

			gpuIndex, _ := strconv.Atoi(gpuMatches[1])
			currentGPU = &gpu.GPUInfo{
				Num:                         gpuIndex,
				DeviceID:                    gpuMatches[1],
				CardVendor:                  "XDX",
				CardModel:                   "XDX GPU",
				CardSeries:                  "XDX",
				TemperatureMemory:           "0",
				TemperatureEdge:             "0",
				TemperatureJunction:         "0",
				VRAMTotalMemory:             "0",
				VRAMTotalUsedMemory:         "0",
				GPUUse:                      "0",
				AverageGraphicsPackagePower: "0",
				PCIBus:                      "",
			}
			currentSection = ""
			continue
		}

		if currentGPU == nil {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)

		if strings.Contains(line, ":") {
			kvMatches := kvRegex.FindStringSubmatch(line)
			if kvMatches == nil {
				continue
			}
			key := strings.TrimSpace(kvMatches[1])
			value := strings.TrimSpace(kvMatches[2])

			if indent <= 8 {
				currentSection = ""
				switch key {
				case "Product Name":
					currentGPU.CardModel = value
				case "Product Architecture":
					currentGPU.CardSeries = value
				case "Temperature":
					parts := strings.Split(value, " ")
					if len(parts) >= 1 {
						if temp, err := strconv.ParseFloat(parts[0], 64); err == nil {
							tempStr := fmt.Sprintf("%.1f", temp)
							currentGPU.TemperatureMemory = tempStr
							currentGPU.TemperatureEdge = tempStr
							currentGPU.TemperatureJunction = tempStr
						}
					}
				}
			} else {
				switch currentSection {
				case "pci":
					if key == "Bus Id" {
						currentGPU.PCIBus = value
					}
				case "fb_memory":
					if key == "Total" {
						parts := strings.Split(value, " ")
						if len(parts) >= 2 && parts[1] == "MiB" {
							if size, err := strconv.ParseFloat(parts[0], 64); err == nil {
								size = size * 1024 * 1024
								currentGPU.VRAMTotalMemory = fmt.Sprintf("%.0f", size)
							}
						}
					}
				case "power":
					if key == "Pwr Total" {
						parts := strings.Split(value, " ")
						if len(parts) >= 1 {
							currentGPU.AverageGraphicsPackagePower = parts[0]
						}
					}
				}
			}
		} else {
			switch trimmed {
			case "PCI":
				currentSection = "pci"
			case "FB Memory Usage":
				currentSection = "fb_memory"
			case "Power":
				currentSection = "power"
			case "Clocks":
				currentSection = "clocks"
			}
		}
	}

	if currentGPU != nil {
		result.GPUInfos = append(result.GPUInfos, *currentGPU)
	}

	return result, nil
}

func (x *xdxSMICommand) Vendor() string {
	return "XDX"
}
