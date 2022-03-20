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

package operations

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type operationCacheKey struct{}
type operationCache map[string]*fftypes.Operation

func getOperationCache(ctx context.Context) operationCache {
	ctxKey := operationCacheKey{}
	cacheVal := ctx.Value(ctxKey)
	if cacheVal != nil {
		if cache, ok := cacheVal.(operationCache); ok {
			return cache
		}
	}
	return nil
}

func getCacheKey(op *fftypes.Operation) (string, error) {
	opCopy := &fftypes.Operation{
		Namespace:   op.Namespace,
		Transaction: op.Transaction,
		Type:        op.Type,
		Plugin:      op.Plugin,
		Input:       op.Input,
	}
	key, err := json.Marshal(opCopy)
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func CreateOperationRetryContext(ctx context.Context) (ctx1 context.Context) {
	l := log.L(ctx).WithField("opcache", fftypes.ShortID())
	ctx1 = log.WithLogger(ctx, l)
	return context.WithValue(ctx1, operationCacheKey{}, operationCache{})
}

func RunWithOperationCache(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(CreateOperationRetryContext(ctx))
}

func (om *operationsManager) AddOrReuseOperation(ctx context.Context, op *fftypes.Operation) error {
	// If a cache has been created via RunWithOperationCache, detect duplicate operation inserts
	cache := getOperationCache(ctx)
	if cache != nil {
		if cacheKey, err := getCacheKey(op); err == nil {
			if cached, ok := cache[cacheKey]; ok {
				// Identical operation already added in this context
				*op = *cached
				return nil
			}
			if err = om.database.InsertOperation(ctx, op); err != nil {
				return err
			}
			cache[cacheKey] = op
			return nil
		}
	}
	return om.database.InsertOperation(ctx, op)
}
