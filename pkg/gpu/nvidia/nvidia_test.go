package nvidia

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUsesExpectedArguments(t *testing.T) {
	tempDir := t.TempDir()
	fixturePath, err := filepath.Abs("testdata/nvidia.txt")
	require.NoError(t, err)
	argsPath := filepath.Join(tempDir, "args")
	commandPath := filepath.Join(tempDir, "nvidia-smi")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$NVIDIA_SMI_ARGS"
/bin/cat "$NVIDIA_SMI_FIXTURE"
`
	require.NoError(t, os.WriteFile(commandPath, []byte(script), 0o755))
	t.Setenv("PATH", tempDir)
	t.Setenv("NVIDIA_SMI_ARGS", argsPath)
	t.Setenv("NVIDIA_SMI_FIXTURE", fixturePath)

	infos, err := New().Load(context.Background())
	require.NoError(t, err)
	require.Len(t, infos, 4)
	assert.Equal(t, "NVIDIA L20", infos[0].CardModel)

	output, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	args := strings.Split(strings.TrimSpace(string(output)), "\n")
	assert.Equal(t, []string{
		"--format=csv,noheader",
		"--query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu,pci.bus_id",
	}, args)
}

func TestDriverInfoUsesExpectedArguments(t *testing.T) {
	tempDir := t.TempDir()
	fixturePath, err := filepath.Abs("testdata/nvidia_version.txt")
	require.NoError(t, err)
	argsPath := filepath.Join(tempDir, "args")
	commandPath := filepath.Join(tempDir, "nvidia-smi")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$NVIDIA_SMI_ARGS"
/bin/cat "$NVIDIA_SMI_VERSION_FIXTURE"
`
	require.NoError(t, os.WriteFile(commandPath, []byte(script), 0o755))
	t.Setenv("PATH", tempDir)
	t.Setenv("NVIDIA_SMI_ARGS", argsPath)
	t.Setenv("NVIDIA_SMI_VERSION_FIXTURE", fixturePath)

	info, err := New().DriverInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "550.54.14", info.Version)
	assert.True(t, info.Installed)
	assert.Equal(t, commandPath, info.InstallPath)

	output, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "--version\n", string(output))
}

func TestCommandUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, test := range []struct {
		name string
		op   string
		run  func(context.Context) error
	}{
		{name: "load", op: "load", run: func(ctx context.Context) error {
			_, err := New().Load(ctx)
			return err
		}},
		{name: "driver info", op: "driver info", run: func(ctx context.Context) error {
			_, err := New().DriverInfo(ctx)
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.run(context.Background())
			assert.ErrorIs(t, err, gpu.ErrUnavailable)
			var opErr *gpu.OpError
			require.ErrorAs(t, err, &opErr)
			assert.Equal(t, test.op, opErr.Op)
		})
	}
}

func TestCommandContextCanceled(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "nvidia-smi")
	require.NoError(t, os.WriteFile(commandPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", tempDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New().Load(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = New().DriverInfo(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestParse(t *testing.T) {
	csvData := `0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 1309 MiB, 0 %, 41, 00000000:50:00.0
1, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB, 13625 MiB, 5 %, 39, 00000000:51:00.0`
	infos, err := (&nvidiaSMICommand{}).parse([]byte(csvData))
	require.NoError(t, err)
	require.Len(t, infos, 2)

	first := infos[0]
	assert.Equal(t, 0, first.Num)
	assert.Equal(t, "0", first.DeviceID)
	assert.Equal(t, "NVIDIA GeForce RTX 4080 SUPER", first.CardModel)
	assert.Equal(t, "NVIDIA", first.CardVendor)
	assert.Equal(t, "NVIDIA", first.CardSeries)
	assert.Equal(t, "00000000:50:00.0", first.PCIBus)
	assert.Equal(t, gpu.Uint64(17171480576), first.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(1372585984), first.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), first.GPUUse)
	assert.Equal(t, gpu.Float64(41), first.TemperatureEdge)
	assert.Nil(t, first.TemperatureJunction)
	assert.Nil(t, first.TemperatureMemory)
	assert.Nil(t, first.AverageGraphicsPackagePower)

	second := infos[1]
	assert.Equal(t, 1, second.Num)
	assert.Equal(t, gpu.Uint64(14286848000), second.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(5), second.GPUUse)
	assert.Equal(t, gpu.Float64(39), second.TemperatureEdge)
}

func TestParseInvalidData(t *testing.T) {
	_, err := (&nvidiaSMICommand{}).parse([]byte(`0, NVIDIA GeForce RTX 4080 SUPER, 16376 MiB`))
	assert.ErrorContains(t, err, "expected 7 columns")
}

func TestParseInvalidMetric(t *testing.T) {
	_, err := (&nvidiaSMICommand{}).parse([]byte(`0, NVIDIA, invalid MiB, 0 MiB, 0 %, 41, 00000000:50:00.0`))
	assert.ErrorContains(t, err, "total memory")
}

func TestParseMetric(t *testing.T) {
	for _, test := range []struct {
		value    string
		expected *float64
	}{
		{value: ""},
		{value: "N/A"},
		{value: "n/a"},
		{value: "12.5 %", expected: gpu.Float64(12.5)},
	} {
		t.Run(test.value, func(t *testing.T) {
			parsed, err := parseMetric(test.value)
			require.NoError(t, err)
			assert.Equal(t, test.expected, parsed)
		})
	}
}

func TestParseMetricRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"12", "12 % extra", "N/A %", "12 percent"} {
		t.Run(value, func(t *testing.T) {
			parsed, err := parseMetric(value)
			assert.Nil(t, parsed)
			assert.ErrorContains(t, err, "expected percentage or N/A")
		})
	}
}

func TestParseRejectsNegativeGPUIndex(t *testing.T) {
	_, err := (&nvidiaSMICommand{}).parse([]byte(`-1, NVIDIA, 16376 MiB, 0 MiB, 0 %, 41, 00000000:50:00.0`))
	assert.ErrorContains(t, err, "GPU index")
	assert.ErrorContains(t, err, "non-negative")
}

func TestParseEmptyData(t *testing.T) {
	infos, err := (&nvidiaSMICommand{}).parse(nil)
	assert.NoError(t, err)
	assert.Empty(t, infos)
}

func TestCommandErrorsIncludeDiagnosticOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nvidia-smi")
	script := "#!/bin/sh\necho 'driver communication failed' >&2\nexit 1\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)

	_, err := New().Load(context.Background())
	assert.ErrorContains(t, err, "driver communication failed")
	_, err = New().DriverInfo(context.Background())
	assert.ErrorContains(t, err, "driver communication failed")
}

//go:embed testdata/nvidia_version.txt
var versionInfo string

func TestParseVersion(t *testing.T) {
	info, err := ParseVersion(versionInfo)
	require.NoError(t, err)
	assert.Equal(t, "550.54.14", info.ClientVersion)
	assert.Equal(t, "550.54.14", info.Version)
	assert.Equal(t, "12.4", info.LibVersion)
}
