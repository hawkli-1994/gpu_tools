package nvidia

import (
	"math"
	"testing"
)

func FuzzParseNVIDIA(f *testing.F) {
	f.Add([]byte("0, NVIDIA, 1024 MiB, 0 MiB, 0 %, 40, 0000:01:00.0"))
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
