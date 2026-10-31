# GPU Tools

GPU Tools is a Go library for collecting GPU and accelerator information through a single, context-aware API. It supports device identity, temperature, utilization, power, memory, and PCI bus information across multiple vendors.

## Features

- One `Loader` interface for all supported vendors.
- Context cancellation and deadlines for every external command.
- Typed metrics with explicit units and unknown-value handling.
- A concurrency-safe, explicitly constructed registry with no global state.
- Structured errors that preserve command, vendor, and operation context.
- Offline fixture-based parser tests.

## Requirements

- Go 1.25 or newer.
- The vendor SMI tool and driver required by each loader. Parser tests do not require GPU hardware.

## Installation

```bash
go get github.com/hawkli-1994/gpu_tools/pkg/gpu/all
```

Import `pkg/gpu/all` when you want every loader, or import an individual vendor package to keep dependencies and initialization explicit.

## Quick start

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/hawkli-1994/gpu_tools/pkg/gpu"
    "github.com/hawkli-1994/gpu_tools/pkg/gpu/all"
)

func main() {
    registry, err := all.NewRegistry()
    if err != nil {
        log.Fatal(err)
    }

    for _, loader := range registry.Loaders() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        devices, err := loader.Load(ctx)
        cancel()

        if errors.Is(err, gpu.ErrUnavailable) {
            continue
        }
        if err != nil {
            log.Printf("load %s GPUs: %v", loader.Vendor(), err)
            continue
        }

        for _, device := range devices {
            fmt.Printf("%s GPU %d: %s\n", loader.Vendor(), device.Num, device.CardModel)
            if device.GPUUse != nil {
                fmt.Printf("  utilization: %.1f%%\n", *device.GPUUse)
            }
            if device.VRAMTotalUsedMemory != nil && device.VRAMTotalMemory != nil {
                fmt.Printf(
                    "  memory: %d/%d bytes\n",
                    *device.VRAMTotalUsedMemory,
                    *device.VRAMTotalMemory,
                )
            }
        }
    }
}
```

`Loaders` returns loaders sorted by vendor. A separate timeout per loader prevents one slow vendor from consuming the timeout budget of all subsequent loaders.

## Use one vendor

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/hawkli-1994/gpu_tools/pkg/gpu/nvidia"
)

func main() {
    loader := nvidia.New()
    devices, err := loader.Load(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    for _, device := range devices {
        fmt.Printf("GPU %d: %s\n", device.Num, device.CardModel)
    }
}
```

AMD prefers `amd-smi` and falls back to `rocm-smi` only when `amd-smi` is unavailable. Execution and parsing failures are returned rather than hidden by the fallback.

## Registry

Create a registry containing only the loaders your application needs:

```go
registry, err := gpu.NewRegistry(amd.New(), nvidia.New())
if err != nil {
    // Handle duplicate or invalid loaders.
}

loader, ok := registry.Lookup("NVIDIA")
loaders := registry.Loaders()
```

The registry API is:

```go
func NewRegistry(loaders ...Loader) (*Registry, error)
func (r *Registry) Register(loader Loader) error
func (r *Registry) Lookup(vendor string) (Loader, bool)
func (r *Registry) Loaders() []Loader
```

Registration rejects nil loaders, empty vendor names, and duplicate vendors. `Loaders` returns a sorted snapshot, so callers cannot mutate the registry's internal map. The zero value of `Registry` is ready for use.

There is no package-global registry and no `init()` self-registration. `all.NewRegistry()` is simply an explicit constructor for the complete set.

## Core interface and data model

```go
type Loader interface {
    Load(context.Context) ([]GPUInfo, error)
    Vendor() string
}
```

`GPUInfo` contains the following fields:

| Field | Type | Meaning |
| --- | --- | --- |
| `Num` | `int` | GPU index reported by the vendor |
| `DeviceID` | `string` | Device identifier |
| `DeviceRev` | `string` | Device revision |
| `SerialNumber` | `string` | Device serial number |
| `CardSeries` | `string` | Product series |
| `CardModel` | `string` | Product model |
| `CardVendor` | `string` | Device vendor |
| `CardSKU` | `string` | Product SKU |
| `PCIBus` | `string` | PCI bus/device/function address |
| `TemperatureEdge` | `*float64` | Edge temperature in Celsius |
| `TemperatureJunction` | `*float64` | Junction or hotspot temperature in Celsius |
| `TemperatureMemory` | `*float64` | Memory temperature in Celsius |
| `AverageGraphicsPackagePower` | `*float64` | Graphics package power in watts |
| `GPUUse` | `*float64` | GPU utilization in percent |
| `VRAMTotalMemory` | `*uint64` | Total VRAM in bytes |
| `VRAMTotalUsedMemory` | `*uint64` | Used VRAM in bytes |

