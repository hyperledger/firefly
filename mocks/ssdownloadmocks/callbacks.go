// Code generated by mockery v1.0.0. DO NOT EDIT.

package ssdownloadmocks

import (
	fftypes "github.com/hyperledger/firefly/pkg/fftypes"
	mock "github.com/stretchr/testify/mock"
)

// Callbacks is an autogenerated mock type for the Callbacks type
type Callbacks struct {
	mock.Mock
}

// SharedStorageBLOBDownloaded provides a mock function with given fields: hash, size, payloadRef
func (_m *Callbacks) SharedStorageBLOBDownloaded(hash fftypes.Bytes32, size int64, payloadRef string) error {
	ret := _m.Called(hash, size, payloadRef)

	var r0 error
	if rf, ok := ret.Get(0).(func(fftypes.Bytes32, int64, string) error); ok {
		r0 = rf(hash, size, payloadRef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SharedStorageBatchDownloaded provides a mock function with given fields: payloadRef, ns, data
func (_m *Callbacks) SharedStorageBatchDownloaded(payloadRef string, ns string, data []byte) (*fftypes.UUID, error) {
	ret := _m.Called(payloadRef, ns, data)

	var r0 *fftypes.UUID
	if rf, ok := ret.Get(0).(func(string, string, []byte) *fftypes.UUID); ok {
		r0 = rf(payloadRef, ns, data)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fftypes.UUID)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, []byte) error); ok {
		r1 = rf(payloadRef, ns, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}