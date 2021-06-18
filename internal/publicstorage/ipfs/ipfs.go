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

package ipfs

import (
	"context"
	"encoding/json"
	"fmt"

	"io"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/pkg/publicstorage"
)

type IPFS struct {
	ctx          context.Context
	capabilities *publicstorage.Capabilities
	callbacks    publicstorage.Callbacks
	apiClient    *resty.Client
	gwClient     *resty.Client
}

type ipfsUploadResponse struct {
	Name string      `json:"Name"`
	Hash string      `json:"Hash"`
	Size json.Number `json:"Size"`
}

func (i *IPFS) Name() string {
	return "ipfs"
}

func (i *IPFS) Init(ctx context.Context, prefix config.Prefix, callbacks publicstorage.Callbacks) error {

	i.ctx = log.WithLogField(ctx, "publicstorage", "ipfs")
	i.callbacks = callbacks

	apiPrefix := prefix.SubPrefix(IPFSConfAPISubconf)
	if apiPrefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, apiPrefix.Resolve(restclient.HTTPConfigURL), "ipfs")
	}
	i.apiClient = restclient.New(i.ctx, apiPrefix)
	gwPrefix := prefix.SubPrefix(IPFSConfGatewaySubconf)
	if gwPrefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, gwPrefix.Resolve(restclient.HTTPConfigURL), "ipfs")
	}
	i.gwClient = restclient.New(i.ctx, gwPrefix)
	i.capabilities = &publicstorage.Capabilities{}
	return nil
}

func (i *IPFS) Capabilities() *publicstorage.Capabilities {
	return i.capabilities
}

func (i *IPFS) PublishData(ctx context.Context, data io.Reader) (payloadRef string, err error) {
	var ipfsResponse ipfsUploadResponse
	res, err := i.apiClient.R().
		SetContext(ctx).
		SetFileReader("document", "file.bin", data).
		SetResult(&ipfsResponse).
		Post("/api/v0/add")
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(i.ctx, res, err, i18n.MsgIPFSRESTErr)
	}
	log.L(ctx).Infof("IPFS published %s Size=%s", ipfsResponse.Hash, ipfsResponse.Size)
	return payloadRef, err
}

func (i *IPFS) RetrieveData(ctx context.Context, payloadRef string) (data io.ReadCloser, err error) {
	res, err := i.gwClient.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(fmt.Sprintf("/ipfs/%s", payloadRef))
	restclient.OnAfterResponse(i.gwClient, res) // required using SetDoNotParseResponse
	if err != nil || !res.IsSuccess() {
		if res != nil && res.RawBody() != nil {
			_ = res.RawBody().Close()
		}
		return nil, restclient.WrapRestErr(i.ctx, res, err, i18n.MsgIPFSRESTErr)
	}
	log.L(ctx).Infof("IPFS retrieved %s", payloadRef)
	return res.RawBody(), nil
}
