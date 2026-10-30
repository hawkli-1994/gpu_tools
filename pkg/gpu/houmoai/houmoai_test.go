package houmoai

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/hm_smi.txt
var hmSMIOutput []byte

func TestParse(t *testing.T) {
	infos, err := New().parse(hmSMIOutput)
	require.NoError(t, err)
	require.Len(t, infos, 2)

	gpu0 := infos[0]
	assert.Equal(t, 0, gpu0.Num)
	assert.Equal(t, "1", gpu0.DeviceID)
	assert.Equal(t, "Houmo", gpu0.CardVendor)
	assert.Equal(t, "Houmo", gpu0.CardSeries)
	assert.Equal(t, "LQ50-18GB", gpu0.CardModel)
	assert.Equal(t, "100I2010", gpu0.CardSKU)
	assert.Equal(t, "0103020100002025003800000003", gpu0.SerialNumber)
	assert.Equal(t, "V1.1.1", gpu0.DeviceRev)
	assert.Equal(t, "0000:18:00.0", gpu0.PCIBus)
	assert.Equal(t, gpu.Float64(0), gpu0.GPUUse)
	assert.Equal(t, gpu.Uint64(19193135104), gpu0.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(0), gpu0.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(46.8), gpu0.TemperatureEdge)
	assert.Equal(t, gpu.Float64(45.9), gpu0.TemperatureJunction)
	assert.Equal(t, gpu.Float64(46.5), gpu0.TemperatureMemory)
	assert.Equal(t, gpu.Float64(6.7), gpu0.AverageGraphicsPackagePower)

	gpu1 := infos[1]
	assert.Equal(t, 1, gpu1.Num)
	assert.Equal(t, "0", gpu1.DeviceID)
	assert.Equal(t, "LQ50-24GB", gpu1.CardModel)
	assert.Equal(t, "100O2010", gpu1.CardSKU)
	assert.Equal(t, "0000:15:00.0", gpu1.PCIBus)
	assert.Equal(t, gpu.Uint64(25635586048), gpu1.VRAMTotalMemory)
	assert.Equal(t, gpu.Uint64(0), gpu1.VRAMTotalUsedMemory)
	assert.Equal(t, gpu.Float64(0), gpu1.GPUUse)
	assert.Equal(t, gpu.Float64(44.9), gpu1.TemperatureEdge)
	assert.Equal(t, gpu.Float64(41.5), gpu1.TemperatureJunction)
	assert.Equal(t, gpu.Float64(40.9), gpu1.TemperatureMemory)
	assert.Equal(t, gpu.Float64(6.6), gpu1.AverageGraphicsPackagePower)
}

func TestParseEmpty(t *testing.T) {
	infos, err := New().parse(nil)
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestParseMissingNumericValues(t *testing.T) {
	infos, err := New().parse([]byte("device0 detail infos\nDev : 0\n"))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Nil(t, infos[0].GPUUse)
	assert.Nil(t, infos[0].VRAMTotalMemory)
	assert.Nil(t, infos[0].TemperatureMemory)
}

func TestParseInvalidNumericValue(t *testing.T) {
	_, err := New().parse([]byte("device0 detail infos\nAverage_Util : invalid\n"))
	assert.Error(t, err)
}

func TestVendorAndInterface(t *testing.T) {
	loader := New()
	assert.Equal(t, "Houmo", loader.Vendor())
	var _ gpu.Loader = loader
}

func TestLoadDeadlineExceeded(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	path := filepath.Join(dir, "hm_smi")
	script := "#!/bin/sh\nprintf started > \"$GPU_TEST_MARKER\"\nexec /bin/sleep 30\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir)
	t.Setenv("GPU_TEST_MARKER", marker)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
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

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		var opErr *gpu.OpError
		require.ErrorAs(t, err, &opErr)
		assert.Equal(t, context.DeadlineExceeded, opErr.Err)
	case <-time.After(3 * time.Second):
		t.Fatal("Load did not return after deadline")
	}
}
