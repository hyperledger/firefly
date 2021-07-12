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

package broadcast

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (bm *broadcastManager) handleDatatypeBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var dt fftypes.Datatype
	valid = bm.getSystemBroadcastPayload(ctx, msg, data, &dt)
	if !valid {
		return false, nil
	}

	if err = dt.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - validate failed: %s", msg.Header.ID, err)
		return false, nil
	}

	if err = bm.data.CheckDatatype(ctx, dt.Namespace, &dt); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - schema check: %s", msg.Header.ID, err)
		return false, nil
	}

	existing, err := bm.database.GetDatatypeByName(ctx, dt.Namespace, dt.Name, dt.Version)
	if err != nil {
		return false, err // We only return database errors
	}
	if existing != nil {
		l.Warnf("Unable to process datatype broadcast %s (%s:%s) - duplicate of %v", msg.Header.ID, dt.Namespace, dt, existing.ID)
		return false, nil
	}

	if err = bm.database.UpsertDatatype(ctx, &dt, false); err != nil {
		return false, err
	}

	event := fftypes.NewEvent(fftypes.EventTypeDatatypeConfirmed, dt.Namespace, dt.ID)
	if err = bm.database.InsertEvent(ctx, event); err != nil {
		return false, err
	}

	return true, nil
}
