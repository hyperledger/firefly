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

package i18n

var (
	registeredPrefixes = map[string]string{
		"FF00":  "FireFly Common Utilities",
		"FF10":  "FireFly Core",
		"FF201": "FireFly Transaction Manager",
		"FF202": "FireFly Signer",
		"FF99":  "Test prefix",
	}
)

//revive:disable
var (
	MsgConfigFailed               = FFE("FF00101", "Failed to read config", 500)
	MsgBigIntTooLarge             = FFE("FF00103", "Byte length of serialized integer is too large %d (max=%d)")
	MsgBigIntParseFailed          = FFE("FF00104", "Failed to parse JSON value '%s' into BigInt")
	MsgTypeRestoreFailed          = FFE("FF00105", "Failed to restore type '%T' into '%T'")
	MsgInvalidHex                 = FFE("FF00106", "Invalid hex supplied", 400)
	MsgInvalidWrongLenB32         = FFE("FF00107", "Byte length must be 32 (64 hex characters)", 400)
	MsgUnknownValidatorType       = FFE("FF00108", "Unknown validator type: '%s'", 400)
	MsgDataValueIsNull            = FFE("FF00109", "Data value is null", 400)
	MsgBlobMismatchSealingData    = FFE("FF00110", "Blob mismatch when sealing data")
	MsgUnknownFieldValue          = FFE("FF00111", "Unknown %s '%v'", 400)
	MsgMissingRequiredField       = FFE("FF00112", "Field '%s' is required", 400)
	MsgDataInvalidHash            = FFE("FF00113", "Invalid data: hashes do not match Hash=%s Expected=%s", 400)
	MsgNilID                      = FFE("FF00114", "ID is nil")
	MsgGroupMustHaveMembers       = FFE("FF00115", "Group must have at least one member", 400)
	MsgEmptyMemberIdentity        = FFE("FF00116", "Identity is blank in member %d")
	MsgEmptyMemberNode            = FFE("FF00117", "Node is blank in member %d")
	MsgDuplicateMember            = FFE("FF00118", "Member %d is a duplicate org+node combination: %s", 400)
	MsgGroupInvalidHash           = FFE("FF00119", "Invalid group: hashes do not match Hash=%s Expected=%s", 400)
	MsgInvalidDIDForType          = FFE("FF00120", "Invalid FireFly DID '%s' for type='%s' namespace='%s' name='%s'", 400)
	MsgCustomIdentitySystemNS     = FFE("FF00121", "Custom identities cannot be defined in the '%s' namespace", 400)
	MsgSystemIdentityCustomNS     = FFE("FF00122", "System identities must be defined in the '%s' namespace", 400)
	MsgNilParentIdentity          = FFE("FF00124", "Identity of type '%s' must have a valid parent", 400)
	MsgNilOrNullObject            = FFE("FF00125", "Object is null")
	MsgUnknownIdentityType        = FFE("FF00126", "Unknown identity type: %s", 400)
	MsgJSONObjectParseFailed      = FFE("FF00127", "Failed to parse '%s' as JSON")
	MsgNilDataReferenceSealFail   = FFE("FF00128", "Invalid message: nil data reference at index %d", 400)
	MsgDupDataReferenceSealFail   = FFE("FF00129", "Invalid message: duplicate data reference at index %d", 400)
	MsgInvalidTXTypeForMessage    = FFE("FF00130", "Invalid transaction type for sending a message: %s", 400)
	MsgVerifyFailedNilHashes      = FFE("FF00131", "Invalid message: nil hashes", 400)
	MsgVerifyFailedInvalidHashes  = FFE("FF00132", "Invalid message: hashes do not match Hash=%s Expected=%s DataHash=%s DataHashExpected=%s", 400)
	MsgDuplicateArrayEntry        = FFE("FF00133", "Duplicate %s at index %d: '%s'", 400)
	MsgTooManyItems               = FFE("FF00134", "Maximum number of %s items is %d (supplied=%d)", 400)
	MsgFieldTooLong               = FFE("FF00135", "Field '%s' maximum length is %d", 400)
	MsgTimeParseFail              = FFE("FF00136", "Cannot parse time as RFC3339, Unix, or UnixNano: '%s'", 400)
	MsgDurationParseFail          = FFE("FF00137", "Unable to parse '%s' as duration string, or millisecond number", 400)
	MsgInvalidUUID                = FFE("FF00138", "Invalid UUID supplied", 400)
	MsgSafeCharsOnly              = FFE("FF00139", "Field '%s' must include only alphanumerics (a-zA-Z0-9), dot (.), dash (-) and underscore (_)", 400)
	MsgInvalidName                = FFE("FF00140", "Field '%s' must be 1-64 characters, including alphanumerics (a-zA-Z0-9), dot (.), dash (-) and underscore (_), and must start/end in an alphanumeric", 400)
	MsgNoUUID                     = FFE("FF00141", "Field '%s' must not be a UUID", 400)
	MsgInvalidFilterField         = FFE("FF00142", "Unknown filter '%s'", 400)
	MsgInvalidValueForFilterField = FFE("FF00143", "Unable to parse value for filter '%s'", 400)
	MsgFieldMatchNoNull           = FFE("FF00144", "Comparison operator for field '%s' cannot accept a null value", 400)
	MsgFieldTypeNoStringMatching  = FFE("FF00145", "Field '%s' of type '%s' does not support partial or case-insensitive string matching", 400)
	MsgWSSendTimedOut             = FFE("FF00146", "Websocket send timed out")
	MsgWSClosing                  = FFE("FF00147", "Websocket closing")
	MsgWSConnectFailed            = FFE("FF00148", "Websocket connect failed")
	MsgInvalidURL                 = FFE("FF00149", "Invalid URL: '%s'")
	MsgWSHeartbeatTimeout         = FFE("FF00150", "Websocket heartbeat timed out after %.2fms", 500)
)
