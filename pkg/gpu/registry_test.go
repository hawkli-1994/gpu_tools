package gpu

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLoader struct {
	vendor string
}

func (l *testLoader) Load(context.Context) ([]GPUInfo, error) {
	return nil, nil
}

func (l *testLoader) Vendor() string {
	return l.vendor
}

func TestRegistry(t *testing.T) {
	first := &testLoader{vendor: "B"}
	second := &testLoader{vendor: "A"}
	registry, err := NewRegistry(first, second)
	require.NoError(t, err)

	loader, ok := registry.Lookup("A")
	assert.True(t, ok)
	assert.Same(t, second, loader)
	loaders := registry.Loaders()
	require.Len(t, loaders, 2)
	assert.Equal(t, "A", loaders[0].Vendor())
	assert.Equal(t, "B", loaders[1].Vendor())

	err = registry.Register(&testLoader{vendor: "A"})
	assert.ErrorIs(t, err, ErrDuplicateLoader)
	assert.Error(t, registry.Register(nil))
	assert.Error(t, registry.Register(&testLoader{}))
}

func TestRegistryZeroValue(t *testing.T) {
	var registry Registry
	require.NoError(t, registry.Register(&testLoader{vendor: "test"}))
	assert.Len(t, registry.Loaders(), 1)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if err := registry.Register(&testLoader{vendor: fmt.Sprintf("vendor-%d", i)}); err != nil {
				t.Errorf("register loader: %v", err)
			}
		}(i)
		go func() {
			defer wg.Done()
			_ = registry.Loaders()
		}()
	}
	wg.Wait()
	assert.Len(t, registry.Loaders(), 20)
}

func TestParseHelpers(t *testing.T) {
	floatValue, err := ParseFloat("1.5")
	require.NoError(t, err)
	assert.Equal(t, Float64(1.5), floatValue)
	uintValue, err := ParseUint("42")
	require.NoError(t, err)
	assert.Equal(t, Uint64(42), uintValue)

	for _, value := range []string{"", "N/A", "n/a"} {
		floatValue, err = ParseFloat(value)
		require.NoError(t, err)
		assert.Nil(t, floatValue)
		uintValue, err = ParseUint(value)
		require.NoError(t, err)
		assert.Nil(t, uintValue)
	}
	for _, value := range []string{"bad", "NaN", "Inf", "-Inf"} {
		_, err = ParseFloat(value)
		assert.Error(t, err)
	}
	_, err = ParseUint("bad")
	assert.Error(t, err)
}
