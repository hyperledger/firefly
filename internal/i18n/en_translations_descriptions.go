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

//revive:disable

/*
This file contains the English field level descriptions that are used in
OpenAPI Spec generation. Each struct field that wants to use one of these
needs to have an ffstruct tag on it, indicating the name of the struct.
That will be combined with the JSON field name (note, it is not the GO
field name, but the JSON serialized name), separated by a "." This is the
key used to lookup the translation below. If it is not found, the description
is left blank in the OpenAPI spec
*/
var (
	// MessageHeader field descriptions
	MessageHeaderID        = ffm("MessageHeader.id", "The UUID of the message. Unique to each message")
	MessageHeaderCID       = ffm("MessageHeader.cid", "The correlation ID of the message. Set this when a message is a response to another message")
	MessageHeaderType      = ffm("MessageHeader.type", "The type of the message")
	MessageHeaderTxType    = ffm("MessageHeader.txtype", "The type of transaction used to order/deliver this message")
	MessageHeaderCreated   = ffm("MessageHeader.created", "The creation time of the message")
	MessageHeaderNamespace = ffm("MessageHeader.namespace", "The namespace of the message")
	MessageHeaderGroup     = ffm("MessageHeader.group", "Private messages only - the identifier hash of the privacy group. Derived from the name and member list of the group")
	MessageHeaderTopics    = ffm("MessageHeader.topics", "A message topic associates this message with an ordered stream of data. A custom topic should be assigned - using the default topic is discouraged")
	MessageHeaderTag       = ffm("MessageHeader.tag", "The message tag indicates the purpose of the message to the applications that process it")
	MessageHeaderDataHash  = ffm("MessageHeader.datahash", "A single hash representing all data in the message. Derived from the array of data ids+hashes attached to this message")

	// Message field descriptions
	MessageHeader    = ffm("Message.header", "The message header contains all fields that are used to build the message hash")
	MessageHash      = ffm("Message.hash", "The hash of the message. Derived from the header, which includes the data hash")
	MessageBatchID   = ffm("Messages.batch", "The UUID of the batch in which the message was pinned/transferred")
	MessageState     = ffm("Message.state", "The current state of the message")
	MessageConfirmed = ffm("Message.confirmed", "The timestamp of when the message was confirmed/rejected")
	MessageData      = ffm("Message.data", "The list of data elements attached to the message")
	MessagePins      = ffm("Message.pins", "For private messages, a unique pin hash:nonce is assigned for each topic")

	// MessageInOut field descriptions
	MessageInOutData  = ffm("MessageInOut.data", "For input allows you to specify data in-line in the message, that will be turned into data attachments. For output when fetchdata is used on API calls, includes the in-line data payloads of all data attachments")
	MessageInOutGroup = ffm("MessageInOut.group", "Allows you to specify details of the private group of recipients in-line in the message. Alternative to using the header.group to specify the hash of a group that has been previously resolved")

	// InputGroup field descriptions
	InputGroupName    = ffm("InputGroup.name", "Optional name for the group. Allows you to have multiple separate groups with the same list of participants")
	InputGroupMembers = ffm("InputGroup.members", "An array of members of the group. If no identities local to the sending node are included, then the organization owner of the local node is added automatically")

	// DataRefOrValue field descriptions
	DataRefOrValueValidator = ffm("DataRefOrValue.validator", "The data validator type to use for in-line data")
	DataRefOrValueDatatype  = ffm("DataRefOrValue.datatype", "The optional datatype to use for validation of the in-line data")
	DataRefOrValueValue     = ffm("DataRefOrValue.value", "The in-line value for the data. Can be any JSON type - object, array, string, number or boolean")
	DataRefOrValueBlob      = ffm("DataRefOrValue.blob", "An optional in-line hash reference to a previously uploaded binary data blob")

	// MessageRef field descriptions
	MessageRefID   = ffm("MessageRef.id", "The UUID of the referenced message")
	MessageRefHash = ffm("MessageRef.hash", "The hash of the referenced message")

	// Group field descriptions
	GroupNamespace = ffm("Group.namespace", "The namespace of the group")
	GroupName      = ffm("Group.name", "The optional name of the group, allowing multiple unique groups to exist with the same list of recipients")
	GroupMembers   = ffm("Group.members", "The list of members in this privacy group")
	GroupMessage   = ffm("Group.message", "The message used to broadcast this group privately to the members")
	GroupHash      = ffm("Group.hash", "The identifier hash of this group. Derived from the name and group members")
	GroupCreated   = ffm("Group.created", "The time when the group was first used to send a message in the network")

	// DataRef field descriptions
	DataRefID   = ffm("DataRef.id", "The UUID of the referenced data resource")
	DataRefHash = ffm("DataRef.hash", "The hash of the referenced data")

	// BlobRef field descriptions
	BlobRefHash   = ffm("BlobRef.hash", "The hash of the binary blob data")
	BlobRefSize   = ffm("BlobRef.size", "The size of the binary data")
	BlobRefName   = ffm("BlobRef.name", "The name field from the metadata attached to the blob, commonly used as a path/filename, and indexed for search")
	BlobRefPublic = ffm("BlobRef.public", "If this data has been published to shared storage, this field is the id of the data in the shared storage plugin (IPFS hash etc.)")

	// Data field descriptions
	DataID        = ffm("Data.id", "The UUID of the data resource")
	DataValidator = ffm("Data.validator", "The data validator type")
	DataNamespace = ffm("Data.namespace", "The namespace of the data resource")
	DataHash      = ffm("Data.hash", "The hash of the data resource. Derived from the value and the hash of any binary blob attachment")
	DataCreated   = ffm("Data.created", "The creation time of the data resource")
	DataDatatype  = ffm("Data.datatype", "The optional datatype to use of validation of this data")
	DataValue     = ffm("Data.value", "The value for the data, stored in the FireFly core database. Can be any JSON type - object, array, string, number or boolean. Can be combined with a binary blob attachment")
	DataBlob      = ffm("Data.blob", "An optional hash reference to a binary blob attachment")

	// SignerRef field descriptions
	SignerRefAuthor = ffm("SignerRef.author", "The DID of identity of the submitter")
	SignerRefKey    = ffm("SignerRef.key", "The on-chain signing key used to sign the transaction")

	// IdentityClaim field descriptions
	IdentityClaimIdentity = ffm("IdentityClaim.identity", "The identity being claimed")

	// IdentityVerification field descriptions
	IdentityVerificationClaim    = ffm("IdentityVerification.claim", "The UUID of the message containing the identity claim being verified")
	IdentityVerificationIdentity = ffm("IdentityVerification.identity", "The identity being verified")

	// IdentityUpdate field descriptions
	IdentityUpdateIdentity = ffm("IdentityUpdate.identity", "The identity being updated")
	IdentityUpdateProfile  = ffm("IdentityUpdate.profile", "The new profile, which is replaced in its entirety when the update is confirmed")

	// MessageManifestEntry field descriptions
	MessageManifestEntry = ffm("MessageManifestEntry.topics", "The count of topics in the message")

	// BatchHeader field descriptions
	BatchHeaderID        = ffm("BatchHeader.id", "The UUID of the batch")
	BatchHeaderType      = ffm("BatchHeader.type", "The type of the batch")
	BatchHeaderNamespace = ffm("BatchHeader.namespace", "The namespace of the batch")
	BatchHeaderNode      = ffm("BatchHeader.node", "The UUID of the node that generated the batch")
	BatchHeaderGroup     = ffm("BatchHeader.group", "The privacy group the batch is sent to, for private batches")
	BatchHeaderCreated   = ffm("BatchHeader.created", "The time the batch was sealed")

	// BatchManifest field descriptions
	BatchManifestVersion  = ffm("BatchManifest.version", "The version of the manifest generated")
	BatchManifestID       = ffm("BatchManifest.id", "The UUID of the batch")
	BatchManifestTX       = ffm("BatchManifest.tx", "The FireFly transaction associated with this batch")
	BatchManifestMessages = ffm("BatchManifest.messages", "Array of manifest entries, succinctly summarizing the messages in the batch")
	BatchManifestData     = ffm("BatchManifest.data", "Array of manifest entries, succinctly summarizing the data in the batch")

	// BatchPersisted field descriptions
	BatchPersistedHash       = ffm("BatchPersisted.hash", "The hash of the manifest of the batch")
	BatchPersistedManifest   = ffm("BatchPersisted.manifest", "The manifest of the batch")
	BatchPersistedTX         = ffm("BatchPersisted.tx", "The FireFly transaction associated with this batch")
	BatchPersistedPayloadRef = ffm("BatchPersisted.payloadRef", "For broadcast batches, this is the reference to the binary batch in shared storage")
	BatchPersistedConfirmed  = ffm("BatchPersisted.confirmed", "The time when the batch was confirmed")

	// TransactionRef field descriptions
	TransactionRef = ffm("TransactionRef.type", "The type of the FireFly transaction")
	TransactionID  = ffm("TransactionRef.id", "The UUID of the FireFly transaction")

	// BlockchainEvent field descriptions
	BlockchainEventID         = ffm("BlockchainEvent.id", "The UUID assigned to the event by FireFly")
	BlockchainEventSource     = ffm("BlockchainEvent.source", "The blockchain plugin or token service that detected the event")
	BlockchainEventNamespace  = ffm("BlockchainEvent.namespace", "The namespace of the listener that detected this blockchain event")
	BlockchainEventName       = ffm("BlockchainEvent.name", "The name of the event in the blockchain smart contract")
	BlockchainEventListener   = ffm("BlockchainEvent.listener", "The UUID of the listener that detected this event, or nil for the built-in BatchPin events in the system namespace")
	BlockchainEventProtocolID = ffm("BlockchainEvent.protocolId", "A alphanumerically sortable string that represents this event uniquely on the blockchain. Convention for plugins is zero-padded values BLOCKNUMBER/TXN_INDEX/EVENT_INDEX")
	BlockchainEventOutput     = ffm("BlockchainEvent.output", "The data output by the event, parsed to JSON according to the interface of the smart contract")
	BlockchainEventInfo       = ffm("BlockchainEvent.info", "Detailed blockchain specific information about the event, as generated by the blockchain connector")
	BlockchainEventTimestamp  = ffm("BlockchainEvent.timestamp", "The time allocated to this event by the blockchain. This is the block timestamp for most blockchain connectors")
	BlockchainEventTX         = ffm("BlockchainEvent.tx", "If this blockchain event is coorelated to FireFly transaction such as a FireFly submitted token transfer, this field is set to the UUID of the FireFly transaction")

	// ChartHistogramBucket field descriptions
	ChartHistogramBucketCount     = ffm("ChartHistogramBucket.count", "Total count of entires in this time bucket within the histogram")
	ChartHistogramBucketTimestamp = ffm("ChartHistogramBucket.timestamp", "Starting timestamp for the bucket")
	ChartHistogramBucketTypes     = ffm("ChartHistogramBucket.types", "Array of separate counts for individual types of record within the bucket")

	// ChartHistogramType field descriptions
	ChartHistogramTypeCount = ffm("ChartHistogramType.count", "Count of entires of a given type within a bucket")
	ChartHistogramTypeType  = ffm("ChartHistogramType.type", "Name of the type")

	// ContractAPI field descriptions
	ContractAPIID        = ffm("ContractAPI.id", "The UUID of the contract API")
	ContractAPINamespace = ffm("ContractAPI.namespace", "The namespace of the contract API")
	ContractAPIInterface = ffm("ContractAPI.interface", "Reference to the FireFly Interface definition associated with the contract API")
	ContractAPILedger    = ffm("ContractAPI.ledger", "If this API is tied to an individual instance of a smart contract, this field can include a blockchain specific ledger identifier. For example a Hyperledger Fabric channel name")
	ContractAPILocation  = ffm("ContractAPI.location", "If this API is tied to an individual instance of a smart contract, this field can include a blockchain specific contract identifier. For example an Ethereum contract address")
	ContractAPIName      = ffm("ContractAPI.name", "The name that is used in the URL to access the API")
	ContractAPIMessage   = ffm("ContractAPI.message", "The UUID of the broadcast message that was used to publish this API to the network")
	ContractAPIURLs      = ffm("ContractAPI.urls", "The URLs to use to access the API")

	// ContractURLs field descriptions
	ContractURLsOpenAPI = ffm("ContractURLs.openapi", "The URL to download the OpenAPI v3 (Swagger) description for the API generated in JSON or YAML format")
	ContractURLsUI      = ffm("ContractURLs.ui", "The URL to use in a web browser to access the SwaggerUI explorer/exerciser for the API")

	// FFIReference field descriptions
	FFIReferenceID      = ffm("FFIReference.id", "The UUID of the FireFly interface")
	FFIReferenceName    = ffm("FFIReference.name", "The name of the FireFly interface")
	FFIReferenceVersion = ffm("FFIReference.version", "The version of the FireFly interface")

	// FFI field descriptions
	FFIID          = ffm("FFI.id", "The UUID of the FireFly interface (FFI) smart contract definition")
	FFIMessage     = ffm("FFI.message", "The UUID of the broadcast message that was used to publish this FFI to the network")
	FFINamespace   = ffm("FFI.namespace", "The namespace of the FFI")
	FFIName        = ffm("FFI.name", "The name of the FFI - usually matching the smart contract name")
	FFIDescription = ffm("FFI.description", "A description of the smart contract this FFI represents")
	FFIVersion     = ffm("FFI.version", "A version for the FFI - use of semantic versioning such as 'v1.0.1' is encouraged")
	FFIMethods     = ffm("FFI.methods", "An array of smart contract method definitions")
	FFIEvents      = ffm("FFI.events", "An array of smart contract event definitions")

	// FFIMethod field descriptions
	FFIMethodID          = ffm("FFIMethod.id", "The UUID of the FFI method definition")
	FFIMethodContract    = ffm("FFIMethod.contract", "The UUID of the FFI smart contract definition that this method is part of")
	FFIMethodName        = ffm("FFIMethod.name", "The name of the method")
	FFIMethodNamespace   = ffm("FFIMethod.namespace", "The namespace of the FFI")
	FFIMethodPathname    = ffm("FFIMethod.pathname", "The unique name allocated to this method within the FFI for use on URL paths. Supports contracts that have multiple method overrides with the same name")
	FFIMethodDescription = ffm("FFIMethod.description", "A description of the smart contract method")
	FFIMethodParams      = ffm("FFIMethod.params", "An array of method parameter/argument definitions")
	FFIMethodReturns     = ffm("FFIMethod.returns", "An array of method return definitions")

	// FFIEvent field descriptions
	FFIEventID          = ffm("FFIEvent.id", "The UUID of the FFI event definition")
	FFIEventContract    = ffm("FFIEvent.contract", "The UUID of the FFI smart contract definition that this event is part of")
	FFIEventName        = ffm("FFIEvent.name", "The name of the event")
	FFIEventNamespace   = ffm("FFIEvent.namespace", "The namespace of the FFI")
	FFIEventPathname    = ffm("FFIEvent.pathname", "The unique name allocated to this event within the FFI for use on URL paths. Supports contracts that have multiple event overrides with the same name")
	FFIEventDescription = ffm("FFIEvent.description", "A description of the smart contract event")
	FFIEventParams      = ffm("FFIEvent.params", "An array of event parameter/argument definitions")

	// FFIParam field descriptions
	FFIParamName   = ffm("FFIParam.name", "The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract")
	FFIParamSchema = ffm("FFIParam.schema", "FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail")
)
