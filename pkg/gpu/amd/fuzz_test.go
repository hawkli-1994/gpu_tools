package amd

import (
	"math"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func checkFuzzInfos(t *testing.T, infos []gpu.GPUInfo) {
	t.Helper()
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
}

func FuzzParsers(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"card-1":{}}`))
	f.Add([]byte(`{"gpu_data":[{"gpu":-1}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		infos, err := (&rocmSMICommand{}).parse(data)
		if err == nil {
			checkFuzzInfos(t, infos)
		}
		infos, err = (&amdSMICommand{}).parse(data, data)
		if err == nil {
			checkFuzzInfos(t, infos)
		}
	})
}
