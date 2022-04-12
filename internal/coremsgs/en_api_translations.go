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

package coremsgs

import "github.com/hyperledger/firefly/pkg/i18n"

var ffm = i18n.FFM

//revive:disable
var (
	CoreSystemNSDescription    = ffm("core.systemNSDescription", "FireFly system namespace")
	APIMessageTBD              = ffm("api.TBD", "TODO: Description") // TODO: Should remove this to force compile errors everywhere it is referenced, and fill in the missing descriptions
	APISuccessResponse         = ffm("api.success", "Success")
	APIRequestTimeoutDesc      = ffm("api.requestTimeout", "Server-side request timeout (millseconds, or set a custom suffix like 10s)")
	APIFilterParamDesc         = ffm("api.filterParam", "Data filter field. Prefixes supported: > >= < <= @ ^ ! !@ !^")
	APIFilterSortDesc          = ffm("api.filterSort", "Sort field. For multi-field sort use comma separated values (or multiple query values) with '-' prefix for descending")
	APIFilterAscendingDesc     = ffm("api.filterAscending", "Ascending sort order (overrides all fields in a multi-field sort)")
	APIFilterDescendingDesc    = ffm("api.filterDescending", "Descending sort order (overrides all fields in a multi-field sort)")
	APIFilterSkipDesc          = ffm("api.filterSkip", "The number of records to skip (max: %d). Unsuitable for bulk operations")
	APIFilterLimitDesc         = ffm("api.filterLimit", "The maximum number of records to return (max: %d)")
	APIFilterCountDesc         = ffm("api.filterCount", "Return a total count as well as items (adds extra database processing)")
	APIFetchDataDesc           = ffm("api.fetchData", "Fetch the data and include it in the messages returned")
	APIConfirmQueryParam       = ffm("api.confirmQueryParam", "When true the HTTP request blocks until the message is confirmed")
	APIHistogramStartTimeParam = ffm("api.histogramStartTime", "Start time of the data to be fetched")
	APIHistogramEndTimeParam   = ffm("api.histogramEndTime", "End time of the data to be fetched")
	APIHistogramBucketsParam   = ffm("api.histogramBuckets", "Number of buckets between start time and end time")
	WebhooksOptJSON            = ffm("ws.options.json", "Whether to assume the response body is JSON, regardless of the returned Content-Type")
	WebhooksOptReply           = ffm("ws.options.reply", "Whether to automatically send a reply event, using the body returned by the webhook")
	WebhooksOptHeaders         = ffm("ws.options.headers", "Static headers to set on the webhook request")
	WebhooksOptQuery           = ffm("ws.options.query", "Static query params to set on the webhook request")
	WebhooksOptInput           = ffm("ws.options.input", "A set of options to extract data from the first JSON input data in the incoming message. Only applies if withData=true")
	WebhooksOptInputQuery      = ffm("ws.options.inputQuery", "A top-level property of the first data input, to use for query parameters")
	WebhooksOptInputHeaders    = ffm("ws.options.inputHeaders", "A top-level property of the first data input, to use for headers")
	WebhooksOptInputBody       = ffm("ws.options.inputBody", "A top-level property of the first data input, to use for the request body. Default is the whole first body")
	WebhooksOptFastAck         = ffm("ws.options.fastAck", "When true the event will be acknowledged before the webhook is invoked, allowing parallel invocations")
	WebhooksOptURL             = ffm("ws.options.url", "Webhook url to invoke. Can be relative if a base URL is set in the webhook plugin config")
	WebhooksOptMethod          = ffm("ws.options.method", "Webhook method to invoke. Default=POST")
	WebhooksOptReplyTag        = ffm("ws.options.replytype", "The tag to set on the reply message")
	WebhooksOptReplyTx         = ffm("ws.options.replyTX", "The transaction type to set on the reply message")
	WebhooksOptInputPath       = ffm("ws.options.inputPath", "A top-level property of the first data input, to use for a path to append with escaping to the webhook path")
	WebhooksOptInputReplyTx    = ffm("ws.options.inputReplyTX", "A top-level property of the first data input, to use to dynamically set whether to pin the response (so the requester can choose)")
)
