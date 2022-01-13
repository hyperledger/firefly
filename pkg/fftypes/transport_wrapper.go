// Copyright © 2022 Kaleido, Inc.
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

package fftypes

type TransportPayloadType = FFEnum

var (
	TransportPayloadTypeMessage TransportPayloadType = ffEnum("transportpayload", "message")
	TransportPayloadTypeBatch   TransportPayloadType = ffEnum("transportpayload", "batch")
)

// TransportWrapper wraps paylaods over data exchange transfers, for easy deserialization at target
type TransportWrapper struct {
	Type    TransportPayloadType `json:"type" ffenum:"transportpayload"`
	Message *Message             `json:"message,omitempty"`
	Data    []*Data              `json:"data,omitempty"`
	Batch   *Batch               `json:"batch,omitempty"`
	Group   *Group               `json:"group,omitempty"`
}

// Manifest lists the contents of the transmission in a Manifest, which can be compared with
// a signed receipt provided back by the DX plugin
func (tw *TransportWrapper) Manifest() *Manifest {
	if tw.Type == TransportPayloadTypeBatch {
		return tw.Batch.Manifest()
	} else if tw.Type == TransportPayloadTypeMessage {
		tm := &Manifest{
			Messages: []MessageRef{},
			Data:     make([]DataRef, len(tw.Data)),
		}
		if tw.Message != nil {
			tm.Messages = []MessageRef{
				{
					ID:   tw.Message.Header.ID,
					Hash: tw.Message.Hash,
				},
			}
		}
		for i, d := range tw.Data {
			tm.Data[i].ID = d.ID
			tm.Data[i].Hash = d.Hash
		}
		return tm
	}
	return nil
}

type TransportStatusUpdate struct {
	Error    string `json:"error,omitempty"`
	Manifest string `json:"manifest,omitempty"`
	Info     string `json:"info,omitempty"`
}
