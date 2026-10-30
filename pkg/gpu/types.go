package gpu

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type GPUInfo struct {
	Num                         int      `json:"num"`
	DeviceID                    string   `json:"device_id"`
	DeviceRev                   string   `json:"device_rev"`
	TemperatureEdge             *float64 `json:"temperature_edge_c,omitempty"`
	TemperatureJunction         *float64 `json:"temperature_junction_c,omitempty"`
	TemperatureMemory           *float64 `json:"temperature_memory_c,omitempty"`
	AverageGraphicsPackagePower *float64 `json:"average_graphics_package_power_w,omitempty"`
	GPUUse                      *float64 `json:"gpu_use_percent,omitempty"`
	SerialNumber                string   `json:"serial_number"`
	VRAMTotalMemory             *uint64  `json:"vram_total_bytes,omitempty"`
	VRAMTotalUsedMemory         *uint64  `json:"vram_used_bytes,omitempty"`
	CardSeries                  string   `json:"card_series"`
	CardModel                   string   `json:"card_model"`
	CardVendor                  string   `json:"card_vendor"`
	CardSKU                     string   `json:"card_sku"`
	PCIBus                      string   `json:"pci_bus"`
}

type GPUDriverInfo struct {
	Vendor        string `json:"vendor"`
	Version       string `json:"version"`
	Installed     bool   `json:"installed"`
	InstallPath   string `json:"install_path"`
	ClientVersion string `json:"client_version"`
	LibVersion    string `json:"lib_version"`
	DriverDate    string `json:"driver_date"`
	KernelModule  string `json:"kernel_module"`
	ModuleLoaded  bool   `json:"module_loaded"`
	DriverType    string `json:"driver_type"`
}

type Loader interface {
	Load(context.Context) ([]GPUInfo, error)
	Vendor() string
}

type DriverLoader interface {
	DriverInfo(context.Context) (GPUDriverInfo, error)
}

func Float64(value float64) *float64 {
	return &value
}

func Uint64(value uint64) *uint64 {
	return &value
}

func ParseFloat(value string) (*float64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return nil, fmt.Errorf("non-finite value %q", value)
	}
	return &parsed, nil
}

func ParseUint(value string) (*uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
