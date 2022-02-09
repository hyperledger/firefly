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
	"github.com/prometheus/client_golang/prometheus/collectors"
	muxprom "gitlab.com/hfuss/mux-prometheus/pkg/middleware"
)

var registry *prometheus.Registry
var adminInstrumentation *muxprom.Instrumentation
var restInstrumentation *muxprom.Instrumentation

// Registry returns FireFly's customized Prometheus registry
func Registry() *prometheus.Registry {
	if registry == nil {
		initMetricsCollectors()
		registry = prometheus.NewRegistry()
		registerMetricsCollectors()
	}

	return registry
}

// GetAdminServerInstrumentation returns the admin server's Prometheus middleware, ensuring its metrics are never
// registered twice
func GetAdminServerInstrumentation() *muxprom.Instrumentation {
	if adminInstrumentation == nil {
		adminInstrumentation = NewInstrumentation("admin")
	}
	return adminInstrumentation
}

// GetRestServerInstrumentation returns the REST server's Prometheus middleware, ensuring its metrics are never
// registered twice
func GetRestServerInstrumentation() *muxprom.Instrumentation {
	if restInstrumentation == nil {
		restInstrumentation = NewInstrumentation("rest")
	}
	return restInstrumentation
}

func NewInstrumentation(subsystem string) *muxprom.Instrumentation {
	return muxprom.NewCustomInstrumentation(
		true,
		"ff_apiserver",
		subsystem,
		prometheus.DefBuckets,
		map[string]string{},
		Registry(),
	)
}

// Clear will reset the Prometheus metrics registry and instrumentations, useful for testing
func Clear() {
	registry = nil
	adminInstrumentation = nil
	restInstrumentation = nil
}

func initMetricsCollectors() {
	InitBroadcastMetrics()
	InitPrivateMsgMetrics()
	InitTokenMintMetrics()
	InitTokenTransferMetrics()
	InitTokenBurnMetrics()
	InitBatchPinMetrics()
}

func registerMetricsCollectors() {
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	RegisterBatchPinMetrics()
	RegisterBroadcastMetrics()
	RegisterPrivateMsgMetrics()
	RegisterTokenMintMetrics()
	RegisterTokenTransferMetrics()
	RegisterTokenBurnMetrics()
}
