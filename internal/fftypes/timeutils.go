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
	"context"
	"database/sql/driver"
	"encoding/json"
	"strconv"
	"time"

	"github.com/kaleido-io/firefly/internal/i18n"
)

// FFTime is serialized to JSON on the API in RFC3339 nanosecond UTC time
// (noting that JavaScript can parse this format happily into millisecond time with Date.pase()).
// It is persisted as a nanosecond resolution timestamp in the database.
// It can be parsed from RFC3339, or unix timestamps (second, millisecond or nanosecond resolution)
type FFTime time.Time

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
func (ft *FFTime) Value() (driver.Value, error) {
	if ft == nil || time.Time(*ft).IsZero() {
		return nil, nil
	}
	return ft.UnixNano(), nil
}

func (ft *FFTime) String() string {
	if ft == nil || time.Time(*ft).IsZero() {
		return ""
	}
	return time.Time(*ft).UTC().Format(time.RFC3339Nano)
}
