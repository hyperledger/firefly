// Copyright © 2021 Kaleido, Inc.
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

package i18n

var (
	MsgConfigFailed            = ffm("FF10101", "Failed to read config: %s")
	MsgPutDefDataTypeDesc      = ffm("FF10102", "Asynchronously add a new data type")
	MsgJSONDecodeFailed        = ffm("FF10103", "Failed to decode input JSON")
	MsgAPIServerStartFailed    = ffm("FF10104", "Unable to start listener on %s: %s")
	MsgTLSConfigFailed         = ffm("FF10105", "Failed to initialize TLS configuration")
	MsgInvalidCAFile           = ffm("FF10106", "Invalid CA certificates file")
	MsgResponseMarshalError    = ffm("FF10107", "Failed to serialize response data")
	MsgWebsocketClientError    = ffm("FF10108", "Error received from WebSocket client: %s")
	Msg404NotFound             = ffm("FF10109", "Not found")
	MsgUnknownBlockchainPlugin = ffm("FF10110", "Unknown blockchain plugin: %s")
	MsgEthconnectRESTErr       = ffm("FF10111", "Error from ethconnect: %s")
)
