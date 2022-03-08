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

package fftypes

const (
	// ChartHistogramMaxBuckets max buckets that can be requested
	ChartHistogramMaxBuckets = 100
	// ChartHistogramMinBuckets min buckets that can be requested
	ChartHistogramMinBuckets = 1
)

// ChartHistogram is a list of buckets with types
type ChartHistogram struct {
	// Count for entire timestamp in histogram
	Count string `json:"count"`
	// Timestamp of bucket
	Timestamp *FFTime `json:"timestamp"`
	// Types list of histogram types and their count
	Types []*ChartHistogramType `json:"types"`
}

// ChartHistogramType is a type and count
type ChartHistogramType struct {
	// Count for type in histogram bucket
	Count string `json:"count"`
	// Type of bucket in histogram
	Type string `json:"type"`
}

// ChartHistogramInterval specifies lower and upper timestamps for histogram bucket
type ChartHistogramInterval struct {
	// StartTime start time of histogram interval
	StartTime *FFTime `json:"startTime"`
	// EndTime end time of histogram interval
	EndTime *FFTime `json:"endTime"`
}
