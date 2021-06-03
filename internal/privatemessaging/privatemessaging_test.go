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
	"context"
	"testing"

	"github.com/kaleido-io/firefly/mocks/batchmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/datamocks"
	"github.com/kaleido-io/firefly/mocks/identitymocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestPrivateMessaging(t *testing.T) (*privateMessaging, func()) {

	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mba := &batchmocks.Manager{}
	mdm := &datamocks.Manager{}

	mba.On("RegisterDispatcher", []fftypes.MessageType{fftypes.MessageTypeGroupInit, fftypes.MessageTypePrivate}, mock.Anything, mock.Anything).Return()

	ctx, cancel := context.WithCancel(context.Background())
	pm, err := NewPrivateMessaging(ctx, mdi, mii, mba, mdm)
	assert.NoError(t, err)

	return pm.(*privateMessaging), cancel
}
