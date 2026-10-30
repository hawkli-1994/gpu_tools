package amd

import (
	"context"
	"errors"
	"testing"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
	"github.com/stretchr/testify/assert"
)

type fakeLoader struct {
	err   error
	loads int
}

func (f *fakeLoader) Load(context.Context) ([]gpu.GPUInfo, error) {
	f.loads++
	return []gpu.GPUInfo{}, f.err
}

func (f *fakeLoader) Vendor() string {
	return "AMD"
}

func TestAMDCommandPrefersAMDSMI(t *testing.T) {
	amdSMI := &fakeLoader{}
	rocmSMI := &fakeLoader{}
	command := &amdCommand{amdSMI: amdSMI, rocmSMI: rocmSMI}

	_, err := command.Load(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, amdSMI.loads)
	assert.Zero(t, rocmSMI.loads)
}

func TestAMDCommandFallsBackToROCMSMI(t *testing.T) {
	amdSMI := &fakeLoader{err: gpu.ErrUnavailable}
	rocmSMI := &fakeLoader{}
	command := &amdCommand{amdSMI: amdSMI, rocmSMI: rocmSMI}

	_, err := command.Load(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, amdSMI.loads)
	assert.Equal(t, 1, rocmSMI.loads)
}

func TestAMDCommandDoesNotHideAMDSMIErrors(t *testing.T) {
	expected := errors.New("metric failed")
	amdSMI := &fakeLoader{err: expected}
	rocmSMI := &fakeLoader{}
	command := &amdCommand{amdSMI: amdSMI, rocmSMI: rocmSMI}

	_, err := command.Load(context.Background())
	assert.ErrorIs(t, err, expected)
	assert.Zero(t, rocmSMI.loads)
}
