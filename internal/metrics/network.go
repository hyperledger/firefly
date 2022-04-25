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

package metrics

import "github.com/prometheus/client_golang/prometheus"

var NetworkIdentitiesGauge *prometheus.GaugeVec

// NetworkIdentitiesGaugeName is the prometheus metric for observing the total identities registered
var NetworkIdentitiesGaugeName = "ff_network_identities"

func InitNetworkMetrics() {
	NetworkIdentitiesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: NetworkIdentitiesGaugeName,
		Help: "Number of nodes registered",
	}, []string{"type"})
}

func RegisterNetworkMetrics() {
	registry.MustRegister(NetworkIdentitiesGauge)
}
