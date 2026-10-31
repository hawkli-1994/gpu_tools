package dl

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *dlsmiCommand {
	return &dlsmiCommand{}
}

type dlsmiCommand struct{}

func (d *dlsmiCommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	output, err := d.query(ctx)
	if err != nil {
		return nil, err
	}
	return parseDLSMIOutput(output)
}

func (d *dlsmiCommand) query(ctx context.Context) ([]byte, error) {
	candidates := [][]string{
		{"dlsmi", "query", "--xml-format"},
		{"/usr/bin/dlsmi", "query", "--xml-format"},
		{"/usr/local/bin/dlsmi", "query", "--xml-format"},
	}

	var lastErr error
	for _, args := range candidates {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			return output, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: d.Vendor(), Op: "query", Command: strings.Join(args, " "), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		lastErr = &gpu.OpError{Vendor: d.Vendor(), Op: "query", Command: strings.Join(args, " "), Err: err}
		if !errors.Is(err, exec.ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
			return nil, lastErr
		}
	}

	if lastErr == nil {
		lastErr = errors.New("dlsmi command not found")
	}
	return nil, fmt.Errorf("%w: %w", gpu.ErrUnavailable, lastErr)
}

func (d *dlsmiCommand) Vendor() string {
	return "Denglin"
}

type dlsmiLog struct {
	XMLName      xml.Name   `xml:"dlsmi_log"`
	AttachedGPUs *int       `xml:"attached_gpus"`
	GPUs         []dlsmiGPU `xml:"gpu"`
}

type dlsmiGPU struct {
	ID                  string             `xml:"id,attr"`
	MinorNumber         string             `xml:"minor_number"`
	ProductName         string             `xml:"product_name"`
	ProductBrand        string             `xml:"product_brand"`
	ProductArchitecture string             `xml:"product_architecture"`
	SerialNumber        string             `xml:"serial_number"`
	FirmwareVersion     string             `xml:"fw_version"`
	BoardPartNumber     string             `xml:"board_part_number"`
	PCI                 dlsmiPCISection    `xml:"pci"`
	MemoryUsage         dlsmiMemorySection `xml:"memory_usage"`
	Utilization         dlsmiUtilization   `xml:"utilization"`
	Temperature         dlsmiTemperature   `xml:"temperature"`
	PowerReadings       dlsmiPowerReadings `xml:"power_readings"`
}

type dlsmiPCISection struct {
	Domain      string `xml:"domain"`
	Bus         string `xml:"bus"`
	Device      string `xml:"device"`
	BusID       string `xml:"bus_id"`
	DeviceID    string `xml:"device_id"`
	SubSystemID string `xml:"sub_system_id"`
}

type dlsmiMemorySection struct {
	Total string `xml:"total"`
	Used  string `xml:"used"`
	Free  string `xml:"free"`
}

type dlsmiUtilization struct {
	GPU string `xml:"gpu"`
}

type dlsmiTemperature struct {
	GPUCurrent    string `xml:"gpu_current_temp"`
	GPUSlowdown   string `xml:"gpu_slowdown_temp"`
	GPUShutdown   string `xml:"gpu_shutdown_temp"`
	GPUMaxOper    string `xml:"gpu_max_operating_temp"`
	MemoryCurrent string `xml:"memory_current_temp"`
	MemoryMax     string `xml:"memory_max_operating_temp"`
}

type dlsmiPowerReadings struct {
	PowerDraw string `xml:"power_draw"`
}

