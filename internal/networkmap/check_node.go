// Copyright © 2025 Kaleido, Inc.
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

package networkmap

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
)

func (nm *networkMap) CheckNodeIdentityStatus(ctx context.Context) error {
	var mismatchState = metrics.NodeIdentityDXCertMismatchStatusUnknown
	defer func() {
		if nm.metrics.IsMetricsEnabled() {
			nm.metrics.NodeIdentityDXCertMismatch(nm.namespace, mismatchState)
		}
	}()

	localNodeName := nm.multiparty.LocalNode().Name
	if localNodeName == "" {
		return i18n.NewError(ctx, coremsgs.MsgNodeAndOrgIDMustBeSet)
	}

	node, err := nm.identity.GetLocalNode(ctx)
	if err != nil {
		return err
	}

	if node.Profile == nil {
		return i18n.NewError(ctx, coremsgs.MsgNoRegistrationMessageData) // TODO
	}

	dxProfile, err := nm.exchange.GetEndpointInfo(ctx, localNodeName)
	if err != nil {
		return err
	}

	nodeCert := node.Profile.GetString("cert")
	if nodeCert == "" {
		return i18n.NewError(ctx, coremsgs.MsgNoRegistrationMessageData) // TODO
	}

	dxCert := dxProfile.GetString("cert")
	if dxCert == "" {
		return i18n.NewError(ctx, coremsgs.MsgInvalidPluginConfiguration) // TODO
	}

	mismatchState = metrics.NodeIdentityDXCertMismatchStatusHealthy
	if nodeCert != dxCert {
		mismatchState = metrics.NodeIdentityDXCertMismatchStatusMismatched
	}

	if nm.metrics.IsMetricsEnabled() {
		expiry, err := extractSoonestExpiryFromCertBundle(dxCert)
		if err != nil {
			return i18n.WrapError(ctx, err, coremsgs.MsgInvalidPluginConfiguration) // TODO
		}

		if expiry.Before(time.Now()) {
			log.L(ctx).Warnf("Certificate for node '%s' has expired", localNodeName)
		}

		nm.metrics.NodeIdentityDXCertExpiry(nm.namespace, expiry)
	}

	return nil
}

// we assume the cert with the soonest expiry is the leaf cert, but even if its the CA,
// thats what will invalidate the leaf anyways, so really we only care about the soonest expiry
func extractSoonestExpiryFromCertBundle(certBundle string) (time.Time, error) {
	var leafCert *x509.Certificate
	var block *pem.Block
	var rest = []byte(certBundle)

	for {
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse certificate: %v", err)
		}
		if leafCert == nil || cert.NotAfter.Before(leafCert.NotAfter) {
			leafCert = cert
		}
	}

	if leafCert == nil {
		return time.Time{}, errors.New("no valid certificate found")
	}

	return leafCert.NotAfter.UTC(), nil
}
