package enflame

import (
	"math"
	"testing"
)

func FuzzParseEnflame(f *testing.F) {
	f.Add([]byte("DEV ID 0\n"))
	f.Add([]byte("invalid"))
	f.Fuzz(func(t *testing.T, data []byte) {
		infos, err := New().parse(data)
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
