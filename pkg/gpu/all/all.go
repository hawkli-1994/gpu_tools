// Package all constructs a registry containing every GPU loader.
package all

import (
	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/amd"
	amdriscv "github.com/hawkli-1994/gpu_tools/pkg/gpu/amd_riscv"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/cpu"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/dl"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/enflame"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/houmoai"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/huawei"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/ix"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/mx"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/nvidia"
	"github.com/hawkli-1994/gpu_tools/pkg/gpu/xdx"
)

func NewRegistry() (*gpu.Registry, error) {
	return gpu.NewRegistry(
		amd.New(),
		amdriscv.New(),
		cpu.New(),
		dl.New(),
		enflame.New(),
		houmoai.New(),
		huawei.New(),
		ix.New(),
		mx.New(),
		nvidia.New(),
		xdx.New(),
	)
}
