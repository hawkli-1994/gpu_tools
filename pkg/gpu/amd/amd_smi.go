package amd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

type amdSMICommand struct {
}

func (a *amdSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	const command = "amd-smi"
	if _, err := exec.LookPath(command); err != nil {
		return nil, unavailable("AMD", command, err)
	}
	staticOutput, err := exec.CommandContext(ctx, command, "static", "-a", "-b", "-v", "-B", "--json").Output()
	if err != nil {
		return nil, commandError("AMD", "static", command, ctx, err)
	}
	metricOutput, err := exec.CommandContext(ctx, command, "metric", "-m", "-u", "-p", "-t", "--json").Output()
	if err != nil {
		return nil, commandError("AMD", "metric", command, ctx, err)
	}
	return a.parse(staticOutput, metricOutput)
}

func (a *amdSMICommand) Vendor() string {
	return "AMD"
}

type smiValue struct {
	Value json.RawMessage `json:"value"`
	Unit  string          `json:"unit"`
}

func (v *smiValue) UnmarshalJSON(output []byte) error {
	if !strings.HasPrefix(strings.TrimSpace(string(output)), "{") {
		v.Value = append(v.Value[:0], output...)
		v.Unit = ""
		return nil
	}
	type plain smiValue
	var decoded plain
	if err := json.Unmarshal(output, &decoded); err != nil {
		return err
	}
	*v = smiValue(decoded)
	return nil
}

type smiStaticGPU struct {
	GPU  int `json:"gpu"`
	ASIC struct {
		MarketName string `json:"market_name"`
		VendorName string `json:"vendor_name"`
		DeviceID   string `json:"device_id"`
		RevID      string `json:"rev_id"`
		Serial     string `json:"asic_serial"`
	} `json:"asic"`
	Bus struct {
		BDF string `json:"bdf"`
	} `json:"bus"`
	Board struct {
		ModelNumber  string `json:"model_number"`
		Manufacturer string `json:"manufacturer_name"`
	} `json:"board"`
	VRAM struct {
		Size smiValue `json:"size"`
	} `json:"vram"`
}

type smiStaticData struct {
	GPUData []smiStaticGPU `json:"gpu_data"`
}

func (d *smiStaticData) UnmarshalJSON(output []byte) error {
	if strings.HasPrefix(strings.TrimSpace(string(output)), "[") {
		return json.Unmarshal(output, &d.GPUData)
	}
	type plain smiStaticData
	var decoded plain
	if err := json.Unmarshal(output, &decoded); err != nil {
		return err
	}
	*d = smiStaticData(decoded)
	return nil
}

type smiGroup[T any] struct {
	Value T
}

func (g *smiGroup[T]) UnmarshalJSON(output []byte) error {
	var text string
	if err := json.Unmarshal(output, &text); err == nil && strings.EqualFold(text, "N/A") {
		var zero T
		g.Value = zero
		return nil
	}
	var value T
	if err := json.Unmarshal(output, &value); err != nil {
		return err
	}
	g.Value = value
	return nil
}

type smiUsage struct {
	GFXActivity smiValue `json:"gfx_activity"`
}

type smiPower struct {
	SocketPower  smiValue `json:"socket_power"`
	CurrentPower smiValue `json:"current_power"`
}

type smiTemperature struct {
	Edge    smiValue `json:"edge"`
	Hotspot smiValue `json:"hotspot"`
	Memory  smiValue `json:"mem"`
}

type smiMemoryUsage struct {
	TotalVRAM smiValue `json:"total_vram"`
	UsedVRAM  smiValue `json:"used_vram"`
}

type smiMetricGPU struct {
	GPU         int                      `json:"gpu"`
	Usage       smiGroup[smiUsage]       `json:"usage"`
	Power       smiGroup[smiPower]       `json:"power"`
	Temperature smiGroup[smiTemperature] `json:"temperature"`
	MemoryUsage smiGroup[smiMemoryUsage] `json:"mem_usage"`
}

type smiMetricData struct {
	GPUData []smiMetricGPU `json:"gpu_data"`
}

func (d *smiMetricData) UnmarshalJSON(output []byte) error {
	if strings.HasPrefix(strings.TrimSpace(string(output)), "[") {
		return json.Unmarshal(output, &d.GPUData)
	}
	type plain smiMetricData
	var decoded plain
	if err := json.Unmarshal(output, &decoded); err != nil {
		return err
	}
	*d = smiMetricData(decoded)
	return nil
}

