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
	assert.Regexp(t, "FF10165", err)
}

func TestNilTimeConversion(t *testing.T) {
	var ft FFTime
	var epoch time.Time
	conversion := ft.Time()

	assert.Equal(t, epoch, *conversion)
}

func TestFFTimeDatabaseSerialization(t *testing.T) {
	now := Now()
	zero := ZeroTime()

	var ft = &zero
	v, err := ft.Value()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), v)

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

func TestStringZero(t *testing.T) {
	var ft *FFTime
	assert.Equal(t, int64(0), ft.UnixNano())
	zero := ZeroTime()
	ft = &zero
	assert.Equal(t, "", ft.String()) // empty string rather than epoch 1970 time
}

func TestSafeEqual(t *testing.T) {
	var ft1 *FFTime
	var ft2 *FFTime
	assert.True(t, ft1.Equal(ft2))
	ft1 = Now()
	assert.False(t, ft1.Equal(ft2))
	ft1 = nil
	ft2 = Now()
	assert.False(t, ft1.Equal(ft2))
	ft1 = Now()
	*ft2 = *ft1
	assert.True(t, ft1.Equal(ft2))
}

func TestFFTimeParseValue(t *testing.T) {

	var ft FFTime

	// Unix Nanosecs
	err := ft.Scan("1621108144123456789")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-15T19:49:04.123456789Z", ft.String())
	assert.Equal(t, "2021-05-15T19:49:04.123456789Z", ft.Time().UTC().Format(time.RFC3339Nano))

	// Unix Millis
	err = ft.Scan("1621108144123")
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-15T19:49:04.123Z", ft.String())
	assert.Equal(t, "2021-05-15T19:49:04.123Z", ft.Time().UTC().Format(time.RFC3339Nano))

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
	assert.Regexp(t, "FF10165", err)

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
	assert.Regexp(t, "FF10125", err)

}

func TestFFDurationJSONSerialization(t *testing.T) {
	var utDurationTest struct {
		D1 *FFDuration `json:"d1"`
		D2 *FFDuration `json:"d2,omitempty"`
		D3 FFDuration  `json:"d3"`
		D4 FFDuration  `json:"d4"`
		D5 FFDuration  `json:"d5"`
		D6 FFDuration  `json:"d6"`
		D7 FFDuration  `json:"d7"`
	}
	jsonToParse := []byte(`
	{
		"d1": null,
		"d3": 12345,
		"d4": "12345",
		"d5": "0",
		"d6": "1h15s",
		"d7": "-24h"
	}
	`)
	err := json.Unmarshal(jsonToParse, &utDurationTest)
	assert.NoError(t, err)
	assert.Nil(t, utDurationTest.D1)
	assert.Nil(t, utDurationTest.D2)
	assert.Equal(t, FFDuration(12345)*FFDuration(time.Millisecond), utDurationTest.D3)
	assert.Equal(t, FFDuration(12345)*FFDuration(time.Millisecond), utDurationTest.D4)
	assert.Equal(t, FFDuration(0), utDurationTest.D5)
	assert.Equal(t, FFDuration(1)*FFDuration(time.Hour)+FFDuration(15)*FFDuration(time.Second), utDurationTest.D6)
	assert.Equal(t, FFDuration(-24)*FFDuration(time.Hour), utDurationTest.D7)

	b, err := json.Marshal(&utDurationTest)
	assert.NoError(t, err)
	assert.Equal(t, `{"d1":null,"d3":"12.345s","d4":"12.345s","d5":"0s","d6":"1h0m15s","d7":"-24h0m0s"}`, string(b))

	jsonToParse = []byte(`{"d1": false}`)
	err = json.Unmarshal(jsonToParse, &utDurationTest)
	assert.Error(t, err)

	jsonToParse = []byte(`{"d1": "1 year"}`)
	err = json.Unmarshal(jsonToParse, &utDurationTest)
	assert.Error(t, err)
}

func TestFFDurationParseValue(t *testing.T) {

	var fd FFDuration

	err := fd.Scan("1h")
	assert.NoError(t, err)
	assert.Equal(t, "1h0m0s", fd.String())

	err = fd.Scan(nil)
	assert.NoError(t, err)
	assert.Equal(t, FFDuration(0), fd)

	err = fd.Scan(12345)
	assert.NoError(t, err)
	assert.Equal(t, FFDuration(12345)*FFDuration(time.Millisecond), fd)

	err = fd.Scan(int64(12345))
	assert.NoError(t, err)
	assert.Equal(t, FFDuration(12345)*FFDuration(time.Millisecond), fd)

	err = fd.Scan(false)
	assert.Regexp(t, "FF10125", err)

	err = fd.Scan("1 year")
	assert.Regexp(t, "FF10167", err)

	var pfd *FFDuration
	v, err := pfd.Value()
	assert.NoError(t, err)
	assert.Equal(t, "", v)

	pfd = &fd
	*pfd = FFDuration(12345) * FFDuration(time.Millisecond)
	v, err = pfd.Value()
	assert.NoError(t, err)
	assert.Equal(t, "12.345s", v)

}

func TestDurationParsing(t *testing.T) {
	assert.Zero(t, ParseToDuration("!a duration"))
	assert.Zero(t, ParseToDuration(""))
	assert.Equal(t, "15ms", ParseToDuration("15ms").String())
}
