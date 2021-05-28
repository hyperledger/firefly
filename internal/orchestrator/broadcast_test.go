// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDatatypeBadType(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Validator: fftypes.ValidatorType("wrong"),
	})
	assert.Regexp(t, "FF10132.*validator", err)
}

func TestBroadcastDatatypeNSGetFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Name:      "name1",
		Namespace: "ns1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{}`),
	})
	assert.EqualError(t, err, "pop")
}
func TestBroadcastDatatypeBadValue(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	or.mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`!unparsable`),
	})
	assert.Regexp(t, "FF10137.*value", err)
}

func TestBroadcastUpsertFail(t *testing.T) {
	or := newTestOrchestrator()

	or.mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	or.mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)

	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastDatatypeInvalid(t *testing.T) {
	or := newTestOrchestrator()
	or.nodeIDentity = "0x12345"

	or.mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	or.mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastBroadcastFail(t *testing.T) {
	or := newTestOrchestrator()
	or.nodeIDentity = "0x12345"

	or.mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	or.mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	or.mbm.On("BroadcastMessage", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastOk(t *testing.T) {
	or := newTestOrchestrator()
	or.nodeIDentity = "0x12345"

	or.mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	or.mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	or.mbm.On("BroadcastMessage", mock.Anything, mock.Anything).Return(nil)

	_, err := or.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.NoError(t, err)
}

func TestBroadcastNamespaceBadName(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	_, err := or.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name: "!ns",
	})
	assert.Regexp(t, "FF10131.*name", err)
}

func TestBroadcastNamespaceDescriptionTooLong(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	buff := strings.Builder{}
	buff.Grow(4097)
	for i := 0; i < 4097; i++ {
		buff.WriteByte(byte('a' + i%26))
	}
	_, err := or.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name:        "ns1",
		Description: buff.String(),
	})
	assert.Regexp(t, "FF10188.*description", err)
}

func TestBroadcastNamespaceBroadcastOk(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	or.mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	or.mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	or.mbm.On("BroadcastMessage", mock.Anything, mock.Anything).Return(nil)
	buff := strings.Builder{}
	buff.Grow(4097)
	for i := 0; i < 4097; i++ {
		buff.WriteByte(byte('a' + i%26))
	}
	_, err := or.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name:        "ns1",
		Description: "my namespace",
	})
	assert.NoError(t, err)
}

func TestBroadcastMessageOk(t *testing.T) {
	or := newTestOrchestrator()

	ctx := context.Background()
	rag := or.mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	or.mbi.On("VerifyIdentitySyntax", ctx, "0x12345").Return("0x12345", nil)
	or.mdm.On("ResolveInputData", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, nil)
	or.mbm.On("BroadcastMessage", ctx, mock.Anything).Return(nil)

	msg, err := or.BroadcastMessage(ctx, "ns1", &fftypes.MessageInput{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Author: "0x12345",
			},
		},
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"hello": "world"}`)},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, msg.Data[0].ID)
	assert.NotNil(t, msg.Data[0].Hash)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	or.mdi.AssertExpectations(t)
	or.mdm.AssertExpectations(t)
	or.mbm.AssertExpectations(t)
}

func TestBroadcastMessageBadAuthor(t *testing.T) {
	or := newTestOrchestrator()

	ctx := context.Background()
	or.mbi.On("VerifyIdentitySyntax", ctx, mock.Anything).Return("", fmt.Errorf("pop"))

	_, err := or.BroadcastMessage(ctx, "ns1", &fftypes.MessageInput{
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"hello": "world"}`)},
		},
	})
	assert.Regexp(t, "FF10206.*pop", err)
}

func TestBroadcastMessageBadInput(t *testing.T) {
	or := newTestOrchestrator()

	ctx := context.Background()
	or.mbi.On("VerifyIdentitySyntax", ctx, mock.Anything).Return("0x12345", nil)
	rag := or.mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	or.mdm.On("ResolveInputData", ctx, "ns1", mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := or.BroadcastMessage(ctx, "ns1", &fftypes.MessageInput{
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"hello": "world"}`)},
		},
	})
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
	or.mdm.AssertExpectations(t)
}
