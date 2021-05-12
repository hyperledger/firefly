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
	MsgConfigFailed               = ffm("FF10101", "Failed to read config")
	MsgTBD                        = ffm("FF10102", "TODO: Description")
	MsgJSONDecodeFailed           = ffm("FF10103", "Failed to decode input JSON")
	MsgAPIServerStartFailed       = ffm("FF10104", "Unable to start listener on %s: %s")
	MsgTLSConfigFailed            = ffm("FF10105", "Failed to initialize TLS configuration")
	MsgInvalidCAFile              = ffm("FF10106", "Invalid CA certificates file")
	MsgResponseMarshalError       = ffm("FF10107", "Failed to serialize response data", 400)
	MsgWebsocketClientError       = ffm("FF10108", "Error received from WebSocket client: %s")
	Msg404NotFound                = ffm("FF10109", "Not found", 404)
	MsgUnknownBlockchainPlugin    = ffm("FF10110", "Unknown blockchain plugin: %s")
	MsgEthconnectRESTErr          = ffm("FF10111", "Error from ethconnect: %s")
	MsgDBInitFailed               = ffm("FF10112", "Database initialization failed")
	MsgDBQueryBuildFailed         = ffm("FF10113", "Database query builder failed")
	MsgDBBeginFailed              = ffm("FF10114", "Database begin transaction failed")
	MsgDBQueryFailed              = ffm("FF10115", "Database query failed")
	MsgDBInsertFailed             = ffm("FF10116", "Database insert failed")
	MsgDBUpdateFailed             = ffm("FF10117", "Database update failed")
	MsgDBDeleteFailed             = ffm("FF10118", "Database delete failed")
	MsgDBCommitFailed             = ffm("FF10119", "Database commit failed")
	MsgDBMissingJoin              = ffm("FF10120", "Database missing expected join entry in table '%s' for id '%s'")
	MsgDBReadErr                  = ffm("FF10121", "Database resultset read error from table '%s'")
	MsgUnknownDatabasePlugin      = ffm("FF10122", "Unknown database plugin '%s'")
	MsgNullDataReferenceID        = ffm("FF10123", "Data id is null in message data reference %d")
	MsgDupDataReferenceID         = ffm("FF10124", "Duplicate data ID in message '%s'")
	MsgScanFailed                 = ffm("FF10125", "Failed to restore type '%T' into '%T'")
	MsgUnregisteredBatchType      = ffm("FF10126", "Unregistered batch type '%s'")
	MsgBatchDispatchTimeout       = ffm("FF10127", "Timed out dispatching work to batch")
	MsgInitializationNilDepError  = ffm("FF10128", "Initialization error due to unmet dependency")
	MsgNilResponseNon204          = ffm("FF10129", "No output from API call")
	MsgInvalidContentType         = ffm("FF10130", "Invalid content type", 415)
	MsgInvalidName                = ffm("FF10131", "Field '%s' must be 1-64 characters, including alphanumerics (a-zA-Z0-9), dot (.), dash (-) and underscore (_), and must start/end in an alphanumeric", 400)
	MsgUnknownFieldValue          = ffm("FF10132", "Unknown '%s", 400)
	MsgDataNotFound               = ffm("FF10133", "Data not found with UUID '%s", 400)
	MsgUnknownP2PFilesystemPlugin = ffm("FF10134", "Unknown P2P Filesystem plugin '%s'")
	MsgIPFSHashDecodeFailed       = ffm("FF10135", "Failed to decode IPFS hash into 32byte value '%s'")
	MsgIPFSRESTErr                = ffm("FF10136", "Error from IPFS: %s")
	MsgSerializationFailed        = ffm("FF10137", "Serialization failed")
	MsgMissingPluginConfig        = ffm("FF10138", "Missing configuration '%s' for %s")
	MsgMissingDataHashIndex       = ffm("FF10139", "Missing data hash for index '%d' in message", 400)
	MsgMissingRequiredField       = ffm("FF10140", "Field '%s' is required", 400)
	MsgInvalidEthAddress          = ffm("FF10141", "Supplied ethereum address is invalid", 400)
	MsgInvalidUUID                = ffm("FF10142", "Invalid UUID supplied", 400)
	Msg404NoResult                = ffm("FF10143", "No result found", 404)
	MsgNilDataReferenceSealFail   = ffm("FF10144", "Invalid message: nil data reference at index %d", 400)
	MsgDupDataReferenceSealFail   = ffm("FF10145", "Invalid message: duplicate data reference at index %d", 400)
	MsgVerifyFailedInvalidHashes  = ffm("FF10146", "Invalid message: hashes do not match", 400)
	MsgVerifyFailedNilHashes      = ffm("FF10147", "Invalid message: nil hashes", 400)
	MsgInvalidFilterField         = ffm("FF10148", "Unknown filter '%s'", 400)
	MsgInvalidValueForFilterField = ffm("FF10149", "Unable to parse value for filter '%s'", 400)
	MsgUnsupportedSQLOpInFilter   = ffm("FF10150", "No SQL mapping implemented for filter operator '%s'", 400)
	MsgJSONDataParseFailed        = ffm("FF10151", "Failed to parse '%s' as JSON")
	MsgFilterParamDesc            = ffm("FF10152", "Data filter field. Prefixes supported: > >= < <= @ ^ ! !@ !^")
	MsgSuccessResponse            = ffm("FF10153", "Success")
	MsgFilterSortDesc             = ffm("FF10154", "Sort field. For multi-field sort use comma separate fields, or multiple query values")
	MsgFilterDescendingDesc       = ffm("FF10155", "Descending. Descending sort order (applies to all fields in a multi-field sort)")
	MsgFilterSkipDesc             = ffm("FF10156", "Pagination - skip. The number of records to skip")
	MsgFilterLimitDesc            = ffm("FF10157", "Pagination - limit. The maximum number of records to return")
	MsgContextCanceled            = ffm("FF10158", "Context cancelled")
	MsgWSSendTimedOut             = ffm("FF10159", "Websocket send timed out")
	MsgWSClosing                  = ffm("FF10160", "Websocket closing")
	MsgWSConnectFailed            = ffm("FF10161", "Websocket connect failed")
	MsgInvalidURL                 = ffm("FF10162", "Invalid URL: '%s'")
	MsgDBMigrationFailed          = ffm("FF10163", "Database migration failed")
)
