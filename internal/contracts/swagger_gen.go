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

package contracts

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type ContractAPISwaggerGen interface {
	Generate(ctx context.Context, ffi *fftypes.FFI) (*openapi3.T, error)
}

// contractAPISwaggerGen generates OpenAPI3 (Swagger) definitions for FFIs
type contractAPISwaggerGen struct {
}

func newContractAPISwaggerGen() ContractAPISwaggerGen {
	return &contractAPISwaggerGen{}
}

func (og *contractAPISwaggerGen) Generate(ctx context.Context, ffi *fftypes.FFI) (*openapi3.T, error) {
	return nil, nil
}
