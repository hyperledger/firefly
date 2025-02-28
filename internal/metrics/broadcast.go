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
	"github.com/prometheus/client_golang/prometheus"
)

var BroadcastSubmittedCounter *prometheus.CounterVec
var BroadcastConfirmedCounter *prometheus.CounterVec
var BroadcastRejectedCounter *prometheus.CounterVec
var BroadcastHistogram *prometheus.HistogramVec

// BroadcastSubmittedCounterName is the prometheus metric for tracking the total number of broadcasts submitted
var BroadcastSubmittedCounterName = "ff_broadcast_submitted_total"

// BroadcastConfirmedCounterName is the prometheus metric for tracking the total number of broadcasts confirmed
var BroadcastConfirmedCounterName = "ff_broadcast_confirmed_total"

// BroadcastRejectedCounterName is the prometheus metric for tracking the total number of broadcasts rejected
var BroadcastRejectedCounterName = "ff_broadcast_rejected_total"

// BroadcastHistogramName is the prometheus metric for tracking the total number of broadcast messages - histogram
var BroadcastHistogramName = "ff_broadcast_histogram"

func InitBroadcastMetrics() {
	BroadcastSubmittedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BroadcastSubmittedCounterName,
		Help: "Number of submitted broadcasts",
	}, namespaceLabels)
	BroadcastConfirmedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BroadcastConfirmedCounterName,
		Help: "Number of confirmed broadcasts",
	}, namespaceLabels)
	BroadcastRejectedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BroadcastRejectedCounterName,
		Help: "Number of rejected broadcasts",
	}, namespaceLabels)
	BroadcastHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: BroadcastHistogramName,
		Help: "Histogram of broadcasts, bucketed by time to finished",
	}, namespaceLabels)
}

func RegisterBroadcastMetrics() {
	registry.MustRegister(BroadcastSubmittedCounter)
	registry.MustRegister(BroadcastConfirmedCounter)
	registry.MustRegister(BroadcastRejectedCounter)
	registry.MustRegister(BroadcastHistogram)
}
