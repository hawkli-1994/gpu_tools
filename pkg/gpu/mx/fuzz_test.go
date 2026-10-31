package mx

import (
	"math"
	"testing"
)

func FuzzParseMX(f *testing.F) {
	f.Add("GPU#0 MXN260 0000:0f:00.0\n")
	f.Add("invalid")
	f.Fuzz(func(t *testing.T, data string) {
		infos, err := parseMxOutput(data)
		if err != nil {
			return
		}
		for _, info := range infos {
			if info.Num < 0 {
				t.Fatalf("negative device number: %d", info.Num)
			}
			for _, value := range []*float64{info.TemperatureEdge, info.TemperatureJunction, info.TemperatureMemory, info.AverageGraphicsPackagePower, info.GPUUse} {
				if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
					t.Fatalf("non-finite metric: %v", *value)
				}
			}
		}
	})
}
