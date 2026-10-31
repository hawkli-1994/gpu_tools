package dl

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDLSMIOutput(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "dlsmi_output.xml"))
	require.NoError(t, err)
	infos, err := parseDLSMIOutput(data)
	require.NoError(t, err)
	require.Len(t, infos, 8)

	first := infos[0]
	assert.Equal(t, 0, first.Num)
	assert.Equal(t, "0x00061E27", first.DeviceID)
	assert.Equal(t, "0.3.16", first.DeviceRev)
	assert.Equal(t, "Goldwasser", first.CardVendor)
	assert.Equal(t, "KS38 QUAD-3", first.CardModel)
	assert.Equal(t, "DLIv2", first.CardSeries)
	assert.Equal(t, "GDF00189C02", first.CardSKU)
	assert.Equal(t, "00000000:1E:00.0", first.PCIBus)
	assert.Equal(t, "GDF00189C01DE25300141", first.SerialNumber)
	assert.Equal(t, gpu.Uint64(32768*1024*1024), first.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(293*1024*1024), first.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), first.GPUUse)
	assert.Equal(t, gpu.Float64(6.29), first.AverageGraphicsPackagePower)
	assert.Equal(t, gpu.Float64(53), first.TemperatureEdge)
	assert.Nil(t, first.TemperatureJunction)
	assert.Equal(t, gpu.Float64(53), first.TemperatureMemory)

	assert.Equal(t, 1, infos[1].Num)
	assert.Equal(t, "00000000:1F:00.0", infos[1].PCIBus)
	assert.Equal(t, "KS38 QUAD-2", infos[1].CardModel)
	assert.Equal(t, 7, infos[7].Num)
	assert.Equal(t, "00000000:30:00.0", infos[7].PCIBus)
	assert.Equal(t, "KS38 QUAD-1", infos[7].CardModel)
}

func TestParseDLSMIOutputMissingNumericValues(t *testing.T) {
	infos, err := parseDLSMIOutput([]byte(`<dlsmi_log><gpu id="0"><memory_usage><total>N/A</total><used></used></memory_usage><utilization><gpu>N/A</gpu></utilization></gpu></dlsmi_log>`))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Nil(t, infos[0].VRAMTotalMemory)
	assert.Nil(t, infos[0].VRAMTotalUsedMemory)
	assert.Nil(t, infos[0].GPUUse)
	assert.Nil(t, infos[0].TemperatureEdge)
}

func TestParseDLSMIOutputInvalidNumericValue(t *testing.T) {
	_, err := parseDLSMIOutput([]byte(`<dlsmi_log><gpu id="0"><utilization><gpu>invalid %</gpu></utilization></gpu></dlsmi_log>`))
	assert.Error(t, err)
}

func TestParseDLSMIOutputDeviceNumberUnitsAndCount(t *testing.T) {
	infos, err := parseDLSMIOutput([]byte(`<dlsmi_log><attached_gpus>1</attached_gpus><gpu id="gpu7"><minor_number>7</minor_number></gpu></dlsmi_log>`))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, 7, infos[0].Num)

	for _, input := range []string{
		`<dlsmi_log><gpu id="0"><utilization><gpu>10 C</gpu></utilization></gpu></dlsmi_log>`,
		`<dlsmi_log><gpu id="0"><temperature><gpu_current_temp>53 W</gpu_current_temp></temperature></gpu></dlsmi_log>`,
		`<dlsmi_log><gpu id="0"><power_readings><power_draw>10 C</power_draw></power_readings></gpu></dlsmi_log>`,
		`<dlsmi_log><gpu id="0"><memory_usage><total>1 MiB trailing</total></memory_usage></gpu></dlsmi_log>`,
		`<dlsmi_log><attached_gpus>2</attached_gpus><gpu id="0"/></dlsmi_log>`,
	} {
		_, err := parseDLSMIOutput([]byte(input))
		assert.Error(t, err, input)
	}
}

func TestLoadDoesNotRetryCommandExitFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dlsmi")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho 'dlsmi exited' >&2\nexit 23\n"), 0o755))
	t.Setenv("PATH", dir)

	_, err := New().Load(context.Background())
	require.Error(t, err)
	assert.False(t, errors.Is(err, gpu.ErrUnavailable))
	var exitErr *exec.ExitError
	assert.True(t, errors.As(err, &exitErr))
	assert.Contains(t, err.Error(), "dlsmi exited")
}

func TestLoadCancellationStopsBlockingCommand(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	path := filepath.Join(dir, "dlsmi")
	script := "#!/bin/sh\nprintf started > \"$GPU_TEST_MARKER\"\nexec /bin/sleep 30\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)
	t.Setenv("GPU_TEST_MARKER", marker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := New().Load(ctx)
		done <- err
	}()
	require.Eventually(t, func() bool {
		_, err := os.Stat(marker)
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		var opErr *gpu.OpError
		require.ErrorAs(t, err, &opErr)
		assert.Equal(t, context.Canceled, opErr.Err)
	case <-time.After(3 * time.Second):
		t.Fatal("Load did not return after cancellation")
	}
}
