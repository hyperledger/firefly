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

package persistence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaleido-io/firefly/internal/i18n"
)

// Filter is the output of the builder
type Filter interface {
	// Sort adds a set of sort conditions (all in a single sort order)
	Sort(...string) Filter

	// Ascending sort order
	Ascending() Filter

	// Descending sort order
	Descending() Filter

	// Skip for pagination
	Skip(uint) Filter

	// Limit for pagination
	Limit(uint) Filter

	// Finalize completes the filter, and for the plugin to validated output structure to convert
	Finalize() (*FilterInfo, error)
}

// FilterOp enum of filter operations that must be implemented by plugins - the string value is
// used in the core string formatting method (for logging etc.)
type FilterOp string

const (
	FilterOpAnd FilterOp = "&&"
	FilterOpOr  FilterOp = "||"
	FilterOpEq  FilterOp = "=="
	FilterOpNe  FilterOp = "!="
	FilterOpGt  FilterOp = ">"
	FilterOpLt  FilterOp = "<"
	FilterOpGte FilterOp = ">="
	FilterOpLte FilterOp = "<="
)

// FilterFactory creates a filter builder in the given context, and contains the rules on
// which fields can be used by the builder (and how they are serialized)
type FilterFactory interface {
	New(ctx context.Context) FilterBuilder
}

// FilterBuilder is the syntax used to build the filter, where And() and Or() can be nested
type FilterBuilder interface {
	// And requires all sub-filters to match
	And(and ...Filter) Filter
	// Or requires any of the sub-filters to match
	Or(and ...Filter) Filter
	// Eq equal
	Eq(name string, value driver.Value) Filter
	// Neq not equal
	Neq(name string, value driver.Value) Filter
	// Lt less than
	Lt(name string, value driver.Value) Filter
	// Gt greater than
	Gt(name string, value driver.Value) Filter
	// Gte greater than or equal
	Gte(name string, value driver.Value) Filter
	// Lte less than or equal
	Lte(name string, value driver.Value) Filter
}

// FilterInfo is the structure returned by Finalize to the plugin, to serialize this filter
// into the underlying persistence mechanism's filter language
type FilterInfo struct {
	Sort       []string
	Skip       uint
	Limit      uint
	Descending bool
	Field      string
	Op         FilterOp
	Value      FilterSerialization
	Children   []*FilterInfo
}

func (f *FilterInfo) String() string {
	var val strings.Builder

	switch f.Op {
	case FilterOpAnd, FilterOpOr:
		cs := make([]string, len(f.Children))
		for i, c := range f.Children {
			cs[i] = fmt.Sprintf("( %s )", c)
		}
		val.WriteString(strings.Join(cs, fmt.Sprintf(" %s ", f.Op)))
	default:
		var s string
		v, _ := f.Value.Value()
		switch tv := v.(type) {
		case int64:
			s = strconv.FormatInt(tv, 10)
		default:
			s = fmt.Sprintf("'%s'", tv)
		}
		val.WriteString(fmt.Sprintf("%s %s %s", f.Field, f.Op, s))
	}

	if len(f.Sort) > 0 {
		val.WriteString(fmt.Sprintf(" sort=%s", strings.Join(f.Sort, ",")))
		if f.Descending {
			val.WriteString(" descending")
		}
	}
	if f.Skip > 0 {
		val.WriteString(fmt.Sprintf(" skip=%d", f.Skip))
	}
	if f.Limit > 0 {
		val.WriteString(fmt.Sprintf(" limit=%d", f.Limit))
	}

	return val.String()
}

type filterDefinition map[string]Filterable

func (fd *filterDefinition) New(ctx context.Context) FilterBuilder {
	return &filterBuilder{
		ctx:           ctx,
		allowedFields: *fd,
	}
}

type filterBuilder struct {
	ctx           context.Context
	allowedFields filterDefinition
}

func (fb *filterBuilder) And(and ...Filter) Filter {
	return &andFilter{
		baseFilter: baseFilter{
			fb:       fb,
			op:       FilterOpAnd,
			children: and,
		},
	}
}

type baseFilter struct {
	fb         *filterBuilder
	children   []Filter
	op         FilterOp
	field      string
	value      interface{}
	sort       []string
	skip       uint
	limit      uint
	descending bool
}

