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

package data

import (
	"context"
	"strings"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/xeipuuv/gojsonschema"
)

type jsonValidator struct {
	id       *fftypes.UUID
	size     int64
	datatype *fftypes.DatatypeRef
	schema   *gojsonschema.Schema
}

func newJSONValidator(ctx context.Context, datatype *fftypes.Datatype) (*jsonValidator, error) {
	jv := &jsonValidator{
		id: datatype.ID,
		datatype: &fftypes.DatatypeRef{
			Name:    datatype.Name,
			Version: datatype.Version,
		},
	}

	schemaBytes := []byte(datatype.Value)
	sl := gojsonschema.NewBytesLoader(schemaBytes)
	schema, err := gojsonschema.NewSchema(sl)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgSchemaLoadFailed, jv.datatype)
	}
	jv.schema = schema
	jv.size = int64(len(schemaBytes))

	return jv, nil
}

func (jv *jsonValidator) Validate(ctx context.Context, data *fftypes.Data) error {
	if data.Value == nil {
		return i18n.NewError(ctx, i18n.MsgDataValueIsNull)
	}

	hash := data.Value.Hash()
	if data.Hash == nil || *hash != *data.Hash {
		return i18n.NewError(ctx, i18n.MsgDataInvalidHash, hash, data.Hash)
	}

	err := jv.validateBytes(ctx, []byte(data.Value))
	if err != nil {
		log.L(ctx).Warnf("JSON schema %s [%v] validation of data %v failed: %s", jv.datatype, jv.id, data.ID, err)
	}
	return err
}

func (jv *jsonValidator) validateBytes(ctx context.Context, b []byte) error {
	res, err := jv.schema.Validate(gojsonschema.NewBytesLoader(b))
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDataCannotBeValidated)
	}
	if !res.Valid() {
		errStrings := make([]string, len(res.Errors()))
		for i, e := range res.Errors() {
			errStrings[i] = e.String()
		}
		errors := strings.Join(errStrings, ",")
		return i18n.NewError(ctx, i18n.MsgJSONDataInvalidPerSchema, jv.datatype, errors)
	}
	return nil
}

func (jv *jsonValidator) Size() int64 {
	return jv.size
}
