package cpu

import (
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	gpu.Register(&cpuSmiCommand{})
}

type cpuSmiCommand struct {
}

func (c *cpuSmiCommand) Load() (*gpu.GPUInfoList, error) {
	list := &gpu.GPUInfoList{
		GPUInfos: []gpu.GPUInfo{},
	}
	return list, nil
}

func (c *cpuSmiCommand) Available() bool {
	return true
}

func (c *cpuSmiCommand) Vendor() string {
	return "CPU"
}
