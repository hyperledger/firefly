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
	"database/sql/driver"

	"github.com/google/uuid"
)

// UUID is a wrapper on a UUID implementation, ensuring Value handles nil
type UUID uuid.UUID

func ParseUUID(uuidStr string) (*UUID, error) {
	u, err := uuid.Parse(uuidStr)
	uuid := UUID(u)
	return &uuid, err
}

func MustParseUUID(uuidStr string) *UUID {
	uuid := UUID(uuid.MustParse(uuidStr))
	return &uuid
}

func NewUUID() *UUID {
	u := UUID(uuid.New())
	return &u
}

func (u *UUID) String() string {
	if u == nil {
		return ""
	}
	return (*uuid.UUID)(u).String()
}

func (u UUID) MarshalText() ([]byte, error) {
	return (uuid.UUID)(u).MarshalText()
}

func (u *UUID) UnmarshalText(b []byte) error {
	return (*uuid.UUID)(u).UnmarshalText(b)
}

func (u UUID) MarshalBinary() ([]byte, error) {
	return (uuid.UUID)(u).MarshalBinary()
}

func (u *UUID) UnmarshalBinary(b []byte) error {
	return (*uuid.UUID)(u).UnmarshalBinary(b)
}

func (u *UUID) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	return (uuid.UUID)(*u).Value()
}

func (u *UUID) Scan(src interface{}) error {
	return (*uuid.UUID)(u).Scan(src)
}

func (u *UUID) Equals(u2 *UUID) bool {
	switch {
	case u == nil && u2 == nil:
		return true
	case u == nil || u2 == nil:
		return false
	default:
		return *u == *u2
	}
}
