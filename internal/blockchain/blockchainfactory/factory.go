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

package blockchainfactory

import (
	"context"

	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/blockchain/ethereum"
	"github.com/kaleido-io/firefly/internal/i18n"
)

func GetBlockchainPlugin(ctx context.Context, pluginType string) (blockchain.Plugin, error) {
	switch pluginType {
	case "ethereum":
		return &ethereum.Ethereum{}, nil
	default:
		return nil, i18n.NewError(ctx, i18n.MsgUnknownBlockchainPlugin, pluginType)
	}
}
