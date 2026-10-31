package houmoai

import (
	"math"
	"testing"
)

func TestAverageDDRTempRejectsOverflow(t *testing.T) {
	_, err := averageDDRTemp(map[string]string{"DDR0": "1e308 C", "DDR2": "1e308 C"})
	if err == nil {
		t.Fatal("expected overflow error")
	}
}

func FuzzParseHoumo(f *testing.F) {
	f.Add([]byte("device0 detail infos\n"))
	f.Add([]byte("device0 detail infos\nDDR0 : 1e308 C\nDDR2 : 1e308 C\n"))
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
