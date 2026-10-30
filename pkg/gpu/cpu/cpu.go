package cpu

import (
	"context"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *cpuSmiCommand {
	return &cpuSmiCommand{}
}

type cpuSmiCommand struct{}

func (c *cpuSmiCommand) Load(context.Context) ([]gpu.GPUInfo, error) {
	return []gpu.GPUInfo{}, nil
}

func (c *cpuSmiCommand) Vendor() string {
	return "CPU"
}
