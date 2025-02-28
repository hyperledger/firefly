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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/multiparty"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/multipartymocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
)

const (
	testCertBundle = `
-----BEGIN CERTIFICATE-----
MIIDqTCCApGgAwIBAgIUbZT+Ds4f2oDmGpgVi+SaQq9gxvcwDQYJKoZIhvcNAQEL
BQAwZDELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExFjAUBgNVBAcM
DVNhbiBGcmFuY2lzY28xEzARBgNVBAoMCkV4YW1wbGUgQ0ExEzARBgNVBAMMCmV4
YW1wbGUtY2EwHhcNMjUwMjI4MTkzMDM4WhcNMzUwMjI2MTkzMDM4WjBkMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5j
aXNjbzETMBEGA1UECgwKRXhhbXBsZSBDQTETMBEGA1UEAwwKZXhhbXBsZS1jYTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAOWBryFqk0YqQ6pzGJvBDjbV
4BnkMzsv+Fq869Xks09OP4eW44oqfFUmpCFyS3fEmRCz+389t4mKvxcCRIJMW0f5
K9jffG1QKUKL4UuNfEPFpM0MXTwhI+dCdvofdelzc+KBGA6CDYlnWYcCKFSuWeSu
xrb/qCEvhcCaSYt3e2WcRHRuK+OLzM3REeJctC4G/pq858OUV5CZU2B6aGV/9uFL
ZW3TCrOaj+Khzzt5FNvjVdLiUw0FS8VESxFA4kH8p+XUshs9S0e7LfIBSID2NU8+
+5D6HliqNqikbsny1Ps6GhLa+nI37LOVj7nFcG7uk+gb6HUN1+0YvjOJ0/zvnLEC
AwEAAaNTMFEwHQYDVR0OBBYEFJfNoXmIn5S6W7Lcj5G/huW5q1YQMB8GA1UdIwQY
MBaAFJfNoXmIn5S6W7Lcj5G/huW5q1YQMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZI
hvcNAQELBQADggEBALsdRYJHQMkhjLcrO4Yha1KXh2d+irmi8AqQqQgbLIsSzuqG
bKFiYnJ8PKHaISHlev2xRM9kEjDZ/9q8T4aUELg4eBjj7VK+gs+gSBO6peJ+AcEg
TepsE5GHmhoIIiE/3dIP6XnaM6NBb8q0ewsIg1c5vLlrt8W96LY6Og7f+742VvoV
H31srpGjy7c5nYjBTn/Bu84eb5Lxfvy10sJjnenkXDJvzkUcnfbRzDQ9k5ZuPa05
x+BsxonN0iaeZH91F+Y3kgJidLnU5EhIB/1KXYjuEbl9qUxD6GFHRststPRPeOmj
7C+BtJCIjjavysSqVMvQWLQ6rXms3SpRPAimWqM=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDqjCCApKgAwIBAgIUWnobAQ4vq8gWBAXZBf7XZG3oSiUwDQYJKoZIhvcNAQEL
BQAwZDELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExFjAUBgNVBAcM
DVNhbiBGcmFuY2lzY28xEzARBgNVBAoMCkV4YW1wbGUgQ0ExEzARBgNVBAMMCmV4
YW1wbGUtY2EwHhcNMjUwMjI4MTkzMDU3WhcNMjgwMjI4MTkzMDU3WjB2MQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5j
aXNjbzEcMBoGA1UECgwTSHlwZXJsZWRnZXIgRmlyZWZseTEcMBoGA1UEAwwTaHlw
ZXJsZWRnZXItZmlyZWZseTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AKlbQ+7cWWS0+QPp03PrdxsnAAtG2tWOk2CEG7HS3AlBU82YImhCidKOw+jQPS68
2f2d0tYBhugqB2Ki6HsfYMGTjHDLbUQ5y+cLk6PFbvhjm39Ayd+WGmhWht5qFtRN
gllTa/SbG8+iGaSPIVFCyvg1IzxsFnBGn+05Gu+KjpL4i0l1RqDmy5ItxKGP77in
RPEUkejiUozg/X3v2TWAGagIVF5+EQ2Cswot9W1faAvyu/QmSGLLfSH22GdEDHXa
U4DV5ArJ2U2eNkOuasSWGKBopa/Wh1SZjKrNsy5Gw84ihAI4k7ARoP+vu1dIPdaX
ElipmGMtUWu0Azn2l9QJZpMCAwEAAaNCMEAwHQYDVR0OBBYEFL798jEmX2+hw70t
SmfJA78PZnnHMB8GA1UdIwQYMBaAFJfNoXmIn5S6W7Lcj5G/huW5q1YQMA0GCSqG
SIb3DQEBCwUAA4IBAQBY1NXTuQJZvjip33dRXyWP6GsSDKbXTSCcSF38P4/m+pcH
r/q/upo+K+8eTtPqUwBsIywH5bypWqoIPtM+rkd3FVBe7uti2FExufpcOruzEGsY
rNTfiFZbc7eHmFRTkKXWW4j6b6ElygrBvV999BhCRNf6NS0/syjqsbALHkFGeIcl
78wdaR+m2XVJBV7SmPmZ/EQzxvhCZONNVyU5zvW2sehI7sRbZt9/FG5U1Ng0LarW
R0gnXX/IZFnLhLh6UpLOBB0KIGENh75EEU7755jMKDKFj16D0uA1Lzrh5YxicTMy
ydFYQLpLycsWl2oV3JB4pO5TIzjY9awkRE0MeMMc
-----END CERTIFICATE-----
`
)

