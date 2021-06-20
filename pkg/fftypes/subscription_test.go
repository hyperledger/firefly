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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionOptionsDatabaseSerialization(t *testing.T) {

	firstEvent := SubOptsFirstEventNewest
	readAhead := uint16(50)
	yes := true
	sub1 := &Subscription{
		Options: SubscriptionOptions{
			SubscriptionCoreOptions: SubscriptionCoreOptions{
				FirstEvent: &firstEvent,
				ReadAhead:  &readAhead,
				WithData:   &yes,
			},
		},
	}
	sub1.Options.TransportOptions()["my-nested-opts"] = map[string]interface{}{
		"myopt1": 12345,
		"myopt2": "test",
	}

	// Verify it serializes as bytes to the database
	b1, err := sub1.Options.Value()
	assert.NoError(t, err)
	assert.Equal(t, `{"firstEvent":"newest","my-nested-opts":{"myopt1":12345,"myopt2":"test"},"readAhead":50,"withData":true}`, string(b1.([]byte)))

	// Verify it restores ok
	sub2 := &Subscription{}
	err = sub2.Options.Scan(b1)
	assert.NoError(t, err)
	b2, err := sub1.Options.Value()
	assert.NoError(t, err)
	assert.Equal(t, SubOptsFirstEventNewest, *sub2.Options.FirstEvent)
	assert.Equal(t, uint16(50), *sub2.Options.ReadAhead)
	assert.Equal(t, string(b1.([]byte)), string(b2.([]byte)))

	// Confirm we don't pass core options, to transports
	assert.Nil(t, sub2.Options.TransportOptions()["withData"])
	assert.Nil(t, sub2.Options.TransportOptions()["firstEvent"])
	assert.Nil(t, sub2.Options.TransportOptions()["readAhead"])

	// Confirm we get back the transport options
	assert.Equal(t, float64(12345), sub2.Options.TransportOptions().GetObject("my-nested-opts")["myopt1"])
	assert.Equal(t, "test", sub2.Options.TransportOptions().GetObject("my-nested-opts")["myopt2"])

	// Verify it can also scan as a string
	err = sub2.Options.Scan(string(b1.([]byte)))
	assert.NoError(t, err)

	// Out of luck with anything else
	err = sub2.Options.Scan(false)
	assert.Regexp(t, "FF10125", err)

}

func TestSubscriptionUnMarshalFail(t *testing.T) {

	b, err := json.Marshal(&SubscriptionOptions{})
	assert.NoError(t, err)
	assert.Equal(t, `{}`, string(b))

	err = json.Unmarshal([]byte(`!badjson`), &SubscriptionOptions{})
	assert.Regexp(t, "invalid", err)

	err = json.Unmarshal([]byte(`{"readAhead": "!a number"}`), &SubscriptionOptions{})
	assert.Regexp(t, "readAhead", err)

}
