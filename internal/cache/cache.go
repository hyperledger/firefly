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

package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/karlseguin/ccache"

	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
)

type CConfig struct {
	ctx               context.Context
	namespace         string
	maxLimitConfigKey config.RootKey
	ttlConfigKey      config.RootKey
}

func NewCacheConfig(ctx context.Context, maxLimitConfigKey config.RootKey, ttlConfigKey config.RootKey, namespace string) *CConfig {
	if namespace == "" {
		namespace = "global"
	}
	cc := &CConfig{
		ctx:               ctx,
		namespace:         namespace,
		maxLimitConfigKey: maxLimitConfigKey,
		ttlConfigKey:      ttlConfigKey,
	}
	return cc
}

func (cc *CConfig) UniqueName() (string, error) {
	category, err := cc.Category()
	if err != nil {
		return "", err
	}
	return cc.namespace + "::" + category, nil
}

func (cc *CConfig) Category() (string, error) {
	if cc.maxLimitConfigKey == "" {
		return "", i18n.NewError(cc.ctx, coremsgs.MsgCacheMissSizeLimitKeyInternal)
	}
	if cc.ttlConfigKey == "" {
		return "", i18n.NewError(cc.ctx, coremsgs.MsgCacheMissTTLKeyInternal)
	}

	categoryDerivedFromMaxLimitConfigKey, _ := parseConfigKeyString(string(cc.maxLimitConfigKey))
	categoryDerivedFromTTLConfigKey, _ := parseConfigKeyString(string(cc.ttlConfigKey))
	if categoryDerivedFromMaxLimitConfigKey != categoryDerivedFromTTLConfigKey {
		return "", i18n.NewError(cc.ctx, coremsgs.MsgCacheConfigKeyMismatchInternal, cc.maxLimitConfigKey, cc.ttlConfigKey, categoryDerivedFromMaxLimitConfigKey, categoryDerivedFromTTLConfigKey)

	}
	return categoryDerivedFromMaxLimitConfigKey, nil
}

func parseConfigKeyString(configKey string) (string, string) {
	keyParts := strings.Split(configKey, ".")
	categoryString := strings.Join(keyParts[:len(keyParts)-1], ".")
	configName := keyParts[len(keyParts)-1]
	return categoryString, configName
}

func (cc *CConfig) MaxSize() (int64, error) {
	_, sizeConfigName := parseConfigKeyString(string(cc.maxLimitConfigKey))
	switch sizeConfigName {
	case "limit":
		return config.GetInt64(cc.maxLimitConfigKey), nil
	case "size":
		return config.GetByteSize(cc.maxLimitConfigKey), nil
	default:
		return 0, i18n.NewError(cc.ctx, coremsgs.MsgCacheUnexpectedSizeKeyNameInternal, sizeConfigName)
	}
}

func (cc *CConfig) TTL() time.Duration {
	return config.GetDuration(cc.ttlConfigKey)
}

type Manager interface {
	GetCache(cc *CConfig) (CInterface, error)
	ListKeys() []string
}

type CInterface interface {
	Get(key string) interface{}
	Set(key string, val interface{})

	GetString(key string) string
	SetString(key string, val string)

	GetInt(key string) int
	SetInt(key string, val int)
}

type CCache struct {
	enabled  bool
	ctx      context.Context
	name     string
	cache    *ccache.Cache
	cacheTTL time.Duration
}

func (c *CCache) Set(key string, val interface{}) {
	if !c.enabled {
		return
	}
	c.cache.Set(c.name+":"+key, val, c.cacheTTL)
}
func (c *CCache) Get(key string) interface{} {
	if !c.enabled {
		return nil
	}
	if cached := c.cache.Get(c.name + ":" + key); cached != nil {
		cached.Extend(c.cacheTTL)
		return cached.Value()
	}
	return nil
}

func (c *CCache) SetString(key string, val string) {
	c.Set(key, val)
}

func (c *CCache) GetString(key string) string {
	val := c.Get(key)
	if val != nil {
		return c.Get(key).(string)
	}
	return ""
}

func (c *CCache) SetInt(key string, val int) {
	c.Set(key, val)
}

func (c *CCache) GetInt(key string) int {
	val := c.Get(key)
	if val != nil {
		return c.Get(key).(int)
	}
	return 0
}

type cacheManager struct {
	ctx     context.Context
	enabled bool
	m       sync.Mutex
	// maintain a list of named configured CCache, the name are unique configuration category id
	// e.g. cache.batch
	configuredCaches map[string]CInterface
}

func (cm *cacheManager) GetCache(cc *CConfig) (CInterface, error) {
	cacheName, err := cc.UniqueName()
	if err != nil {
		return nil, err
	}
	maxSize, err := cc.MaxSize()
	if err != nil {
		return nil, err
	}
	cm.m.Lock()
	cache, exists := cm.configuredCaches[cacheName]
	if !exists {
		cache = &CCache{
			ctx:      cc.ctx,
			name:     cacheName,
			cache:    ccache.New(ccache.Configure().MaxSize(maxSize)),
			cacheTTL: cc.TTL(),
			enabled:  cm.enabled,
		}
		cm.configuredCaches[cacheName] = cache
	}
	cm.m.Unlock()
	return cache, nil
}

func (cm *cacheManager) ListKeys() []string {
	keys := make([]string, 0, len(cm.configuredCaches))
	for k := range cm.configuredCaches {
		keys = append(keys, k)
	}
	return keys
}

func NewCacheManager(ctx context.Context) Manager {
	cm := &cacheManager{
		ctx:              ctx,
		enabled:          config.GetBool(coreconfig.CacheEnabled),
		configuredCaches: map[string]CInterface{},
	}
	return cm
}

// should only be used for testing purpose
func NewUmanagedCache(ctx context.Context, sizeLimit int64, ttl time.Duration) CInterface {
	return &CCache{
		ctx:      ctx,
		name:     "cache.unmanaged",
		cache:    ccache.New(ccache.Configure().MaxSize(sizeLimit)),
		cacheTTL: ttl,
		enabled:  true,
	}
}
