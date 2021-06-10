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

package retry

import (
	"context"
	"time"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
)

const (
	defaultFactor = 2.0
)

// Retry is a concurrency safe retry structure that configures a simple backoff retry mechanism
type Retry struct {
	InitialDelay time.Duration
	MaximumDelay time.Duration
	Factor       float64
}

// DoCustomLog disables the automatic attempt logging, so the caller should do logging for each attempt
func (r *Retry) DoCustomLog(ctx context.Context, f func(attempt int) (retry bool, err error)) error {
	return r.Do(ctx, "", f)
}

// Do invokes the function until the function returns false, or the retry pops.
// This simple interface doesn't pass through errors or return values, on the basis
// you'll be using a closure for that.
func (r *Retry) Do(ctx context.Context, logDescription string, f func(attempt int) (retry bool, err error)) error {
	attempt := 0
	delay := r.InitialDelay
	factor := r.Factor
	if factor < 1 { // Can't reduce
		factor = defaultFactor
	}
	for {
		attempt++
		retry, err := f(attempt)
		if err != nil && logDescription != "" {
			log.L(ctx).Errorf("%s attempt %d: %s", logDescription, attempt, err)
		}
		if !retry || err == nil {
			return err
		}

		// Check the context isn't cancelled
		select {
		case <-ctx.Done():
			return i18n.NewError(ctx, i18n.MsgContextCanceled)
		default:
		}

		// Limit the delay based on the context deadline and maximum delay
		deadline, dok := ctx.Deadline()
		now := time.Now()
		if delay > r.MaximumDelay {
			delay = r.MaximumDelay
		}
		if dok {
			timeleft := deadline.Sub(now)
			if timeleft < delay {
				delay = timeleft
			}
		}

		// Sleep and set the delay for next time
		time.Sleep(delay)
		delay = time.Duration(float64(delay) * factor)
	}
}
