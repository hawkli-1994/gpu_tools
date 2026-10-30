package gpu_test

import (
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/nvidia"
)

var _ gpu.Loader = nvidia.New()
var _ gpu.DriverLoader = nvidia.New()
