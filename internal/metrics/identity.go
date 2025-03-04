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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var NodeIdentityDXCertMismatchGauge *prometheus.GaugeVec
var NodeIdentityDXCertExpiryGauge *prometheus.GaugeVec

const (
	NodeIdentityDXCertMismatch = "ff_multiparty_node_identity_dx_mismatch"
	NodeIdentityDXCertExpiry   = "ff_multiparty_node_identity_dx_expiry_epoch"
)

func InitIdentityMetrics() {
	NodeIdentityDXCertMismatchGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: NodeIdentityDXCertMismatch,
		Help: "Status of node identity DX cert mismatch",
	}, namespaceLabels)

	NodeIdentityDXCertExpiryGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: NodeIdentityDXCertExpiry,
		Help: "Timestamp, in Unix epoch format, of node identity DX cert's expiry",
	}, namespaceLabels)
}

func RegisterIdentityMetrics() {
	registry.MustRegister(NodeIdentityDXCertMismatchGauge)
	registry.MustRegister(NodeIdentityDXCertExpiryGauge)
}

// TODO should this type live elsewhere ??
type NodeIdentityDXCertMismatchStatus string

const (
	NodeIdentityDXCertMismatchStatusMismatched NodeIdentityDXCertMismatchStatus = "mismatched"
	NodeIdentityDXCertMismatchStatusHealthy    NodeIdentityDXCertMismatchStatus = "healthy"
	NodeIdentityDXCertMismatchStatusUnknown    NodeIdentityDXCertMismatchStatus = "unknown"
)

func (mm *metricsManager) NodeIdentityDXCertMismatch(namespace string, state NodeIdentityDXCertMismatchStatus) {
	var gaugeState float64
	switch state {
	case NodeIdentityDXCertMismatchStatusMismatched:
		gaugeState = 1.0
	case NodeIdentityDXCertMismatchStatusHealthy:
		gaugeState = 0.0
	case NodeIdentityDXCertMismatchStatusUnknown:
		fallthrough
	default:
		gaugeState = -1.0
	}

	NodeIdentityDXCertMismatchGauge.WithLabelValues(namespace).Set(gaugeState)
}

func (mm *metricsManager) NodeIdentityDXCertExpiry(namespace string, expiry time.Time) {
	NodeIdentityDXCertExpiryGauge.WithLabelValues(namespace).Set(float64(expiry.UTC().Unix()))
}