For every pointer metric:

- `nil` means unknown, unsupported, or not supplied by the vendor tool.
- A non-nil pointer to zero means the measured value is actually zero.
- A generic temperature is reported only in the most appropriate field; it is not copied into unrelated sensor fields.
- Non-finite floating-point values and malformed numeric input are rejected.

JSON uses stable `snake_case` names such as `temperature_edge_c`, `gpu_use_percent`, and `vram_total_bytes`. Unknown metrics are omitted with `omitempty`.

## Errors and cancellation

Use `errors.Is` to distinguish an unavailable loader from a real collection failure:

```go
devices, err := loader.Load(ctx)
switch {
case errors.Is(err, gpu.ErrUnavailable):
    // The tool is missing or the platform is unsupported.
case errors.Is(err, context.DeadlineExceeded):
    // The vendor command exceeded the caller's deadline.
case err != nil:
    // Command execution or output parsing failed.
}
```

For command and operation details, inspect `gpu.OpError`:

```go
var opErr *gpu.OpError
if errors.As(err, &opErr) {
    fmt.Printf("vendor=%s op=%s command=%s\n", opErr.Vendor, opErr.Op, opErr.Command)
}
```

`OpError.Unwrap` preserves the underlying error. Loaders do not turn arbitrary driver, permission, or parsing failures into `ErrUnavailable`.

## NVIDIA driver information

NVIDIA also implements the optional `DriverLoader` interface:

```go
loader := nvidia.New()
info, err := loader.DriverInfo(ctx)
```

The returned `GPUDriverInfo` includes the NVIDIA SMI path and the client, driver, and CUDA versions reported by `nvidia-smi --version`.

## Supported vendors

| Vendor | Loader package | Backend |
| --- | --- | --- |
| NVIDIA | `pkg/gpu/nvidia` | `nvidia-smi` |
| AMD | `pkg/gpu/amd` | `amd-smi`, then `rocm-smi` if unavailable |
| AMD RISC-V | `pkg/gpu/amd_riscv` | `sensors -j` and sysfs |
| CPU | `pkg/gpu/cpu` | Placeholder loader returning no GPU devices |
| Denglin | `pkg/gpu/dl` | `dlsmi` |
| Enflame | `pkg/gpu/enflame` | `efsmi` |
| HoumoAI | `pkg/gpu/houmoai` | `hm_smi` |
| Huawei | `pkg/gpu/huawei` | `npu-smi` |
| Iluvatar | `pkg/gpu/ix` | CoreX `ixsmi` |
| MetaX | `pkg/gpu/mx` | `mx-smi` |
| XDX | `pkg/gpu/xdx` | `xdxsmi` |

Not every backend exposes every metric. Missing capabilities are represented by `nil`, not fabricated zero values.

## Development

Run the complete validation suite before submitting changes:

```bash
gofmt -w pkg
go test ./...
go test -race ./...
go build ./...
go vet ./...
# Run each vendor's fuzz target for a bounded smoke test, for example:
go test -run='^$' -fuzz=FuzzParseNVIDIA -fuzztime=10s ./pkg/gpu/nvidia
```

Vendor parser tests use offline fixtures in co-located `testdata` directories, so the standard test suite does not require GPU hardware. Fuzz targets additionally check that arbitrary input cannot panic a parser or produce negative device numbers and non-finite metrics.

### Add a vendor

1. Create `pkg/gpu/<vendor>` and provide `New()` returning a loader.
2. Implement `Load(context.Context) ([]gpu.GPUInfo, error)` and `Vendor() string`.
3. Use `exec.CommandContext` for external commands and preserve errors with `%w` or `gpu.OpError`.
4. Convert values to the units in `GPUInfo`; use `nil` for unknown metrics.
5. Add offline fixtures and tests for parsing, invalid input, and missing values.
6. Add the constructor to `pkg/gpu/all/all.go` and update `pkg/gpu/all/all_test.go`.

Do not add package-level registration or mutable global loader state.

## License

GPU Tools is available under the [MIT License](LICENSE).
