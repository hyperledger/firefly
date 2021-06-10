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
	"context"
	"database/sql/driver"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
)

// FFTime is serialized to JSON on the API in RFC3339 nanosecond UTC time
// (noting that JavaScript can parse this format happily into millisecond time with Date.pase()).
// It is persisted as a nanosecond resolution timestamp in the database.
// It can be parsed from RFC3339, or unix timestamps (second, millisecond or nanosecond resolution)
type FFTime time.Time

// FFDuration is serialized to JSON in the string format of time.Duration
// It can be unmarshalled from a number, or a string.
// - If it is a string in time.Duration format, that will be used
// - If it is a string that can be parsed as an int64, that will be used in Milliseconds
// - If it is a number, that will be used in Milliseconds
type FFDuration time.Duration

func Now() *FFTime {
	t := FFTime(time.Now().UTC())
	return &t
}

func ZeroTime() FFTime {
	return FFTime(time.Time{}.UTC())
}

func UnixTime(unixTime int64) *FFTime {
	if unixTime < 1e10 {
		unixTime *= 1e3 // secs to millis
	}
	if unixTime < 1e15 {
		unixTime *= 1e6 // millis to nanos
	}
	t := FFTime(time.Unix(0, unixTime))
	return &t
}

func (ft *FFTime) MarshalJSON() ([]byte, error) {
	if ft == nil || time.Time(*ft).IsZero() {
		return json.Marshal(nil)
	}
	return json.Marshal(ft.String())
}

func ParseString(str string) (*FFTime, error) {
	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		var unixTime int64
		unixTime, err = strconv.ParseInt(str, 10, 64)
		if err == nil {
			return UnixTime(unixTime), nil
		}
	}
	if err != nil {
		zero := ZeroTime()
		return &zero, i18n.NewError(context.Background(), i18n.MsgTimeParseFail, str)
	}
	ft := FFTime(t)
	return &ft, nil
}

func (ft *FFTime) UnixNano() int64 {
	if ft == nil {
		return 0
	}
	return time.Time(*ft).UnixNano()
}

func (ft *FFTime) UnmarshalText(b []byte) error {
	t, err := ParseString(string(b))
	if err != nil {
		return err
	}
	*ft = *t
	return nil
}

// Scan implements sql.Scanner
func (ft *FFTime) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		*ft = ZeroTime()
		return nil

	case string:
		t, err := ParseString(src)
		if err != nil {
			return err
		}
		*ft = *t
		return nil

	case int64:
		if src == 0 {
			return nil
		}
		t := UnixTime(src)
		*ft = *t
		return nil

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, ft)
	}

}

// Value implements sql.Valuer
func (ft FFTime) Value() (driver.Value, error) {
	if time.Time(ft).IsZero() {
		return int64(0), nil
	}
	return ft.UnixNano(), nil
}

func (ft FFTime) String() string {
	if time.Time(ft).IsZero() {
		return ""
	}
	return time.Time(ft).UTC().Format(time.RFC3339Nano)
}

// ParseToDuration is a standard handling of any duration string, in config or API options
func ParseToDuration(durationString string) time.Duration {
	if durationString == "" {
		return time.Duration(0)
	}
	ffd, err := ParseDurationString(durationString)
	if err != nil {
		log.L(context.Background()).Warn(err)
	}
	return time.Duration(ffd)
}

// ParseDurationString is a standard handling of any duration string, in config or API options
func ParseDurationString(durationString string) (FFDuration, error) {
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		intVal, err := strconv.ParseInt(durationString, 10, 64)
		if err != nil {
			return 0, i18n.NewError(context.Background(), i18n.MsgDurationParseFail, durationString)
		}
		// Default is milliseconds for all durations
		duration = time.Duration(intVal) * time.Millisecond
	}
	return FFDuration(duration), nil
}

func (fd *FFDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*fd).String())
}

func (fd *FFDuration) UnmarshalJSON(b []byte) error {
	var stringVal string
	err := json.Unmarshal(b, &stringVal)
	if err != nil {
		var intVal int64
		err = json.Unmarshal(b, &intVal)
		if err != nil {
			return err
		}
		*fd = FFDuration(intVal) * FFDuration(time.Millisecond)
		return nil
	}
	duration, err := ParseDurationString(stringVal)
	if err != nil {
		return err
	}
	*fd = duration
	return nil
}

// Scan implements sql.Scanner
func (fd *FFDuration) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		*fd = 0
		return nil

	case string:
		duration, err := ParseDurationString(src)
		if err != nil {
			return err
		}
		*fd = duration
		return nil

	case int:
		*fd = FFDuration(time.Duration(src) * time.Millisecond)
		return nil

	case int64:
		*fd = FFDuration(time.Duration(src) * time.Millisecond)
		return nil

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, fd)
	}

}

// Value implements sql.Valuer
func (fd *FFDuration) Value() (driver.Value, error) {
	return fd.String(), nil
}

func (fd *FFDuration) String() string {
	if fd == nil {
		return ""
	}
	return time.Duration(*fd).String()
}
