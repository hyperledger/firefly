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

package ethereum

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func utAddresResolverConfig() config.Section {
	coreconfig.Reset()
	config := config.RootSection("utaddressresovler")
	(&Ethereum{}).InitConfig(config)
	return config.SubSection(AddressResolverConfigKey)
}

func TestCacheInitFail(t *testing.T) {
	cacheInitError := errors.New("Initialization error.")
	ctx := context.Background()
	config := utAddresResolverConfig()

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(nil, cacheInitError)
	_, err := newAddressResolver(ctx, config, cmi)
	assert.Equal(t, cacheInitError, err)
}

func TestAddressResolverInEthereumOKCached(t *testing.T) {

	count := 0
	addr := "0xf1A9dB812D6710040185e9d981A0AB25003878ce"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/resolve/testkeystring", r.URL.Path)
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf(`{"address":"%s"}`, addr)))
		assert.Zero(t, count)
		count++
	}))
	defer server.Close()

	config := utAddresResolverConfig()
	config.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve/{{.Key}}", server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	defer cancel()

	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	e := &Ethereum{
		ctx:             ctx,
		addressResolver: ar,
	}

	resolved, err := e.NormalizeSigningKey(ctx, "testkeystring")
	assert.NoError(t, err)
	assert.Equal(t, strings.ToLower(addr), resolved)

	resolved, err = e.NormalizeSigningKey(ctx, "testkeystring") // cached
	assert.NoError(t, err)
	assert.Equal(t, strings.ToLower(addr), resolved)
}

func TestAddressResolverPOSTOk(t *testing.T) {

	addr := "0x256e288EDF9392B9236F698a64365F216A4Eff97"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var jo fftypes.JSONObject
		json.NewDecoder(r.Body).Decode(&jo)
		assert.Equal(t, "testkeystring", jo.GetString("key"))
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf(`{"Addr":"%s"}`, addr)))
	}))
	defer server.Close()

	config := utAddresResolverConfig()
	config.Set(AddressResolverRetainOriginal, true)
	config.Set(AddressResolverMethod, "POST")
	config.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve", server.URL))
	config.Set(AddressResolverBodyTemplate, `{"key":"{{.Key}}"}`)
	config.Set(AddressResolverResponseField, "Addr")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	resolved, err := ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.NoError(t, err)

	assert.Equal(t, strings.ToLower(addr), resolved)

}

func TestAddressResolverPOSTBadKey(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(200)
		rw.Write([]byte(`{"address":"badness"}`))
	}))
	defer server.Close()

	config := utAddresResolverConfig()
	config.Set(AddressResolverMethod, "POST")
	config.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve", server.URL))
	config.Set(AddressResolverBodyTemplate, `{"key":"{{.Key}}"}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	_, err = ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10341", err)

}

func TestAddressResolverPOSTResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(204)
	}))
	defer server.Close()

	config := utAddresResolverConfig()
	config.Set(AddressResolverMethod, "POST")
	config.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve", server.URL))
	config.Set(AddressResolverBodyTemplate, `{"key":"{{.Key}}"}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	_, err = ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10341", err)

}

func TestAddressResolverFailureResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(500)
	}))
	defer server.Close()

	config := utAddresResolverConfig()
	config.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve/{{.Key}}", server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	_, err = ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10340", err)

}

func TestAddressResolverErrorResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(500)
	}))
	server.Close() // close immediately

	config := utAddresResolverConfig()
	config.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve/{{.Key}}", server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	_, err = ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10339", err)

}

func TestAddressResolverBadBodyTemplate(t *testing.T) {

	config := utAddresResolverConfig()
	config.Set(AddressResolverURLTemplate, "http://ff.example/resolve")
	config.Set(AddressResolverBodyTemplate, `{{unclosed!}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	_, err := newAddressResolver(ctx, config, cmi)
	assert.Regexp(t, "FF10337.*bodyTemplate", err)

}

func TestAddressResolverErrorURLTemplate(t *testing.T) {

	config := utAddresResolverConfig()
	config.Set(AddressResolverURLTemplate, "http://ff.example/resolve/{{.Wrong}}")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	assert.NoError(t, err)

	_, err = ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10338.*urlTemplate", err)

}

func TestAddressResolverErrorBodyTemplate(t *testing.T) {

	config := utAddresResolverConfig()
	config.Set(AddressResolverURLTemplate, "http://ff.example/resolve")
	config.Set(AddressResolverBodyTemplate, "{{.Wrong}}")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ar, err := newAddressResolver(ctx, config, cmi)
	cmi.AssertCalled(t, "GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheAddressResolverLimit,
		coreconfig.CacheAddressResolverTTL,
		"",
	))
	assert.NoError(t, err)

	_, err = ar.NormalizeSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10338.*bodyTemplate", err)

}
