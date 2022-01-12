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

package fftypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestToString(t *testing.T) {

	tw := &TransportWrapper{
		Type:    TransportPayloadTypeMessage,
		Message: &Message{Header: MessageHeader{ID: MustParseUUID("c38e76ec-92a6-4659-805d-8ae3b7437c40")}, Hash: MustParseBytes32("169ef5233cf44df3d71df59f25928743e9a76378bb1375e06539b732b1fc57e5")},
		Data: []*Data{
			{ID: MustParseUUID("7bc49647-cd1c-4633-98fa-ddbb208d61bd"), Hash: MustParseBytes32("2b849d47e44a291cd83bee4e7ace66178a5245a151d3bbd02011312ec2604ed6")},
			{ID: MustParseUUID("5b80eec3-04b5-4557-bced-6a458ecb9ef2"), Hash: MustParseBytes32("2bcddd992d17e89a5aafbe99c59d954018ddadf4e533a164808ae2389bbf33dc")},
		},
	}
	assert.Equal(t, "{\"messages\":[{\"id\":\"c38e76ec-92a6-4659-805d-8ae3b7437c40\",\"hash\":\"169ef5233cf44df3d71df59f25928743e9a76378bb1375e06539b732b1fc57e5\"}],\"data\":[{\"id\":\"7bc49647-cd1c-4633-98fa-ddbb208d61bd\",\"hash\":\"2b849d47e44a291cd83bee4e7ace66178a5245a151d3bbd02011312ec2604ed6\"},{\"id\":\"5b80eec3-04b5-4557-bced-6a458ecb9ef2\",\"hash\":\"2bcddd992d17e89a5aafbe99c59d954018ddadf4e533a164808ae2389bbf33dc\"}]}", tw.Manifest().String())
}
