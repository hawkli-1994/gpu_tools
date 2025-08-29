package gpu_test

import (
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

type DriverGetter interface {
	DriverInfo() (gpu.GPUDriverInfo, error)
}
