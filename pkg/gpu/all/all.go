// Package all provides a convenient way to import all GPU provider packages
// so that they self-register with the gpu registry.
//
// Use a blank import in your main package or initialization code:
//
//	import _ "github.com/hawkli-1994/gpu_tools/pkg/gpu/all"
package all

import (
	// Ensure all GPU providers register themselves.
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/amd"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/amd_riscv"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/cpu"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/dl"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/enflame"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/huawei"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/ix"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/mx"
	_ "github.com/hawkli-1994/gpu_tools/pkg/gpu/nvidia"
)
