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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func utAddresResolverConfigPrefix() config.Prefix {
	config.Reset()
	prefix := config.NewPluginConfig("utaddressresovler")
	(&Ethereum{}).InitPrefix(prefix)
	return prefix.SubPrefix(AddressResolverConfigKey)
}

func TestAddressResolverInEthereumOK(t *testing.T) {

	addr := "0xf1A9dB812D6710040185e9d981A0AB25003878ce"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/resolve/testkeystring", r.URL.Path)
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf(`{"address":"%s"}`, addr)))
	}))
	defer server.Close()

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve/{{.Key}}", server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	e := &Ethereum{
		ctx:             ctx,
		addressResolver: ar,
	}

	resolved, err := e.ResolveSigningKey(ctx, "testkeystring")
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

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverRetainOriginal, true)
	prefix.Set(AddressResolverMethod, "POST")
	prefix.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve", server.URL))
	prefix.Set(AddressResolverBodyTemplate, `{"key":"{{.Key}}"}`)
	prefix.Set(AddressResolverResponseField, "Addr")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	resolved, err := ar.ResolveSigningKey(ctx, "testkeystring")
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

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverMethod, "POST")
	prefix.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve", server.URL))
	prefix.Set(AddressResolverBodyTemplate, `{"key":"{{.Key}}"}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	_, err = ar.ResolveSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10335", err)

}

func TestAddressResolverPOSTResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(204)
	}))
	defer server.Close()

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverMethod, "POST")
	prefix.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve", server.URL))
	prefix.Set(AddressResolverBodyTemplate, `{"key":"{{.Key}}"}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	_, err = ar.ResolveSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10335", err)

}

func TestAddressResolverFailureResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(500)
	}))
	defer server.Close()

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve/{{.Key}}", server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	_, err = ar.ResolveSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10334", err)

}

func TestAddressResolverErrorResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(500)
	}))
	server.Close() // close immediately

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverURLTemplate, fmt.Sprintf("%s/resolve/{{.Key}}", server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	_, err = ar.ResolveSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10333", err)

}

func TestAddressResolverBadBodyTemplate(t *testing.T) {

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverURLTemplate, "http://ff.example/resolve")
	prefix.Set(AddressResolverBodyTemplate, `{{unclosed!}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := newAddressResolver(ctx, prefix)
	assert.Regexp(t, "FF10331.*bodyTemplate", err)

}

func TestAddressResolverErrorURLTemplate(t *testing.T) {

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverURLTemplate, "http://ff.example/resolve/{{.Wrong}}")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	_, err = ar.ResolveSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10332.*urlTemplate", err)

}

func TestAddressResolverErrorBodyTemplate(t *testing.T) {

	prefix := utAddresResolverConfigPrefix()
	prefix.Set(AddressResolverURLTemplate, "http://ff.example/resolve")
	prefix.Set(AddressResolverBodyTemplate, "{{.Wrong}}")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar, err := newAddressResolver(ctx, prefix)
	assert.NoError(t, err)

	_, err = ar.ResolveSigningKey(ctx, "testkeystring")
	assert.Regexp(t, "FF10332.*bodyTemplate", err)

}
