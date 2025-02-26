// Code generated by mockery v2.46.0. DO NOT EDIT.

package apiservermocks

import (
	context "context"

	ffapi "github.com/hyperledger/firefly-common/pkg/ffapi"
	core "github.com/hyperledger/firefly/pkg/core"

	fftypes "github.com/hyperledger/firefly-common/pkg/fftypes"

	mock "github.com/stretchr/testify/mock"
)

// FFISwaggerGen is an autogenerated mock type for the FFISwaggerGen type
type FFISwaggerGen struct {
	mock.Mock
}

// Build provides a mock function with given fields: ctx, api, ffi
func (_m *FFISwaggerGen) Build(ctx context.Context, api *core.ContractAPI, ffi *fftypes.FFI) (*ffapi.SwaggerGenOptions, []*ffapi.Route) {
	ret := _m.Called(ctx, api, ffi)

	if len(ret) == 0 {
		panic("no return value specified for Build")
	}

	var r0 *ffapi.SwaggerGenOptions
	var r1 []*ffapi.Route
	if rf, ok := ret.Get(0).(func(context.Context, *core.ContractAPI, *fftypes.FFI) (*ffapi.SwaggerGenOptions, []*ffapi.Route)); ok {
		return rf(ctx, api, ffi)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *core.ContractAPI, *fftypes.FFI) *ffapi.SwaggerGenOptions); ok {
		r0 = rf(ctx, api, ffi)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ffapi.SwaggerGenOptions)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *core.ContractAPI, *fftypes.FFI) []*ffapi.Route); ok {
		r1 = rf(ctx, api, ffi)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]*ffapi.Route)
		}
	}

	return r0, r1
}

// NewFFISwaggerGen creates a new instance of FFISwaggerGen. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewFFISwaggerGen(t interface {
	mock.TestingT
	Cleanup(func())
}) *FFISwaggerGen {
	mock := &FFISwaggerGen{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
