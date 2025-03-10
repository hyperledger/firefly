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

var TransferSubmittedCounter *prometheus.CounterVec
var TransferConfirmedCounter *prometheus.CounterVec
var TransferRejectedCounter *prometheus.CounterVec
var TransferHistogram *prometheus.HistogramVec

// TransferSubmittedCounterName is the prometheus metric for tracking the total number of transfers submitted
var TransferSubmittedCounterName = "ff_transfer_submitted_total"

// TransferConfirmedCounterName is the prometheus metric for tracking the total number of transfers confirmed
var TransferConfirmedCounterName = "ff_transfer_confirmed_total"

// TransferRejectedCounterName is the prometheus metric for tracking the total number of transfers rejected
var TransferRejectedCounterName = "ff_transfer_rejected_total"

// TransferHistogramName is the prometheus metric for tracking the total number of transfers - histogram
var TransferHistogramName = "ff_transfer_histogram"

func InitTokenTransferMetrics() {
	TransferSubmittedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: TransferSubmittedCounterName,
		Help: "Number of submitted transfers",
	}, namespaceLabels)
	TransferConfirmedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: TransferConfirmedCounterName,
		Help: "Number of confirmed transfers",
	}, namespaceLabels)
	TransferRejectedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: TransferRejectedCounterName,
		Help: "Number of rejected transfers",
	}, namespaceLabels)
	TransferHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: TransferHistogramName,
		Help: "Histogram of transfers, bucketed by time to finished",
	}, namespaceLabels)
}

func RegisterTokenTransferMetrics() {
	registry.MustRegister(TransferSubmittedCounter)
	registry.MustRegister(TransferConfirmedCounter)
	registry.MustRegister(TransferRejectedCounter)
	registry.MustRegister(TransferHistogram)
}
