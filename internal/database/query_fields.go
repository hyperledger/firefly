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

package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strconv"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
)

// QueryFactory creates a filter builder in the given context, and contains the rules on
// which fields can be used by the builder (and how they are serialized)
type QueryFactory interface {
	NewFilter(ctx context.Context, defLimit uint64) FilterBuilder
	NewUpdate(ctx context.Context) UpdateBuilder
}

type queryFields map[string]Field

func (qf *queryFields) NewFilter(ctx context.Context, defLimit uint64) FilterBuilder {
	return &filterBuilder{
		ctx:         ctx,
		queryFields: *qf,
		limit:       defLimit,
	}
}

func (qf *queryFields) NewUpdate(ctx context.Context) UpdateBuilder {
	return &updateBuilder{
		ctx:         ctx,
		queryFields: *qf,
	}
}

// We stand on the shoulders of the well adopted SQL serialization interface here to help us define what
// string<->value looks like, even though this plugin interface is not tightly coupled to SQL.
type FieldSerialization interface {
	driver.Valuer
	sql.Scanner // Implementations can assume the value is ALWAYS a string
}

type Field interface {
	getSerialization() FieldSerialization
}

type StringField struct{}
type stringField struct{ s string }

func (f *stringField) Scan(src interface{}) error {
	switch tv := src.(type) {
	case string:
		f.s = tv
	case int:
		f.s = strconv.FormatInt(int64(tv), 10)
	case int32:
		f.s = strconv.FormatInt(int64(tv), 10)
	case int64:
		f.s = strconv.FormatInt(int64(tv), 10)
	case uint:
		f.s = strconv.FormatInt(int64(tv), 10)
	case uint32:
		f.s = strconv.FormatInt(int64(tv), 10)
	case uint64:
		f.s = strconv.FormatInt(int64(tv), 10)
	case *uuid.UUID:
		if tv != nil {
			f.s = tv.String()
		}
	case uuid.UUID:
		f.s = tv.String()
	case *fftypes.Bytes32:
		f.s = tv.String()
	case fftypes.Bytes32:
		f.s = tv.String()
	case nil:
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, "")
	}
	return nil
}
func (f *stringField) Value() (driver.Value, error)         { return f.s, nil }
func (f *StringField) getSerialization() FieldSerialization { return &stringField{} }

type Int64Field struct{}
type int64Field struct{ i int64 }

func (f *int64Field) Scan(src interface{}) (err error) {
	switch tv := src.(type) {
	case int:
		f.i = int64(tv)
	case int32:
		f.i = int64(tv)
	case int64:
		f.i = int64(tv)
	case uint:
		f.i = int64(tv)
	case uint32:
		f.i = int64(tv)
	case uint64:
		f.i = int64(tv)
	case string:
		f.i, err = strconv.ParseInt(src.(string), 10, 64)
		if err != nil {
			return i18n.WrapError(context.Background(), err, i18n.MsgScanFailed, src, int64(0))
		}
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, "")
	}
	return nil
}
func (f *int64Field) Value() (driver.Value, error)         { return f.i, nil }
func (f *Int64Field) getSerialization() FieldSerialization { return &int64Field{} }
