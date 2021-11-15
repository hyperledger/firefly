// Copyright © 2021 Kaleido, Inc.
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

var registry *prometheus.Registry

// Registry returns FireFly's customized Prometheus registry
func Registry() *prometheus.Registry {
	if registry == nil {
		registry = prometheus.NewRegistry()
		registry.MustRegister(prometheus.NewGoCollector())
		registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	return registry
}

// Clear will reset the Prometheus metrics registry, useful for testing
func Clear() {
	registry = nil
}
