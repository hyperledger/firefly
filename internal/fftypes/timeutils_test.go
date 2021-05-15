// Copyright © 2021 Kaleido, Inc.
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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type UTTimeTest struct {
	T1 *FFTime `json:"t1"`
	T2 *FFTime `json:"t2,omitempty"`
	T3 *FFTime `json:"t3,omitempty"`
	T4 *FFTime `json:"t4"`
	T5 *FFTime `json:"t5,omitempty"`
	T6 *FFTime `json:"t6,omitempty"`
	T7 *FFTime `json:"t7,omitempty"`
}

func TestFFTimeJSONSerialization(t *testing.T) {
	now := Now()
	zeroTime := ZeroTime()
	assert.True(t, time.Time(zeroTime).IsZero())
	t6 := UnixTime(1621103852123456789)
	t7 := UnixTime(1621103797)
	utTimeTest := &UTTimeTest{
		T1: nil,
		T2: nil,
		T3: &zeroTime,
		T4: &zeroTime,
		T5: now,
		T6: t6,
		T7: t7,
	}
	b, err := json.Marshal(&utTimeTest)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(
		`{"t1":null,"t3":null,"t4":null,"t5":"%s","t6":"2021-05-15T18:37:32.123456789Z","t7":"2021-05-15T18:36:37Z"}`,
		time.Time(*now).UTC().Format(time.RFC3339Nano)), string(b))

	var utTimeTest2 UTTimeTest
	err = json.Unmarshal(b, &utTimeTest2)
	assert.NoError(t, err)
	assert.Nil(t, utTimeTest.T1)
	assert.Nil(t, utTimeTest.T2)
	assert.Nil(t, nil, utTimeTest.T3)
	assert.Nil(t, nil, utTimeTest.T4)
	assert.Equal(t, *now, *utTimeTest.T5)
	assert.Equal(t, *t6, *utTimeTest.T6)
	assert.Equal(t, *t7, *utTimeTest.T7)
}

func TestFFTimeJSONUnmarshalFail(t *testing.T) {
	var utTimeTest UTTimeTest
	err := json.Unmarshal([]byte(`{"t1": "!Badness"}`), &utTimeTest)
	assert.Regexp(t, "FF10165", err.Error())
}

func TestFFTimeDatabaseSerialization(t *testing.T) {
	now := Now()

	var ft *FFTime
	v, err := ft.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, int64(0), ft.UnixNano())

	ft = now
	v, err = ft.Value()
	assert.NoError(t, err)
	assert.Equal(t, now.UnixNano(), v)

	ft = UnixTime(1621103852123456789)
	v, err = ft.Value()
	assert.NoError(t, err)
	assert.Equal(t, int64(1621103852123456789), v)

	ft = UnixTime(1621103797)
	v, err = ft.Value()
	assert.NoError(t, err)
	assert.Equal(t, int64(1621103797000000000), v)

}

func TestStringNillOrZero(t *testing.T) {
	var ft *FFTime
	assert.Equal(t, "", ft.String())
	zero := ZeroTime()
	ft = &zero
	assert.Equal(t, "", ft.String()) // empty string rather than epoch 1970 time
}

func TestFFTimeParseValue(t *testing.T) {

	var ft FFTime

	// Unix Nanosecs
	err := ft.Scan("1621108144123456789")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-15T19:49:04.123456789Z", ft.String())

	// Unix Millis
	err = ft.Scan("1621108144123")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-15T19:49:04.123Z", ft.String())

	// Unix Secs
	err = ft.Scan("1621108144")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-15T19:49:04Z", ft.String())

	// RFC3339 Secs - timezone to UTC
	err = ft.Scan("2021-05-15T19:49:04-05:00")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-16T00:49:04Z", ft.String())

	// RFC3339 Nanosecs
	err = ft.Scan("2021-05-15T19:49:04.123456789Z")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-15T19:49:04.123456789Z", ft.String())

	// A bad string
	err = ft.Scan("!a supported time format")
	assert.Regexp(t, "FF10165", err.Error())

	// Nil
	err = ft.Scan(nil)
	assert.NoError(t, nil, err)
	assert.True(t, time.Time(ft).IsZero())

	// Zero
	err = ft.Scan(int64(0))
	assert.NoError(t, nil, err)
	assert.True(t, time.Time(ft).IsZero())

	// Unix time
	err = ft.Scan(int64(1621108144123))
	assert.NoError(t, nil, err)
	assert.Equal(t, "2021-05-15T19:49:04.123Z", ft.String())

	// A bad type
	err = ft.Scan(false)
	assert.Regexp(t, "FF10125", err.Error())

}
