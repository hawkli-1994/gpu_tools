package gpu_test

import (
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/nvidia"
)

var _ DriverGetter = nvidia.New()

type DriverGetter interface {
	DriverInfo() (gpu.GPUDriverInfo, error)
}

