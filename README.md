# GPU Tools

A comprehensive Go library for monitoring and managing GPU information across multiple vendors including NVIDIA, AMD, Enflame, and other accelerators.

## Features

- **Multi-vendor Support**: Unified interface for different GPU manufacturers
  - NVIDIA GPUs (via nvidia-smi)
  - AMD GPUs (via rocm-smi) 
  - Enflame GCU (via efsmi)
  - Intel GPUs (via ix)
  - CPU information
  - AMD RISC-V accelerators

- **Comprehensive GPU Information**:
  - Device identification (ID, model, vendor, serial number)
  - Performance metrics (temperature, usage, power consumption)
  - Memory information (total VRAM, used memory)
  - PCI bus information

- **Plugin Architecture**: Extensible design allowing easy addition of new GPU vendors

## Installation

```bash
go get github.com/hawkli-1994/gpu_tools
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func main() {
    // Get all available GPU loaders
    loaders := gpu.GetAllGPULoaders()
    
    for _, loader := range loaders {
        if loader.Available() {
            fmt.Printf("Found %s GPU loader\n", loader.Vendor())
            
            // Load GPU information
            info, err := loader.Load()
            if err != nil {
                log.Printf("Error loading %s GPU info: %v", loader.Vendor(), err)
                continue
            }
            
            // Print GPU information
            for _, gpuInfo := range info.GPUInfos {
                fmt.Printf("GPU %d: %s %s\n", gpuInfo.Num, gpuInfo.CardVendor, gpuInfo.CardModel)
                fmt.Printf("  Temperature: %s°C\n", gpuInfo.TemperatureEdge)
                fmt.Printf("  GPU Usage: %s%%\n", gpuInfo.GPUUse)
                fmt.Printf("  Memory: %s/%s MB\n", 
                    formatBytes(gpuInfo.VRAMTotalUsedMemory),
                    formatBytes(gpuInfo.VRAMTotalMemory))
            }
        }
    }
}

func formatBytes(bytesStr string) string {
    // Convert bytes to MB for display
    bytes, _ := strconv.ParseInt(bytesStr, 10, 64)
    return fmt.Sprintf("%.0f", float64(bytes)/(1024*1024))
}
```

## Supported Vendors

### NVIDIA
- **Command**: `nvidia-smi`
- **Features**: GPU index, name, memory usage, utilization, temperature, PCI bus ID
- **Requirements**: NVIDIA drivers with nvidia-smi utility

### AMD
- **Command**: `rocm-smi`
- **Features**: Comprehensive GPU information including temperature sensors, power consumption, serial numbers
- **Requirements**: ROCm installation with rocm-smi utility

### Enflame
- **Command**: `efsmi`
- **Features**: GCU temperature, memory usage, utilization metrics
- **Requirements**: Enflame SMI utility installation

### Intel
- **Command**: Intel GPU utilities
- **Features**: Intel GPU monitoring capabilities

### CPU
- **Features**: CPU information and monitoring

## Architecture

The library uses a plugin-based architecture with the following components:

### Core Types
- `GPUInfo`: Struct containing comprehensive GPU information
- `GPUInfoList`: Collection of GPUInfo objects
- `GPUInfoLoader`: Interface for implementing GPU vendor plugins

### Plugin System
- `registry.go`: Central registry for GPU loaders
- Automatic registration via `init()` functions in each vendor package
- Dynamic discovery of available GPU hardware

## Data Structure

```go
type GPUInfo struct {
    Num                         int    `json:"num"`                    // GPU index
    DeviceID                    string `json:"Device ID"`              // Device identifier
    DeviceRev                   string `json:"Device Rev"`             // Device revision
    TemperatureEdge             string `json:"Temperature (Sensor edge) (C)"`
    TemperatureJunction         string `json:"Temperature (Sensor junction) (C)"`
    TemperatureMemory           string `json:"Temperature (Sensor memory) (C)"`
    AverageGraphicsPackagePower string `json:"Average Graphics Package Power (W)"`
    GPUUse                      string `json:"GPU use (%)"`             // GPU utilization
    SerialNumber                string `json:"Serial Number"`          // GPU serial number
    VRAMTotalMemory             string `json:"VRAM Total Memory (B)"`  // Total VRAM in bytes
    VRAMTotalUsedMemory         string `json:"VRAM Total Used Memory (B)"` // Used VRAM in bytes
    CardSeries                  string `json:"Card series"`            // GPU series
    CardModel                   string `json:"Card model"`             // GPU model name
    CardVendor                  string `json:"Card vendor"`             // GPU vendor
    CardSKU                     string `json:"Card SKU"`               // GPU SKU
    PCIBus                      string `json:"PCI Bus"`                // PCI bus identifier
}
```

## Testing

Run the test suite:

```bash
go test ./...
```

The library includes comprehensive tests for each vendor implementation with sample data files in the `testdata/` directories.

## Adding New Vendors

To add support for a new GPU vendor:

1. Create a new package under `pkg/gpu/`
2. Implement the `GPUInfoLoader` interface:
   ```go
   type VendorLoader struct {}
   
   func (v *VendorLoader) Load() (*gpu.GPUInfoList, error)
   func (v *VendorLoader) Available() bool
   func (v *VendorLoader) Vendor() string
   ```
3. Register the loader in `init()`:
   ```go
   func init() {
       gpu.Register(&VendorLoader{})
   }
   ```

## Dependencies

- Go 1.25.0 or higher
- [github.com/hawkli-1994/go-radeontop](https://github.com/hawkli-1994/go-radeontop) - AMD GPU monitoring
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - Testing utilities

## License

This project is open source and available under the [MIT License](LICENSE).

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for:
- New GPU vendor support
- Bug fixes
- Performance improvements
- Documentation updates

## Author

Created by [hawkli-1994](https://github.com/hawkli-1994)

## Support

For issues, questions, or contributions, please open an issue on the [GitHub repository](https://github.com/hawkli-1994/gpu_tools/issues).