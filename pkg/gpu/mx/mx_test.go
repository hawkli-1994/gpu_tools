package mx

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
var output string

//go:embed testdata/meta_2.2.12.txt
var output212 string

//go:embed testdata/debug-mx.txt
var output234 string

func TestParseMxOutput(t *testing.T) {
	infos, err := parseMxOutput(output)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Nil(t, infos[0].TemperatureEdge)
	assert.Equal(t, gpu.Float64(44), infos[0].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(62684897280), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:0f:00.0", infos[0].PCIBus)
}

func TestParseMxOutput212(t *testing.T) {
	infos, err := parseMxOutput(output212)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(38), infos[0].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(62379499520), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:19:00.0", infos[0].PCIBus)
	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, "MXN260", infos[1].CardModel)
	assert.Equal(t, gpu.Float64(37.5), infos[1].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[1].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(2467725312), infos[1].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[1].GPUUse)
	assert.Equal(t, "0000:1a:00.0", infos[1].PCIBus)
}

func TestParseMxOutput234(t *testing.T) {
	infos, err := parseMxOutput(output234)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, 0, infos[0].Num)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(45.75), infos[0].TemperatureJunction)
	assert.Equal(t, gpu.Uint64(68719476736), infos[0].VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(700903424), infos[0].VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), infos[0].GPUUse)
	assert.Equal(t, "0000:0c:00.0", infos[0].PCIBus)
	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, gpu.Float64(44.25), infos[1].TemperatureJunction)
	assert.Equal(t, "0000:0f:00.0", infos[1].PCIBus)
}

func TestParseMxOutputEmptyOrInvalid(t *testing.T) {
	infos, err := parseMxOutput("")
	require.NoError(t, err)
	assert.Empty(t, infos)
	_, err = parseMxOutput("invalid mx-smi output")
	assert.Error(t, err)
}

func TestParseMxOutputRejectsCountMismatchAndInvalidUnits(t *testing.T) {
	for _, input := range []string{
		"Attached GPUs : 2\nGPU#0 MXN260 0000:0f:00.0\n",
		"Attached GPUs : 1\nGPU#0 MXN260 0000:0f:00.0\nGPU#1 MXN260 0000:10:00.0\n",
		"GPU#0 MXN260 0000:0f:00.0\nGPU hotspot: 41 W\n",
		"GPU#0 MXN260 0000:0f:00.0\nvram total: 1024 MB\n",
	} {
		_, err := parseMxOutput(input)
		assert.Error(t, err, input)
	}
}

func TestParseMxOutputInvalidNumericValue(t *testing.T) {
	_, err := parseMxOutput("GPU#0 MXN260 0000:0f:00.0\nGPU hotspot: invalid C\n")
	assert.Error(t, err)
}

func TestMXLoadSuccess(t *testing.T) {
	fixturePath := filepath.Join(t.TempDir(), "output.txt")
	require.NoError(t, os.WriteFile(fixturePath, []byte(output212), 0o600))
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MX_TEST_FIXTURE", fixturePath)
	t.Setenv("MX_TEST_ARGS_FILE", argsPath)
	command := writeMXTestCommand(t, "printf '%s\\n' \"$@\" > \"$MX_TEST_ARGS_FILE\"\n/bin/cat \"$MX_TEST_FIXTURE\"\n")

	infos, err := (&mxCommand{path: command}).Load(context.Background())
	require.NoError(t, err)
	require.Len(t, infos, 2)
	assert.Equal(t, "MXN260", infos[0].CardModel)
	assert.Equal(t, gpu.Float64(38), infos[0].TemperatureJunction)
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "--show-temperature\n--show-usage\n--show-memory\n", string(args))
}

func TestMXLoadCommandFailureIncludesStderr(t *testing.T) {
	command := writeMXTestCommand(t, "printf 'mx-smi failed: permission denied\\n' >&2\nexit 7\n")

	infos, err := (&mxCommand{path: command}).Load(context.Background())
	assert.Nil(t, infos)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mx-smi failed: permission denied")
	assert.Contains(t, err.Error(), "exit status 7")
}

func TestMXLoadContextCanceled(t *testing.T) {
	command := writeMXTestCommand(t, "exit 0\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	infos, err := (&mxCommand{path: command}).Load(ctx)
	assert.Nil(t, infos)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMXLoadToolMissing(t *testing.T) {
	command := filepath.Join(t.TempDir(), "missing-mx-smi")

	infos, err := (&mxCommand{path: command}).Load(context.Background())
	assert.Nil(t, infos)
	assert.ErrorIs(t, err, gpu.ErrUnavailable)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func writeMXTestCommand(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mx-smi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755))
	return path
}
