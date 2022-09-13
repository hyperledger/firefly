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
	"context"
	"strings"
	"text/template"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
)

// addressResolver is a REST-pluggable interface to allow arbitrary strings that reference
// keys, to be resolved down to an Ethereum address - which will be kept in a LRU cache.
// This supports cases where the signing device behind Ethconnect is able to support keys
// addressed using somthing like a HD Wallet hierarchical syntax.
// Once the resolver has returned the String->Address mapping, the ethconnect downstream
// signing process must be able to process using the resolved ethereum address (meaning
// it might have to reliably store the reverse mapping, it the case of a HD wallet).
type addressResolver struct {
	retainOriginal bool
	method         string
	urlTemplate    *template.Template
	bodyTemplate   *template.Template
	responseField  string
	client         *resty.Client
	cache          cache.CInterface
}

type addressResolverInserts struct {
	Key string
}

func newAddressResolver(ctx context.Context, localConfig config.Section, cacheManager cache.Manager) (ar *addressResolver, err error) {
	cache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheAddressResolverLimit,
			coreconfig.CacheAddressResolverTTL,
			"",
		),
	)
	if err != nil {
		return nil, err
	}
	ar = &addressResolver{
		retainOriginal: localConfig.GetBool(AddressResolverRetainOriginal),
		method:         localConfig.GetString(AddressResolverMethod),
		responseField:  localConfig.GetString(AddressResolverResponseField),
		client:         ffresty.New(ctx, localConfig),
		cache:          cache,
	}

	urlTemplateString := localConfig.GetString(AddressResolverURLTemplate)
	ar.urlTemplate, err = template.New(AddressResolverURLTemplate).Option("missingkey=error").Parse(urlTemplateString)
	if err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgGoTemplateCompileFailed, AddressResolverURLTemplate, err)
	}

	bodyTemplateString := localConfig.GetString(AddressResolverBodyTemplate)
	if bodyTemplateString != "" {
		ar.bodyTemplate, err = template.New(AddressResolverBodyTemplate).Option("missingkey=error").Parse(bodyTemplateString)
		if err != nil {
			return nil, i18n.NewError(ctx, coremsgs.MsgGoTemplateCompileFailed, AddressResolverBodyTemplate, err)
		}
	}

	return ar, nil
}

func (ar *addressResolver) NormalizeSigningKey(ctx context.Context, keyDescriptor string) (string, error) {

	if cached := ar.cache.GetString(keyDescriptor); cached != "" {
		return cached, nil
	}

	inserts := &addressResolverInserts{
		Key: keyDescriptor,
	}

	urlStr := &strings.Builder{}
	err := ar.urlTemplate.Execute(urlStr, inserts)
	if err != nil {
		return "", i18n.NewError(ctx, coremsgs.MsgGoTemplateExecuteFailed, AddressResolverURLTemplate, err)
	}

	bodyStr := &strings.Builder{}
	if ar.bodyTemplate != nil {
		err := ar.bodyTemplate.Execute(bodyStr, inserts)
		if err != nil {
			return "", i18n.NewError(ctx, coremsgs.MsgGoTemplateExecuteFailed, AddressResolverBodyTemplate, err)
		}
	}

	var jsonRes fftypes.JSONObject
	res, err := ar.client.NewRequest().
		SetContext(ctx).
		SetBody(bodyStr.String()).
		SetResult(&jsonRes).
		Execute(ar.method, urlStr.String())
	if err != nil {
		return "", i18n.NewError(ctx, coremsgs.MsgAddressResolveFailed, keyDescriptor, err)
	}
	if res.IsError() {
		return "", i18n.NewError(ctx, coremsgs.MsgAddressResolveBadStatus, keyDescriptor, res.StatusCode(), jsonRes.String())
	}

	address, err := validateEthAddress(ctx, jsonRes.GetString(ar.responseField))
	if err != nil {
		return "", i18n.NewError(ctx, coremsgs.MsgAddressResolveBadResData, keyDescriptor, jsonRes.String(), err)
	}

	ar.cache.SetString(keyDescriptor, address)
	return address, nil
}
