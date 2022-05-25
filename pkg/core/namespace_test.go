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

package core

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceValidation(t *testing.T) {

	ns := &Namespace{
		Name: "!wrong",
	}
	assert.Regexp(t, "FF00140.*name", ns.Validate(context.Background(), false))

	ns = &Namespace{
		Name:        "ok",
		Description: string(make([]byte, 4097)),
	}
	assert.Regexp(t, "FF00135.*description", ns.Validate(context.Background(), false))

	ns = &Namespace{
		Name:        "ok",
		Description: "ok",
	}
	assert.NoError(t, ns.Validate(context.Background(), false))

	assert.Regexp(t, "FF00114", ns.Validate(context.Background(), true))

	var nsDef Definition = ns
	assert.Equal(t, "358de1708c312f6b9eb4c44e0d9811c6f69bf389871d38dd7501992b2c00b557", nsDef.Topic())
	nsDef.SetBroadcastMessage(fftypes.NewUUID())
	assert.NotNil(t, ns.Message)

}