func parseDLSMIOutput(output []byte) ([]gpu.GPUInfo, error) {
	var log dlsmiLog
	if err := xml.Unmarshal(output, &log); err != nil {
		return nil, fmt.Errorf("parse dlsmi output: %w", err)
	}
	if log.AttachedGPUs != nil && (*log.AttachedGPUs < 0 || *log.AttachedGPUs != len(log.GPUs)) {
		return nil, fmt.Errorf("parse dlsmi output: attached GPU count %d does not match %d parsed devices", *log.AttachedGPUs, len(log.GPUs))
	}

	infos := make([]gpu.GPUInfo, 0, len(log.GPUs))
	for idx, node := range log.GPUs {
		num := idx
		if minor := strings.TrimSpace(node.MinorNumber); minor != "" && !strings.EqualFold(minor, "N/A") {
			parsed, err := strconv.Atoi(minor)
			if err != nil || parsed < 0 {
				return nil, fmt.Errorf("parse dlsmi gpu %d minor number %q", idx, node.MinorNumber)
			}
			num = parsed
		}
		info := gpu.GPUInfo{
			Num:          num,
			DeviceID:     strings.TrimSpace(node.PCI.DeviceID),
			DeviceRev:    strings.TrimSpace(node.FirmwareVersion),
			SerialNumber: strings.TrimSpace(node.SerialNumber),
			CardSeries:   strings.TrimSpace(node.ProductArchitecture),
			CardModel:    strings.TrimSpace(node.ProductName),
			CardVendor:   resolveVendor(node.ProductBrand),
			CardSKU:      strings.TrimSpace(node.BoardPartNumber),
			PCIBus:       resolveBusID(node),
		}
		if info.DeviceID == "" {
			info.DeviceID = strings.TrimSpace(node.ID)
		}

		var err error
		if info.VRAMTotalMemory, err = convertSizeToBytes(node.MemoryUsage.Total); err != nil {
			return nil, fmt.Errorf("parse dlsmi gpu %d total memory: %w", num, err)
		}
		if info.VRAMTotalUsedMemory, err = convertSizeToBytes(node.MemoryUsage.Used); err != nil {
			return nil, fmt.Errorf("parse dlsmi gpu %d used memory: %w", num, err)
		}
		if info.GPUUse, err = parseNumericField(node.Utilization.GPU, "%"); err != nil {
			return nil, fmt.Errorf("parse dlsmi gpu %d utilization: %w", num, err)
		}
		if info.TemperatureEdge, err = parseNumericField(node.Temperature.GPUCurrent, "C"); err != nil {
			return nil, fmt.Errorf("parse dlsmi gpu %d current temperature: %w", num, err)
		}
		if info.TemperatureMemory, err = parseNumericField(node.Temperature.MemoryCurrent, "C"); err != nil {
			return nil, fmt.Errorf("parse dlsmi gpu %d memory temperature: %w", num, err)
		}
		if info.AverageGraphicsPackagePower, err = parseNumericField(node.PowerReadings.PowerDraw, "W"); err != nil {
			return nil, fmt.Errorf("parse dlsmi gpu %d power draw: %w", num, err)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func resolveVendor(productBrand string) string {
	if trimmed := strings.TrimSpace(productBrand); trimmed != "" && !strings.EqualFold(trimmed, "N/A") {
		return trimmed
	}
	return "Denglin"
}

func resolveBusID(g dlsmiGPU) string {
	if bid := strings.TrimSpace(g.PCI.BusID); bid != "" {
		return bid
	}
	return strings.TrimSpace(g.ID)
}

func parseNumericField(value, expectedUnit string) (*float64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	fields := strings.Fields(value)
	if len(fields) != 2 || fields[1] != expectedUnit {
		return nil, fmt.Errorf("parse %q: expected unit %s", value, expectedUnit)
	}
	parsed, err := gpu.ParseFloat(fields[0])
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", value, err)
	}
	return parsed, nil
}

func convertSizeToBytes(value string) (*uint64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	if len(fields) > 2 {
		return nil, fmt.Errorf("parse %q: unexpected trailing data", value)
	}
	num, err := strconv.ParseFloat(strings.TrimPrefix(fields[0], "+"), 64)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", value, err)
	}
	if math.IsNaN(num) || math.IsInf(num, 0) || num < 0 {
		return nil, fmt.Errorf("parse %q: invalid size", value)
	}

	unit := "B"
	if len(fields) > 1 {
		unit = strings.ToUpper(strings.TrimSuffix(fields[1], "s"))
	}
	multiplier := float64(1)
	switch unit {
	case "B":
	case "KB", "KIB":
		multiplier = 1024
	case "MB", "MIB":
		multiplier = 1024 * 1024
	case "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	case "TB", "TIB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return nil, fmt.Errorf("parse %q: unsupported size unit %q", value, fields[1])
	}

	bytes := math.Round(num * multiplier)
	if bytes < 0 || bytes >= math.Exp2(64) {
		return nil, fmt.Errorf("parse %q: size out of range", value)
	}
	return gpu.Uint64(uint64(bytes)), nil
}
