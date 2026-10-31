package enflame

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/efs.txt
var efs []byte

//go:embed testdata/efs-15.txt
var efs15 []byte

//go:embed testdata/efs17.txt
var efs17 []byte

func enflameExpected(num int, deviceID string, temp float64, total, used uint64, usage float64, pciBus string) gpu.GPUInfo {
	return gpu.GPUInfo{
		Num: num, DeviceID: deviceID, CardVendor: "Enflame", CardModel: "Enflame GCU",
		TemperatureEdge: gpu.Float64(temp),
		VRAMTotalMemory: gpu.Uint64(total), VRAMTotalUsedMemory: gpu.Uint64(used), GPUUse: gpu.Float64(usage), PCIBus: pciBus,
	}
}

func TestEnflameParse(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []gpu.GPUInfo
	}{
		{
			name: "efs_old_format", input: efs,
			expected: []gpu.GPUInfo{
				enflameExpected(0, "0", 39, 42976*1024*1024, 1129*1024*1024, 0, "0000:0c:00.0"),
				enflameExpected(1, "1", 39, 42976*1024*1024, 1129*1024*1024, 0, "0000:0f:00.0"),
			},
		},
		{
			name: "efs15_total_size_no_bar", input: efs15,
			expected: []gpu.GPUInfo{
				enflameExpected(0, "0", 58, 42976*1024*1024, 24028*1024*1024, 0, "0000:0c:00.0"),
				enflameExpected(1, "1", 59, 42976*1024*1024, 24028*1024*1024, 0, "0000:0f:00.0"),
			},
		},
		{
			name: "efs17_total_size_with_bar", input: efs17,
			expected: []gpu.GPUInfo{
				enflameExpected(0, "0", 76, 42976*1024*1024, 8684*1024*1024, 0, "0000:0c:00.0"),
				{Num: 1, DeviceID: "1", CardVendor: "Enflame", CardModel: "Enflame GCU", PCIBus: "0000:0f:00.0"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infos, err := New().parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, infos)
		})
	}
}

func TestEnflameParseInvalidNumericValue(t *testing.T) {
	_, err := New().parse([]byte("DEV ID 0\nTemperature Info\nGCU Temp : invalid C\n"))
	assert.Error(t, err)
}

func TestEnflameParseDeviceNumberAndUnits(t *testing.T) {
	infos, err := New().parse([]byte("DEV ID 4\nTemperature Info\nGCU Temp : 39 ℃\nDevice Usage Info\nGCU Usage : 12.5 %\n"))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 4, infos[0].Num)
	assert.Equal(t, gpu.Float64(39), infos[0].TemperatureEdge)
	assert.Equal(t, gpu.Float64(12.5), infos[0].GPUUse)

	for _, input := range []string{
		"DEV ID 0\nTemperature Info\nGCU Temp : 39 %\n",
		"DEV ID 0\nDevice Usage Info\nGCU Usage : 10 C\n",
		"DEV ID 0\nDevice Mem Info\nMem Size : 1 GiB trailing\n",
	} {
		_, err := New().parse([]byte(input))
		assert.Error(t, err, input)
	}
}

func TestLoadDoesNotRetryCommandExitFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "efsmi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho 'efsmi exited' >&2\nexit 24\n"), 0o755))
	t.Setenv("PATH", dir)

	_, err := New().Load(context.Background())
	require.Error(t, err)
	assert.False(t, errors.Is(err, gpu.ErrUnavailable))
	var exitErr *exec.ExitError
	assert.True(t, errors.As(err, &exitErr))
	assert.Contains(t, err.Error(), "efsmi exited")
}
