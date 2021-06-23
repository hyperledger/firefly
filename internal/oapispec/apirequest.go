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

package oapispec

import (
	"context"
	"net/http"

	"github.com/hyperledger-labs/firefly/internal/orchestrator"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type APIRequest struct {
	Ctx    context.Context
	Or     orchestrator.Orchestrator
	Req    *http.Request
	QP     map[string]string
	PP     map[string]string
	FP     map[string]string
	Filter database.AndFilter
	Input  interface{}
	Part   *fftypes.Multipart
}
