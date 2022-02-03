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

import (
	"github.com/prometheus/client_golang/prometheus"
)

var PrivateMsgSubmittedCounter prometheus.Counter
var PrivateMsgConfirmedCounter prometheus.Counter
var PrivateMsgRejectedCounter prometheus.Counter
var PrivateMsgHistogram prometheus.Histogram

// PrivateMsgSubmittedCounterName is the prometheus metric for tracking the total number of private messages submitted
var PrivateMsgSubmittedCounterName = "ff_private_msg_submitted_total"

// PrivateMsgConfirmedCounterName is the prometheus metric for tracking the total number of private messages confirmed
var PrivateMsgConfirmedCounterName = "ff_private_msg_confirmed_total"

// PrivateMsgRejectedCounterName is the prometheus metric for tracking the total number of private messages rejected
var PrivateMsgRejectedCounterName = "ff_private_msg_rejected_total"

// PrivateMsgHistogramName is the prometheus metric for tracking the total number of private messages - histogram
var PrivateMsgHistogramName = "ff_private_msg_histogram"

func InitPrivateMsgMetrics() {
	PrivateMsgSubmittedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: PrivateMsgSubmittedCounterName,
		Help: "Number of submitted private messages",
	})
	PrivateMsgConfirmedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: PrivateMsgConfirmedCounterName,
		Help: "Number of confirmed private messages",
	})
	PrivateMsgRejectedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: PrivateMsgRejectedCounterName,
		Help: "Number of rejected private messages",
	})
	PrivateMsgHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: PrivateMsgHistogramName,
		Help: "Histogram of private messages, bucketed by time to finished",
	})
}

func RegisterPrivateMsgMetrics() {
	registry.MustRegister(PrivateMsgSubmittedCounter)
	registry.MustRegister(PrivateMsgConfirmedCounter)
	registry.MustRegister(PrivateMsgRejectedCounter)
	registry.MustRegister(PrivateMsgHistogram)
}
