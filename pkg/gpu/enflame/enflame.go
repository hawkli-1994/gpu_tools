package enflame

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
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func New() *enflameSMICommand {
	return &enflameSMICommand{}
}

type enflameSMICommand struct{}

func (e *enflameSMICommand) Load(ctx context.Context) ([]gpu.GPUInfo, error) {
	candidates := []string{"efsmi", "/usr/bin/efsmi"}
	var lastErr error
	for _, path := range candidates {
		args := []string{"-q", "-d", "TEMP,MEMORY,USAGE,PCIE"}
		cmd := exec.CommandContext(ctx, path, args...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			return e.parse(output)
		}
		command := strings.Join(append([]string{path}, args...), " ")
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, &gpu.OpError{Vendor: e.Vendor(), Op: "query", Command: command, Err: ctxErr}
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		lastErr = &gpu.OpError{Vendor: e.Vendor(), Op: "query", Command: command, Err: err}
		if !errors.Is(err, exec.ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
			return nil, lastErr
		}
	}
	return nil, fmt.Errorf("%w: %w", gpu.ErrUnavailable, lastErr)
}

func (e *enflameSMICommand) parse(output []byte) ([]gpu.GPUInfo, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	result := make([]gpu.GPUInfo, 0)

	var current *gpu.GPUInfo
	var currentSection string
	valueRegex := regexp.MustCompile(`:\s+(.+)`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "DEV ID") {
			if current != nil {
				result = append(result, *current)
			}
			current = &gpu.GPUInfo{
				Num:        len(result),
				DeviceID:   strings.TrimSpace(line[len("DEV ID"):]),
				CardVendor: "Enflame",
				CardModel:  "Enflame GCU",
			}
			currentSection = ""
			continue
		}
		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "PCIe Info"):
			currentSection = "pcie"
			continue
		case strings.HasPrefix(line, "Device Mem Info"):
			currentSection = "device_mem"
			continue
		case strings.HasPrefix(line, "BAR1 Mem Info"):
			currentSection = "bar1"
			continue
		case strings.HasPrefix(line, "BAR2 Mem Info"):
			currentSection = "bar2"
			continue
		case strings.HasPrefix(line, "Temperature Info"):
			currentSection = "temperature"
			continue
		case strings.HasPrefix(line, "Device Usage Info"):
			currentSection = "usage"
			continue
		case strings.HasPrefix(line, "Link Info"):
			currentSection = "link"
			continue
		}

		matches := valueRegex.FindStringSubmatch(line)
		if len(matches) <= 1 {
			continue
		}
		value := matches[1]
		var err error
		switch currentSection {
		case "device_mem":
			switch {
			case strings.Contains(line, "Mem Size") || strings.Contains(line, "Total Size"):
				current.VRAMTotalMemory, err = parseMiB(value)
			case strings.Contains(line, "Mem Usage") || strings.Contains(line, "Used Size"):
				current.VRAMTotalUsedMemory, err = parseMiB(value)
			}
		case "temperature":
			if strings.Contains(line, "GCU Temp") {
				current.TemperatureEdge, err = parseEnflameFloat(value)
			}
		case "usage":
			if strings.Contains(line, "GCU Usage") {
				current.GPUUse, err = parseEnflameFloat(value)
			}
		case "pcie":
			switch {
			case strings.Contains(line, "Domain"):
				current.PCIBus = strings.TrimSpace(value)
			case strings.Contains(line, "Bus"):
				current.PCIBus = fmt.Sprintf("%s:%s", current.PCIBus, strings.TrimSpace(value))
			case strings.Contains(line, "Dev  "):
				current.PCIBus = fmt.Sprintf("%s:%s", current.PCIBus, strings.TrimSpace(value))
			case strings.Contains(line, "Func"):
				current.PCIBus = fmt.Sprintf("%s.%s", current.PCIBus, strings.TrimSpace(value))
			}
		}
		if err != nil {
			return nil, fmt.Errorf("parse efsmi gpu %d field %q: %w", current.Num, line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan efsmi output: %w", err)
	}
	if current != nil {
		result = append(result, *current)
	}
	return result, nil
}

func parseEnflameFloat(value string) (*float64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil, nil
	}
	parsed, err := gpu.ParseFloat(fields[0])
	if err != nil {
		return nil, err
	}
	if parsed != nil && (math.IsNaN(*parsed) || math.IsInf(*parsed, 0)) {
		return nil, fmt.Errorf("non-finite value %q", value)
	}
	return parsed, nil
}

func parseMiB(value string) (*uint64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 || strings.EqualFold(strings.TrimSpace(value), "N/A") {
		return nil, nil
	}
	if len(fields) < 2 || fields[1] != "MiB" {
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

func (e *enflameSMICommand) Vendor() string {
	return "Enflame"
}