func (f *baseFilter) Finalize() (fi *FilterInfo, err error) {
	var children []*FilterInfo
	var value FilterSerialization

	switch f.op {
	case FilterOpAnd, FilterOpOr:
		children = make([]*FilterInfo, len(f.children))
		for i, c := range f.children {
			if children[i], err = c.Finalize(); err != nil {
				return nil, err
			}
		}
	default:
		name := strings.ToLower(f.field)
		field, ok := f.fb.allowedFields[name]
		if !ok {
			return nil, i18n.NewError(f.fb.ctx, i18n.MsgInvalidFilterField, name)
		}
		value = field.getSerialization()
		if err = value.Scan(f.value); err != nil {
			return nil, i18n.NewError(f.fb.ctx, i18n.MsgInvalidValueForFilterField, name)
		}
	}
	return &FilterInfo{
		Children:   children,
		Op:         f.op,
		Field:      f.field,
		Value:      value,
		Sort:       f.sort,
		Skip:       f.skip,
		Limit:      f.limit,
		Descending: f.descending,
	}, nil
}

func (f *baseFilter) Sort(fields ...string) Filter {
	for _, field := range fields {
		if _, ok := f.fb.allowedFields[field]; ok {
			f.sort = append(f.sort, field)
		}
	}
	return f
}

func (f *baseFilter) Skip(skip uint) Filter {
	f.skip = skip
	return f
}

func (f *baseFilter) Limit(limit uint) Filter {
	f.limit = limit
	return f
}

func (f *baseFilter) Ascending() Filter {
	f.descending = false
	return f
}

func (f *baseFilter) Descending() Filter {
	f.descending = true
	return f
}

type andFilter struct {
	baseFilter
}

func (fb *filterBuilder) Or(or ...Filter) Filter {
	return &orFilter{
		baseFilter: baseFilter{
			fb:       fb,
			op:       FilterOpOr,
			children: or,
		},
	}
}

type orFilter struct {
	baseFilter
}

func (fb *filterBuilder) Eq(name string, value driver.Value) Filter {
	return fb.fieldFilter(FilterOpEq, name, value)
}

func (fb *filterBuilder) Neq(name string, value driver.Value) Filter {
	return fb.fieldFilter(FilterOpNe, name, value)
}

func (fb *filterBuilder) Lt(name string, value driver.Value) Filter {
	return fb.fieldFilter(FilterOpLt, name, value)
}

func (fb *filterBuilder) Gt(name string, value driver.Value) Filter {
	return fb.fieldFilter(FilterOpGt, name, value)
}

func (fb *filterBuilder) Gte(name string, value driver.Value) Filter {
	return fb.fieldFilter(FilterOpGte, name, value)
}

func (fb *filterBuilder) Lte(name string, value driver.Value) Filter {
	return fb.fieldFilter(FilterOpLte, name, value)
}

func (fb *filterBuilder) fieldFilter(op FilterOp, name string, value interface{}) Filter {
	return &fieldFilter{
		baseFilter: baseFilter{
			fb:    fb,
			op:    op,
			field: name,
			value: value,
		},
	}
}

type fieldFilter struct {
	baseFilter
}

// We stand on the shoulders of the well adopted SQL serialization interface here to help us define what
// string<->value looks like, even though this plugin interface is not tightly coupled to SQL.
type FilterSerialization interface {
	driver.Valuer
	sql.Scanner // Implementations can assume the value is ALWAYS a string
}

type Filterable interface {
	getSerialization() FilterSerialization
}

type FilterableString struct{}
type stringValue struct{ s string }

func (f *stringValue) Scan(src interface{}) error                 { f.s = src.(string); return nil }
func (f *stringValue) Value() (driver.Value, error)               { return f.s, nil }
func (f *FilterableString) getSerialization() FilterSerialization { return &stringValue{} }

type FilterableInt64 struct{}
type int64Value struct{ i int64 }

func (f *int64Value) Scan(src interface{}) (err error) {
	f.i, err = strconv.ParseInt(src.(string), 10, 64)
	return err
}
func (f *int64Value) Value() (driver.Value, error)               { return f.i, nil }
func (f *FilterableInt64) getSerialization() FilterSerialization { return &int64Value{} }