func TestCheckNodeIdentityStatusSetsMismatchStateWhenCertsMatch(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	ctx := context.TODO()

	nm.multiparty.(*multipartymocks.Manager).On("LocalNode").Return(multiparty.LocalNode{
		Name: "local-node",
	})

	matchingProfile := fftypes.JSONAnyPtr(fmt.Sprintf(`{"cert": "%s" }`, strings.ReplaceAll(testCertBundle, "\n", `\n`))).JSONObject()

	nm.identity.(*identitymanagermocks.Manager).On("GetLocalNode", ctx).Return(&core.Identity{
		IdentityProfile: core.IdentityProfile{
			Profile: matchingProfile,
		},
	}, nil)

	expiry := time.Unix(1835379057, 0).UTC()
	nm.exchange.(*dataexchangemocks.Plugin).On("GetEndpointInfo", ctx, "local-node").Return(matchingProfile, nil)
	nm.metrics.(*metricsmocks.Manager).On("IsMetricsEnabled").Return(true)
	nm.metrics.(*metricsmocks.Manager).On("NodeIdentityDXCertMismatch", nm.namespace, metrics.NodeIdentityDXCertMismatchStatusHealthy).Return()
	nm.metrics.(*metricsmocks.Manager).On("NodeIdentityDXCertExpiry", nm.namespace, expiry).Return() // 2028-02-28 19:30:57 +0000 UTC

	err := nm.CheckNodeIdentityStatus(context.TODO())
	assert.NoError(t, err)
}

func TestCheckNodeIdentityStatusSetsMismatchStateWhenCertsDiffer(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	ctx := context.TODO()

	nm.multiparty.(*multipartymocks.Manager).On("LocalNode").Return(multiparty.LocalNode{
		Name: "local-node",
	})

	nodeProfile := fftypes.JSONAnyPtr(`{"cert": "a-different-cert" }`).JSONObject()
	dxPeer := fftypes.JSONAnyPtr(fmt.Sprintf(`{"cert": "%s" }`, strings.ReplaceAll(testCertBundle, "\n", `\n`))).JSONObject()

	nm.identity.(*identitymanagermocks.Manager).On("GetLocalNode", ctx).Return(&core.Identity{
		IdentityProfile: core.IdentityProfile{
			Profile: nodeProfile,
		},
	}, nil)

	expiry := time.Unix(1835379057, 0).UTC()
	nm.exchange.(*dataexchangemocks.Plugin).On("GetEndpointInfo", ctx, "local-node").Return(dxPeer, nil)
	nm.metrics.(*metricsmocks.Manager).On("IsMetricsEnabled").Return(true)
	nm.metrics.(*metricsmocks.Manager).On("NodeIdentityDXCertMismatch", nm.namespace, metrics.NodeIdentityDXCertMismatchStatusMismatched).Return()
	nm.metrics.(*metricsmocks.Manager).On("NodeIdentityDXCertExpiry", nm.namespace, expiry).Return() // 2028-02-28 19:30:57 +0000 UTC

	err := nm.CheckNodeIdentityStatus(context.TODO())
	assert.NoError(t, err)
}

func TestCheckNodeIdentityStatusSetsMismatchStateWhenEndpointInfoFails(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	ctx := context.TODO()

	nm.multiparty.(*multipartymocks.Manager).On("LocalNode").Return(multiparty.LocalNode{
		Name: "local-node",
	})

	nodeProfile := fftypes.JSONAnyPtr(`{"cert": "a-cert" }`).JSONObject()

	nm.identity.(*identitymanagermocks.Manager).On("GetLocalNode", ctx).Return(&core.Identity{
		IdentityProfile: core.IdentityProfile{
			Profile: nodeProfile,
		},
	}, nil)

	nm.exchange.(*dataexchangemocks.Plugin).On("GetEndpointInfo", ctx, "local-node").Return(nil, errors.New("endpoint info went pop"))
	nm.metrics.(*metricsmocks.Manager).On("IsMetricsEnabled").Return(true)
	nm.metrics.(*metricsmocks.Manager).On("NodeIdentityDXCertMismatch", nm.namespace, metrics.NodeIdentityDXCertMismatchStatusUnknown).Return()

	err := nm.CheckNodeIdentityStatus(context.TODO())
	assert.Error(t, err)
}