func (a *amdSMICommand) parse(staticOutput, metricOutput []byte) ([]gpu.GPUInfo, error) {
	var staticData smiStaticData
	if err := json.Unmarshal(staticOutput, &staticData); err != nil {
		return nil, fmt.Errorf("parse amd-smi static output: %w", err)
	}
	var metricData smiMetricData
	if err := json.Unmarshal(metricOutput, &metricData); err != nil {
		return nil, fmt.Errorf("parse amd-smi metric output: %w", err)
	}

	infos := make(map[int]gpu.GPUInfo, len(staticData.GPUData))
	for _, data := range staticData.GPUData {
		if data.GPU < 0 {
			return nil, fmt.Errorf("parse amd-smi GPU index %d: invalid index", data.GPU)
		}
		vendor := data.Board.Manufacturer
		if vendor == "" || vendor == "N/A" {
			vendor = data.ASIC.VendorName
		}
		totalMemory, err := data.VRAM.Size.bytes()
		if err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d total VRAM: %w", data.GPU, err)
		}
		infos[data.GPU] = gpu.GPUInfo{
			Num:             data.GPU,
			DeviceID:        data.ASIC.DeviceID,
			DeviceRev:       data.ASIC.RevID,
			SerialNumber:    data.ASIC.Serial,
			VRAMTotalMemory: totalMemory,
			CardSeries:      strings.TrimSpace(data.ASIC.MarketName),
			CardModel:       data.Board.ModelNumber,
			CardVendor:      vendor,
			CardSKU:         data.Board.ModelNumber,
			PCIBus:          data.Bus.BDF,
		}
	}
	for _, data := range metricData.GPUData {
		if data.GPU < 0 {
			return nil, fmt.Errorf("parse amd-smi GPU index %d: invalid index", data.GPU)
		}
		info := infos[data.GPU]
		info.Num = data.GPU
		var err error
		if info.TemperatureEdge, err = data.Temperature.Value.Edge.float(); err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d edge temperature: %w", data.GPU, err)
		}
		if info.TemperatureJunction, err = data.Temperature.Value.Hotspot.float(); err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d hotspot temperature: %w", data.GPU, err)
		}
		if info.TemperatureMemory, err = data.Temperature.Value.Memory.float(); err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d memory temperature: %w", data.GPU, err)
		}
		if info.AverageGraphicsPackagePower, err = data.Power.Value.SocketPower.float(); err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d power: %w", data.GPU, err)
		}
		if info.AverageGraphicsPackagePower == nil {
			if info.AverageGraphicsPackagePower, err = data.Power.Value.CurrentPower.float(); err != nil {
				return nil, fmt.Errorf("parse amd-smi GPU %d current power: %w", data.GPU, err)
			}
		}
		if info.GPUUse, err = data.Usage.Value.GFXActivity.float(); err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d utilization: %w", data.GPU, err)
		}
		totalMemory, err := data.MemoryUsage.Value.TotalVRAM.bytes()
		if err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d total VRAM: %w", data.GPU, err)
		}
		usedMemory, err := data.MemoryUsage.Value.UsedVRAM.bytes()
		if err != nil {
			return nil, fmt.Errorf("parse amd-smi GPU %d used VRAM: %w", data.GPU, err)
		}
		if totalMemory != nil {
			info.VRAMTotalMemory = totalMemory
		}
		info.VRAMTotalUsedMemory = usedMemory
		infos[data.GPU] = info
	}

	nums := make([]int, 0, len(infos))
	for num := range infos {
		nums = append(nums, num)
	}
	sort.Ints(nums)
	result := make([]gpu.GPUInfo, 0, len(nums))
	for _, num := range nums {
		result = append(result, infos[num])
	}
	return result, nil
}

func (v smiValue) String() string {
	if len(v.Value) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(v.Value, &text); err == nil {
		return text
	}
	return string(v.Value)
}

func (v smiValue) float() (*float64, error) {
	return gpu.ParseFloat(v.String())
}

func (v smiValue) bytes() (*uint64, error) {
	text := v.String()
	if text == "" || text == "N/A" {
		return nil, nil
	}
	multiplier := float64(1)
	switch strings.ToUpper(v.Unit) {
	case "B":
	case "KB", "KIB":
		multiplier = 1024
	case "", "MB", "MIB":
		multiplier = 1024 * 1024
	case "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	default:
		return nil, fmt.Errorf("unsupported memory unit %q", v.Unit)
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, err
	}
	bytes := value * multiplier
	if math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes < 0 || bytes >= math.Exp2(64) || math.Trunc(bytes) != bytes {
		return nil, fmt.Errorf("invalid byte quantity %q %s", text, v.Unit)
	}
	return gpu.Uint64(uint64(bytes)), nil
}
