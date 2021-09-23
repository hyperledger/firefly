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

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/hyperledger/firefly/internal/i18n"
)

// UpdateBuilder is the output of the builder
type UpdateBuilder interface {
	// Set starts creation of a set operation
	Set(field string, value interface{}) Update

	// S starts an update that doesn't have any fields
	S() Update

	// Fields returns the available fields on the update
	Fields() []string
}

type Update interface {
	// Set adds a set condition to the update
	Set(field string, value interface{}) Update

	// IsEmpty
	IsEmpty() bool

	// Finalize completes the update, and for the plugin to validated output structure to convert
	Finalize() (*UpdateInfo, error)
}

// UpdateFactory creates a update builder in the given context, and contains the rules on
// which fields can be used by the builder (and how they are serialized)
type UpdateFactory interface {
	New(ctx context.Context) UpdateBuilder
}

// SetOperation is an individual update action to perform
type SetOperation struct {
	Field string
	Value FieldSerialization
}

// UpdateInfo is the structure returned by Finalize to the plugin, to serialize this uilter
// into the underlying database mechanism's uilter language
type UpdateInfo struct {
	SetOperations []*SetOperation
}

type setOperation struct {
	field string
	value interface{}
}

type updateBuilder struct {
	ctx         context.Context
	queryFields queryFields
}

func (ub *updateBuilder) Fields() []string {
	keys := make([]string, len(ub.queryFields))
	i := 0
	for k := range ub.queryFields {
		keys[i] = k
		i++
	}
	return keys
}

func (ub *updateBuilder) Set(field string, value interface{}) Update {
	return &setUpdate{
		ub:            ub,
		setOperations: []*setOperation{{field, value}},
	}
}

func (ub *updateBuilder) S() Update {
	return &setUpdate{
		ub:            ub,
		setOperations: []*setOperation{},
	}
}

type setUpdate struct {
	ub            *updateBuilder
	setOperations []*setOperation
}

func (u *setUpdate) IsEmpty() bool {
	return len(u.setOperations) == 0
}

func (u *setUpdate) Set(field string, value interface{}) Update {
	u.setOperations = append(u.setOperations, &setOperation{field, value})
	return u
}

func (u *UpdateInfo) String() string {
	var buf strings.Builder
	for i, si := range u.SetOperations {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s=%s", si.Field, valueString(si.Value)))
	}
	return buf.String()
}

func (u *setUpdate) Finalize() (*UpdateInfo, error) {
	ui := &UpdateInfo{
		SetOperations: make([]*SetOperation, len(u.setOperations)),
	}
	for i, si := range u.setOperations {
		name := strings.ToLower(si.field)
		field, ok := u.ub.queryFields[name]
		if !ok {
			return nil, i18n.NewError(u.ub.ctx, i18n.MsgInvalidFilterField, name)
		}
		value := field.getSerialization()
		if err := value.Scan(si.value); err != nil {
			return nil, i18n.WrapError(u.ub.ctx, err, i18n.MsgInvalidValueForFilterField, name)
		}
		ui.SetOperations[i] = &SetOperation{
			Field: name,
			Value: value,
		}
	}
	return ui, nil
}
