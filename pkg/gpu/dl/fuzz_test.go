package dl

import (
	"math"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func FuzzParseDLSMI(f *testing.F) {
	f.Add([]byte(`<dlsmi_log><gpu id="0"/></dlsmi_log>`))
	f.Add([]byte(`invalid`))
	f.Fuzz(func(t *testing.T, data []byte) {
		infos, err := parseDLSMIOutput(data)
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

var _ gpu.Loader = New()
