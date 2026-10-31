package xdx

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/output.txt
var output []byte

func TestXDXParse(t *testing.T) {
	expected := []gpu.GPUInfo{
		{
			Num: 0, DeviceID: "0", CardVendor: "XDX", CardModel: "R1900", CardSeries: "PANGU_A0",
			TemperatureEdge: gpu.Float64(35),
			VRAMTotalMemory: gpu.Uint64(16384 * 1024 * 1024), AverageGraphicsPackagePower: gpu.Float64(41), PCIBus: "0000:19:00.0",
		},
		{
			Num: 1, DeviceID: "1", CardVendor: "XDX", CardModel: "R1900", CardSeries: "PANGU_A0",
			TemperatureEdge: gpu.Float64(35),
			VRAMTotalMemory: gpu.Uint64(16384 * 1024 * 1024), AverageGraphicsPackagePower: gpu.Float64(41), PCIBus: "0000:1a:00.0",
		},
	}
	infos, err := New().parse(output)
	require.NoError(t, err)
	assert.Equal(t, expected, infos)
}

func TestXDXParseEmptyData(t *testing.T) {
	infos, err := New().parse(nil)
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestXDXParseInvalidNumericValue(t *testing.T) {
	_, err := New().parse([]byte("GPU 0\nTemperature : invalid C\n"))
	assert.Error(t, err)
}

func TestXDXParseRejectsCountMismatchAndInvalidUnits(t *testing.T) {
	for _, input := range []string{
		"Attached GPUs : 2\nGPU 0\n",
		"Attached GPUs : 1\nGPU 0\nGPU 1\n",
		"GPU 0\nTemperature : 35 W\n",
		"GPU 0\n    Power\n            Pwr Total : 41 C\n",
		"GPU 0\n    FB Memory Usage\n            Total : 1024 MiB trailing\n",
		"unrecognized xdxsmi output",
	} {
		_, err := New().parse([]byte(input))
		assert.Error(t, err, input)
	}
}

func TestXDXLoadSuccess(t *testing.T) {
	fixturePath := filepath.Join(t.TempDir(), "output.txt")
	require.NoError(t, os.WriteFile(fixturePath, output, 0o600))
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("XDX_TEST_FIXTURE", fixturePath)
	t.Setenv("XDX_TEST_ARGS_FILE", argsPath)
	command := writeXDXTestCommand(t, "printf '%s\\n' \"$@\" > \"$XDX_TEST_ARGS_FILE\"\n/bin/cat \"$XDX_TEST_FIXTURE\"\n")
	t.Setenv("PATH", filepath.Dir(command)+string(os.PathListSeparator)+os.Getenv("PATH"))

	infos, err := New().Load(context.Background())
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, "R1900", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(35), infos[0].TemperatureEdge)
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "out\n", string(args))
}

func TestXDXLoadCommandFailureIncludesStderr(t *testing.T) {
	command := writeXDXTestCommand(t, "printf 'xdxsmi failed: device unavailable\\n' >&2\nexit 9\n")
	t.Setenv("PATH", filepath.Dir(command)+string(os.PathListSeparator)+os.Getenv("PATH"))

	infos, err := New().Load(context.Background())
	assert.Nil(t, infos)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "xdxsmi failed: device unavailable")
	assert.Contains(t, err.Error(), "exit status 9")
}

func TestXDXLoadContextCanceled(t *testing.T) {
	command := writeXDXTestCommand(t, "exit 0\n")
	t.Setenv("PATH", filepath.Dir(command)+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	infos, err := New().Load(ctx)
	assert.Nil(t, infos)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestXDXLoadToolMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	infos, err := New().Load(context.Background())
	assert.Nil(t, infos)
	assert.ErrorIs(t, err, gpu.ErrUnavailable)
}

func writeXDXTestCommand(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "xdxsmi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755))
	return path
}
