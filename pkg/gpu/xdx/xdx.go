package xdx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *xdxSMICommand {
	return &xdxSMICommand{}
}

type xdxSMICommand struct{}

func (x *xdxSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	cmd := exec.CommandContext(ctx, "xdxsmi", "out")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: x.Vendor(), Op: "query", Command: cmd.String(), Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		opErr := &gpu.OpError{Vendor: x.Vendor(), Op: "query", Command: cmd.String(), Err: err}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %w", gpu.ErrUnavailable, opErr)
		}
		return nil, opErr
	}
	return x.parse(output)
}

func (x *xdxSMICommand) parse(output []byte) ([]gpu.GPUInfo, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	result := make([]gpu.GPUInfo, 0)
	var current *gpu.GPUInfo
	var currentSection string
	gpuCount := -1
	kvRegex := regexp.MustCompile(`^\s*(.+?)\s*:\s+(.+)$`)
	gpuRegex := regexp.MustCompile(`^\s*GPU\s+(\d+)\s*$`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "======") {
			continue
		}
		if strings.Contains(line, "Attached GPUs") {
			_, value, ok := strings.Cut(line, ":")
			if !ok {
				return nil, fmt.Errorf("parse xdxsmi attached GPU count %q", line)
			}
			count, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || count < 0 {
				return nil, fmt.Errorf("parse xdxsmi attached GPU count %q", value)
			}
			gpuCount = count
			continue
		}
		if matches := gpuRegex.FindStringSubmatch(line); matches != nil {
			if current != nil {
				result = append(result, *current)
			}
			index, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, fmt.Errorf("parse xdxsmi GPU index %q: %w", matches[1], err)
			}
			current = &gpu.GPUInfo{
				Num:        index,
				DeviceID:   matches[1],
				CardVendor: "XDX",
				CardModel:  "XDX GPU",
				CardSeries: "XDX",
			}
			currentSection = ""
			continue
		}
		if current == nil {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			switch trimmed {
			case "PCI":
				currentSection = "pci"
			case "FB Memory Usage":
				currentSection = "fb_memory"
			case "Power":
				currentSection = "power"
			case "Clocks":
				currentSection = "clocks"
			}
			continue
		}

		matches := kvRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		key := strings.TrimSpace(matches[1])
		value := strings.TrimSpace(matches[2])
		var err error
		if indent <= 8 {
			currentSection = ""
			switch key {
			case "Product Name":
				current.CardModel = value
			case "Product Architecture":
				current.CardSeries = value
			case "Temperature":
				current.TemperatureEdge, err = parseXDXFloat(value, "°C", "C")
			}
		} else {
			switch currentSection {
			case "pci":
				if key == "Bus Id" {
					current.PCIBus = value
				}
			case "fb_memory":
				if key == "Total" {
					current.VRAMTotalMemory, err = parseXDXMiB(value)
				}
			case "power":
				if key == "Pwr Total" {
					current.AverageGraphicsPackagePower, err = parseXDXFloat(value, "W")
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("parse xdxsmi gpu %d field %s: %w", current.Num, key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan xdxsmi output: %w", err)
	}
	if current != nil {
		result = append(result, *current)
	}
	if gpuCount >= 0 && len(result) != gpuCount {
		return nil, fmt.Errorf("parse xdxsmi output: attached GPU count %d does not match %d parsed devices", gpuCount, len(result))
	}
	if len(result) == 0 && gpuCount < 0 && strings.TrimSpace(string(output)) != "" {
		return nil, fmt.Errorf("parse xdxsmi output: no GPU devices found")
	}
	return result, nil
}

func parseXDXFloat(value string, expectedUnits ...string) (*float64, error) {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	fields := strings.Fields(value)
	if len(fields) != 2 || !slices.Contains(expectedUnits, fields[1]) {
		return nil, fmt.Errorf("invalid numeric value %q", value)
	}
	return gpu.ParseFloat(fields[0])
}

func parseXDXMiB(value string) (*uint64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	if len(fields) != 2 || fields[1] != "MiB" {
		return nil, fmt.Errorf("invalid MiB value %q", value)
	}
	size, err := gpu.ParseFloat(fields[0])
	if err != nil {
		return nil, err
	}
	if size == nil {
		return nil, nil
	}
	bytes := *size * 1024 * 1024
	if math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes < 0 || bytes >= math.Exp2(64) {
		return nil, fmt.Errorf("invalid memory size %q", value)
	}
	return gpu.Uint64(uint64(bytes + 0.5)), nil
}

func (x *xdxSMICommand) Vendor() string {
	return "XDX"
}
