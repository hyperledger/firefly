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

package ipfs

import (
	"context"
	"encoding/json"
	"fmt"

	"io"

	"github.com/akamensky/base58"
	"github.com/go-resty/resty/v2"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/p2pfs"
)

type IPFS struct {
	ctx          context.Context
	capabilities *p2pfs.Capabilities
	events       p2pfs.Events
	client       *resty.Client
}

type ipfsUploadResponse struct {
	Name string      `json:"Name"`
	Hash string      `json:"Hash"`
	Size json.Number `json:"Size"`
}

func (i *IPFS) Init(ctx context.Context, prefix config.ConfigPrefix, events p2pfs.Events) error {

	i.ctx = log.WithLogField(ctx, "p2pfs", "ipfs")
	i.events = events
	if prefix.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "p2pfs")
	}
	i.client = ffresty.New(i.ctx, prefix)
	i.capabilities = &p2pfs.Capabilities{}
	return nil
}

func (i *IPFS) Capabilities() *p2pfs.Capabilities {
	return i.capabilities
}

func (i *IPFS) ipfsHashToBytes32(ipfshash string) (*fftypes.Bytes32, error) {
	b, err := base58.Decode(ipfshash)
	if err != nil {
		return nil, i18n.WrapError(i.ctx, err, i18n.MsgIPFSHashDecodeFailed, ipfshash)
	}
	if len(b) != 34 {
		return nil, i18n.NewError(i.ctx, i18n.MsgIPFSHashDecodeFailed, b)
	}
	var b32 fftypes.Bytes32
	copy(b32[:], b[2:34])
	return &b32, nil
}

func (i *IPFS) bytes32ToIPFSHash(payloadRef *fftypes.Bytes32) string {
	var hashBytes [34]byte
	copy(hashBytes[0:2], []byte{0x12, 0x20})
	copy(hashBytes[2:34], payloadRef[0:32])
	return base58.Encode(hashBytes[:])
}

func (i *IPFS) PublishData(ctx context.Context, data io.Reader) (payloadRef *fftypes.Bytes32, err error) {
	var ipfsResponse ipfsUploadResponse
	res, err := i.client.R().
		SetContext(ctx).
		SetFileReader("document", "file.bin", data).
		SetResult(&ipfsResponse).
		Post("/api/v0/add")
	if err != nil || !res.IsSuccess() {
		return nil, ffresty.WrapRestErr(i.ctx, res, err, i18n.MsgIPFSRESTErr)
	}
	log.L(ctx).Infof("IPFS published %s Size=%s", ipfsResponse.Hash, ipfsResponse.Size)
	return i.ipfsHashToBytes32(ipfsResponse.Hash)
}

func (i *IPFS) RetrieveData(ctx context.Context, payloadRef *fftypes.Bytes32) (data io.ReadCloser, err error) {
	ipfsHash := i.bytes32ToIPFSHash(payloadRef)
	res, err := i.client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(fmt.Sprintf("/ipfs/%s", ipfsHash))
	if err != nil || !res.IsSuccess() {
		if res != nil && res.RawBody() != nil {
			_ = res.RawBody().Close()
		}
		return nil, ffresty.WrapRestErr(i.ctx, res, err, i18n.MsgIPFSRESTErr)
	}
	log.L(ctx).Infof("IPFS retrieved %s", ipfsHash)
	return res.RawBody(), nil
}
