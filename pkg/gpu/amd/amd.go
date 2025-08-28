package amd

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&rocmSMICommand{})
}

type rocmSMICommand struct {
}

func (r *rocmSMICommand) Load() (*gpu.GPUInfoList, error) {
	// rocm-smi -i --showmeminfo vram --showpower --showserial --showuse --showtemp --showproductname --json
	smiCmd := exec.Command("/usr/bin/rocm-smi", "-i", "--showmeminfo", "vram", "--showpower", "--showserial", "--showuse", "--showtemp", "--showproductname", "--showbus", "--json")
	smiCmd.Env = append(os.Environ(),
		"PATH=/usr/bin:/usr/local/bin:/bin:/usr/sbin:/sbin", // 确保 python3 在 PATH 里
		// 你还可以加其它环境变量，比如 PYTHONPATH
		// "PYTHONPATH=/your/python/site-packages",
	)
	output, err := smiCmd.Output()
	if err != nil {
		return nil, err
	}

	return r.parse(output)
}

func (r *rocmSMICommand) parse(output []byte) (*gpu.GPUInfoList, error) {
	var gpuInfoList gpu.GPUInfoList
	// {
	// 	"card0": {
	// 		"Device ID": "0x747e",
	// 		"Device Rev": "0xc8",
	// 		"Temperature (Sensor edge) (C)": "36.0",
	// 		"Temperature (Sensor junction) (C)": "41.0",
	// 		"Temperature (Sensor memory) (C)": "44.0",
	// 		"Average Graphics Package Power (W)": "4.0",
	// 		"GPU use (%)": "0",
	// 		"Serial Number": "5c88007d760374f3",
	// 		"VRAM Total Memory (B)": "17163091968",
	// 		"VRAM Total Used Memory (B)": "283090944",
	// 		"Card series": "0x747e",
	// 		"Card model": "0x7801",
	// 		"Card vendor": "Advanced Micro Devices, Inc. [AMD/ATI]",
	// 		"Card SKU": "EXT94393"
	// 	}
	// }
	var jsonData map[string]gpu.GPUInfo
	if err := json.Unmarshal(output, &jsonData); err != nil {
		return nil, err
	}
	gpuList := make([]gpu.GPUInfo, 0, len(jsonData))
	for cardNum, gpuInfo := range jsonData {

		num := strings.TrimPrefix(cardNum, "card")
		numInt, err := strconv.Atoi(num)
		if err != nil {
			return nil, err
		}
		gpuInfo.Num = numInt
		gpuList = append(gpuList, gpuInfo)
	}
	gpuInfoList.GPUInfos = gpuList

	return &gpuInfoList, nil
}

func (r *rocmSMICommand) Available() bool {
	smiCmd := exec.Command("/usr/bin/rocm-smi")
	smiCmd.Env = append(os.Environ(),
		"PATH=/usr/bin:/usr/local/bin:/bin:/usr/sbin:/sbin", // 确保 python3 在 PATH 里
		// 你还可以加其它环境变量，比如 PYTHONPATH
		// "PYTHONPATH=/your/python/site-packages",
	)
	if err := smiCmd.Run(); err != nil {
		return false
	}

	// rocminfoCmd := exec.Command("rocminfo")

	// if err := rocminfoCmd.Run(); err != nil {
	// 	return false
	// }

	return true
}

func (r *rocmSMICommand) Vendor() string {
	return "AMD"
}
