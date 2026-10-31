package amdriscv

import "testing"

func FuzzSensorParsers(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`invalid`))
	f.Add([]byte(`0000:03:00.0`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseSensors(data)
		_ = sensorName(string(data))
	})
}
