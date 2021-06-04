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

package privatemessaging

import (
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupInitSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, nil)
	assert.Regexp(t, "FF10137", err)
}

func TestGroupInitWriteFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, &fftypes.Group{})
	assert.Regexp(t, "pop", err)
}
