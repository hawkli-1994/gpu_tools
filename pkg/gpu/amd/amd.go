package amd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *amdCommand {
	return &amdCommand{
		amdSMI:  &amdSMICommand{},
		rocmSMI: &rocmSMICommand{},
	}
}

type amdCommand struct {
	amdSMI  gpu.Loader
	rocmSMI gpu.Loader
}

func (a *amdCommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	infos, err := a.amdSMI.Load(ctx)
	if errors.Is(err, gpu.ErrUnavailable) {
		return a.rocmSMI.Load(ctx)
	}
	return infos, err
}

func (a *amdCommand) Vendor() string {
	return "AMD"
}

type rocmSMICommand struct {
}

func (r *rocmSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	const command = "rocm-smi"
	commandPath, err := exec.LookPath(command)
	if err != nil {
		commandPath, err = exec.LookPath("/opt/rocm/bin/rocm-smi")
		if err != nil {
			return nil, unavailable("ROCm", command, err)
		}
	}
	cmd := exec.CommandContext(ctx, commandPath, "--showid", "--showmeminfo", "vram", "--showpower", "--showserial", "--showuse", "--showtemp", "--showproductname", "--showbus", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError("ROCm", "load", command, ctx, err)
	}
	return r.parse(output)
}

type rocmGPUInfo struct {
	DeviceID                          string `json:"Device ID"`
	DeviceRev                         string `json:"Device Rev"`
	TemperatureEdge                   string `json:"Temperature (Sensor edge) (C)"`
	TemperatureJunction               string `json:"Temperature (Sensor junction) (C)"`
	TemperatureMemory                 string `json:"Temperature (Sensor memory) (C)"`
	AverageGraphicsPackagePower       string `json:"Average Graphics Package Power (W)"`
	CurrentSocketGraphicsPackagePower string `json:"Current Socket Graphics Package Power (W)"`
	GPUUse                            string `json:"GPU use (%)"`
	SerialNumber                      string `json:"Serial Number"`
	VRAMTotalMemory                   string `json:"VRAM Total Memory (B)"`
	VRAMTotalUsedMemory               string `json:"VRAM Total Used Memory (B)"`
	CardSeries                        string `json:"Card series"`
	CardModel                         string `json:"Card model"`
	CardVendor                        string `json:"Card vendor"`
	CardSKU                           string `json:"Card SKU"`
	PCIBus                            string `json:"PCI Bus"`
}

func (r *rocmSMICommand) parse(output []byte) ([]gpu.GPUInfo, error) {
	var jsonData map[string]rocmGPUInfo
	if err := json.Unmarshal(output, &jsonData); err != nil {
		return nil, fmt.Errorf("parse rocm-smi output: %w", err)
	}
	infos := make([]gpu.GPUInfo, 0, len(jsonData))
	for cardNum, data := range jsonData {
		num, err := strconv.Atoi(strings.TrimPrefix(cardNum, "card"))
		if err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU index %q: %w", cardNum, err)
		}
		info := gpu.GPUInfo{
			Num:          num,
			DeviceID:     data.DeviceID,
			DeviceRev:    data.DeviceRev,
			SerialNumber: data.SerialNumber,
			CardSeries:   data.CardSeries,
			CardModel:    data.CardModel,
			CardVendor:   data.CardVendor,
			CardSKU:      data.CardSKU,
			PCIBus:       data.PCIBus,
		}
		if info.TemperatureEdge, err = gpu.ParseFloat(data.TemperatureEdge); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d edge temperature: %w", num, err)
		}
		if info.TemperatureJunction, err = gpu.ParseFloat(data.TemperatureJunction); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d junction temperature: %w", num, err)
		}
		if info.TemperatureMemory, err = gpu.ParseFloat(data.TemperatureMemory); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d memory temperature: %w", num, err)
		}
		if info.AverageGraphicsPackagePower, err = gpu.ParseFloat(data.AverageGraphicsPackagePower); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d power: %w", num, err)
		}
		if info.AverageGraphicsPackagePower == nil {
			if info.AverageGraphicsPackagePower, err = gpu.ParseFloat(data.CurrentSocketGraphicsPackagePower); err != nil {
				return nil, fmt.Errorf("parse rocm-smi GPU %d current power: %w", num, err)
			}
		}
		if info.GPUUse, err = gpu.ParseFloat(data.GPUUse); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d utilization: %w", num, err)
		}
		if info.VRAMTotalMemory, err = gpu.ParseUint(data.VRAMTotalMemory); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d total VRAM: %w", num, err)
		}
		if info.VRAMTotalUsedMemory, err = gpu.ParseUint(data.VRAMTotalUsedMemory); err != nil {
			return nil, fmt.Errorf("parse rocm-smi GPU %d used VRAM: %w", num, err)
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Num < infos[j].Num })
	return infos, nil
}

func (r *rocmSMICommand) Vendor() string {
	return "AMD"
}

func unavailable(vendor, command string, err error) error {
	return &gpu.OpError{
		Vendor:  vendor,
		Op:      "load",
		Command: command,
		Err:     fmt.Errorf("%w: %w", gpu.ErrUnavailable, err),
	}
}

func commandError(vendor, op, command string, ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return unavailable(vendor, command, err)
	}
	return &gpu.OpError{Vendor: vendor, Op: op, Command: command, Err: err}
}
