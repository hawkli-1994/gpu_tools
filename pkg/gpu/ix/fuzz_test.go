package ix

import (
	"math"
	"testing"
)

func FuzzParseIXSMI(f *testing.F) {
	f.Add(`<ixsmi_log><gpu id="0"/></ixsmi_log>`)
	f.Add(`invalid`)
	f.Fuzz(func(t *testing.T, data string) {
		infos, err := ParseIXSMI(data)
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
