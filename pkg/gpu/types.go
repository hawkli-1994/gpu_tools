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

type GPUInfoLoader interface {
	Load() (*GPUInfoList, error)
	Available() bool
	Vendor() string
}
