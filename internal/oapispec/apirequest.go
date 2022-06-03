// Copyright © 2022 Kaleido, Inc.
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

	"github.com/hyperledger/firefly/internal/namespace"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type APIRequest struct {
	Ctx             context.Context
	RootMgr         namespace.Manager
	Req             *http.Request
	QP              map[string]string
	PP              map[string]string
	FP              map[string]string
	Filter          database.AndFilter
	Input           interface{}
	Part            *core.Multipart
	SuccessStatus   int
	APIBaseURL      string
	ResponseHeaders http.Header
}
