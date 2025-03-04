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

var BurnSubmittedCounter *prometheus.CounterVec
var BurnConfirmedCounter *prometheus.CounterVec
var BurnRejectedCounter *prometheus.CounterVec
var BurnHistogram *prometheus.HistogramVec

// BurnSubmittedCounterName is the prometheus metric for tracking the total number of burns submitted
var BurnSubmittedCounterName = "ff_burn_submitted_total"

// BurnConfirmedCounterName is the prometheus metric for tracking the total number of burns confirmed
var BurnConfirmedCounterName = "ff_burn_confirmed_total"

// BurnRejectedCounterName is the prometheus metric for tracking the total number of burns rejected
var BurnRejectedCounterName = "ff_burn_rejected_total"

// BurnHistogramName is the prometheus metric for tracking the total number of burns - histogram
var BurnHistogramName = "ff_burn_histogram"

func InitTokenBurnMetrics() {
	BurnSubmittedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BurnSubmittedCounterName,
		Help: "Number of submitted burns",
	}, namespaceLabels)
	BurnConfirmedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BurnConfirmedCounterName,
		Help: "Number of confirmed burns",
	}, namespaceLabels)
	BurnRejectedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BurnRejectedCounterName,
		Help: "Number of rejected burns",
	}, namespaceLabels)
	BurnHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: BurnHistogramName,
		Help: "Histogram of burns, bucketed by time to finished",
	}, namespaceLabels)
}

func RegisterTokenBurnMetrics() {
	registry.MustRegister(BurnSubmittedCounter)
	registry.MustRegister(BurnConfirmedCounter)
	registry.MustRegister(BurnRejectedCounter)
	registry.MustRegister(BurnHistogram)
}
