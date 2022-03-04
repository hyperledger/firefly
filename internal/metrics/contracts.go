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

var BlockchainTransactionsCounter *prometheus.CounterVec
var BlockchainQueriesCounter *prometheus.CounterVec
var BlockchainEventsCounter *prometheus.CounterVec

// BlockchainTransactionsCounterName is the prometheus metric for tracking the total number of blockchain transactions
var BlockchainTransactionsCounterName = "ff_blockchain_transactions_total"

// BlockchainQueriesCounterName is the prometheus metric for tracking the total number of blockchain queries
var BlockchainQueriesCounterName = "ff_blockchain_queries_total"

// BlockchainEventsCounterName is the prometheus metric for tracking the total number of blockchain events
var BlockchainEventsCounterName = "ff_blockchain_events_total"

var LocationLabelName = "location"
var MethodNameLabelName = "methodName"
var SignatureLabelName = "signature"

func InitBlockchainMetrics() {
	BlockchainTransactionsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BlockchainTransactionsCounterName,
		Help: "Number of blockchain transactions",
	}, []string{LocationLabelName, MethodNameLabelName})
	BlockchainQueriesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BlockchainQueriesCounterName,
		Help: "Number of blockchain queries",
	}, []string{LocationLabelName, MethodNameLabelName})
	BlockchainEventsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BlockchainEventsCounterName,
		Help: "Number of blockchain events",
	}, []string{LocationLabelName, SignatureLabelName})
}

func RegisterBlockchainMetrics() {
	registry.MustRegister(BlockchainTransactionsCounter)
	registry.MustRegister(BlockchainQueriesCounter)
	registry.MustRegister(BlockchainEventsCounter)
}
