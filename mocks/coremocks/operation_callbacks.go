// Code generated by mockery v2.46.0. DO NOT EDIT.

package coremocks

import (
	context "context"

	core "github.com/hyperledger/firefly/pkg/core"
	mock "github.com/stretchr/testify/mock"
)

// OperationCallbacks is an autogenerated mock type for the OperationCallbacks type
type OperationCallbacks struct {
	mock.Mock
}

// BulkOperationUpdates provides a mock function with given fields: ctx, updates
func (_m *OperationCallbacks) BulkOperationUpdates(ctx context.Context, updates []*core.OperationUpdate) error {
	ret := _m.Called(ctx, updates)

	if len(ret) == 0 {
		panic("no return value specified for BulkOperationUpdates")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*core.OperationUpdate) error); ok {
		r0 = rf(ctx, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// OperationUpdate provides a mock function with given fields: update
func (_m *OperationCallbacks) OperationUpdate(update *core.OperationUpdate) {
	_m.Called(update)
}

// NewOperationCallbacks creates a new instance of OperationCallbacks. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewOperationCallbacks(t interface {
	mock.TestingT
	Cleanup(func())
}) *OperationCallbacks {
	mock := &OperationCallbacks{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
