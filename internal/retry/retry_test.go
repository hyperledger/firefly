// Copyright © 2021 Kaleido, Inc.
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryEventuallyOk(t *testing.T) {
	r := Retry{
		MaximumDelay: 3 * time.Microsecond,
		InitialDelay: 1 * time.Microsecond,
	}
	r.Do(context.Background(), func(i int) (retry bool, err error) {
		return i < 10, nil
	})
}

func TestRetryDeadlineTimeout(t *testing.T) {
	r := Retry{
		MaximumDelay: 1 * time.Second,
		InitialDelay: 1 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	err := r.Do(ctx, func(i int) (retry bool, err error) {
		return true, nil
	})
	assert.Regexp(t, "FF10158", err.Error())
}

func TestRetryContextCancellled(t *testing.T) {
	r := Retry{
		MaximumDelay: 1 * time.Second,
		InitialDelay: 1 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := r.Do(ctx, func(i int) (retry bool, err error) {
		return true, nil
	})
	assert.Regexp(t, "FF10158", err.Error())
}
