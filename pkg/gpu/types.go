package gpu

type GPUInfo struct {
	Num                         int    `json:"num"`
	DeviceID                    string `json:"Device ID"`
	DeviceRev                   string `json:"Device Rev"`
	TemperatureEdge             string `json:"Temperature (Sensor edge) (C)"`
	TemperatureJunction         string `json:"Temperature (Sensor junction) (C)"`
	TemperatureMemory           string `json:"Temperature (Sensor memory) (C)"` // 温度
	AverageGraphicsPackagePower string `json:"Average Graphics Package Power (W)"`
	GPUUse                      string `json:"GPU use (%)"` // 利用率
	SerialNumber                string `json:"Serial Number"`
	VRAMTotalMemory             string `json:"VRAM Total Memory (B)"`      // 显存总量
	VRAMTotalUsedMemory         string `json:"VRAM Total Used Memory (B)"` // 显存使用量
	CardSeries                  string `json:"Card series"`
	CardModel                   string `json:"Card model"`
	CardVendor                  string `json:"Card vendor"`
	CardSKU                     string `json:"Card SKU"`
	PCIBus                      string `json:"PCI Bus"`
}

type GPUInfoList struct {
	GPUInfos []GPUInfo
}

// DriverInfo stores information about a GPU driver installed on a Linux system.
type GPUDriverInfo struct {
	Vendor        string `json:"vendor"`         // GPU vendor (e.g., NVIDIA, AMD, Intel)
	Version       string `json:"version"`        // Driver version (e.g., "535.113.01")
	Installed     bool   `json:"installed"`      // Whether the driver is installed
	InstallPath   string `json:"install_path"`   // Path to driver installation (empty if not installed)
	ClientVersion string `json:"client_version"` // Version of the client utility (e.g., nvidia-smi version)
	LibVersion    string `json:"lib_version"`    // Version of the driver library (e.g., CUDA library version)
	DriverDate    string `json:"driver_date"`    // Release date of the driver (e.g., "2025-01-20")
	KernelModule  string `json:"kernel_module"`  // Loaded kernel module (e.g., "nvidia")
	ModuleLoaded  bool   `json:"module_loaded"`  // Whether the kernel module is loaded
	DriverType    string `json:"driver_type"`    // Type of driver (e.g., "proprietary", "open-source")
	// Architecture   string   `json:"architecture"`    // System architecture (e.g., "x86_64")
	// LastError      string   `json:"last_error"`      // Error during data collection, if any
	// CudaVersion    string   `json:"cuda_version"`    // CUDA version supported (e.g., "12.2")
	// OpenGLVersion  string   `json:"opengl_version"`  // OpenGL version supported (e.g., "4.6")
	// Status         string   `json:"status"`          // Driver status (e.g., "active", "inactive")
	// SupportedGPUs  []string `json:"supported_gpus"`  // List of supported GPU models
}

type GPUInfoLoader interface {
	Load() (*GPUInfoList, error)
	Available() bool
	Vendor() string
}
