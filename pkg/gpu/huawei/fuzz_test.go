package huawei

import (
	"math"
	"testing"
)

func FuzzHuaweiParsers(f *testing.F) {
	f.Add([]byte("| 0 910B | OK | 50 40 0 / 0 |"))
	f.Add([]byte("Chip ID : 0\nTemperature(C) : 40"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = extractNPUIDs(data)
		memory, devices := extractOverview(data)
		usages := parseChipSections(data)
		if usages == nil {
			usages = make(map[string]map[string]string)
		}
		for npu, chips := range memory {
			for chip, used := range chips {
				if usages[chip] == nil {
					usages[chip] = make(map[string]string)
				}
				usages[chip]["Memory Usage(MB)"] = used
				infos, _, err := buildGPUInfoListWithDeviceIDs(npu, nil, parseChipSections(data), usages, nil, nil, devices[npu], 0)
				if err == nil {
					for _, info := range infos {
						for _, value := range []*float64{info.TemperatureEdge, info.TemperatureJunction, info.TemperatureMemory, info.AverageGraphicsPackagePower, info.GPUUse} {
							if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
								t.Fatalf("non-finite metric: %v", *value)
							}
						}
					}
				}
			}
		}
	})
}
