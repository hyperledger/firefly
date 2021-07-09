// Code generated by mockery v1.0.0. DO NOT EDIT.

package assetmocks

import (
	context "context"

	fftypes "github.com/hyperledger-labs/firefly/pkg/fftypes"
	mock "github.com/stretchr/testify/mock"
)

// Manager is an autogenerated mock type for the Manager type
type Manager struct {
	mock.Mock
}

// CreateTokenPool provides a mock function with given fields: ctx, in
func (_m *Manager) CreateTokenPool(ctx context.Context, in *fftypes.TokenPool) (*fftypes.TokenPool, error) {
	ret := _m.Called(ctx, in)

	var r0 *fftypes.TokenPool
	if rf, ok := ret.Get(0).(func(context.Context, *fftypes.TokenPool) *fftypes.TokenPool); ok {
		r0 = rf(ctx, in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fftypes.TokenPool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *fftypes.TokenPool) error); ok {
		r1 = rf(ctx, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodeSigningIdentity provides a mock function with given fields: ctx
func (_m *Manager) GetNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error) {
	ret := _m.Called(ctx)

	var r0 *fftypes.Identity
	if rf, ok := ret.Get(0).(func(context.Context) *fftypes.Identity); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fftypes.Identity)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Start provides a mock function with given fields:
func (_m *Manager) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WaitStop provides a mock function with given fields:
func (_m *Manager) WaitStop() {
	_m.Called()
}
