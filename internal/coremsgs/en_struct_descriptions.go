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

import "golang.org/x/text/language"

//revive:disable

/*
This file contains the English field level descriptions that are used in
OpenAPI Spec generation. Each struct field that wants to use one of these
needs to have an ffstruct tag on it, indicating the name of the struct.
That will be combined with the JSON field name (note, it is not the GO
field name, but the JSON serialized name), separated by a "." This is the
key used to lookup the translation below. If it is not found, the description
is left blank in the OpenAPI spec

Example:
// message.go
type Message struct {
	Header    MessageHeader `ffstruct:"Message" json:"header"`

// en_translations_descriptions.go
MessageHeader    = ffm(language.AmericanEnglish, "Message.header", "The message header")

*/
var (
	// MessageHeader field descriptions
	MessageHeaderID        = ffm(language.AmericanEnglish, "MessageHeader.id", "The UUID of the message. Unique to each message")
	MessageHeaderCID       = ffm(language.AmericanEnglish, "MessageHeader.cid", "The correlation ID of the message. Set this when a message is a response to another message")
	MessageHeaderType      = ffm(language.AmericanEnglish, "MessageHeader.type", "The type of the message")
	MessageHeaderTxType    = ffm(language.AmericanEnglish, "MessageHeader.txtype", "The type of transaction used to order/deliver this message")
	MessageHeaderCreated   = ffm(language.AmericanEnglish, "MessageHeader.created", "The creation time of the message")
	MessageHeaderNamespace = ffm(language.AmericanEnglish, "MessageHeader.namespace", "The namespace of the message")
	MessageHeaderGroup     = ffm(language.AmericanEnglish, "MessageHeader.group", "Private messages only - the identifier hash of the privacy group. Derived from the name and member list of the group")
	MessageHeaderTopics    = ffm(language.AmericanEnglish, "MessageHeader.topics", "A message topic associates this message with an ordered stream of data. A custom topic should be assigned - using the default topic is discouraged")
	MessageHeaderTag       = ffm(language.AmericanEnglish, "MessageHeader.tag", "The message tag indicates the purpose of the message to the applications that process it")
	MessageHeaderDataHash  = ffm(language.AmericanEnglish, "MessageHeader.datahash", "A single hash representing all data in the message. Derived from the array of data ids+hashes attached to this message")

	// Message field descriptions
	MessageHeader    = ffm(language.AmericanEnglish, "Message.header", "The message header contains all fields that are used to build the message hash")
	MessageHash      = ffm(language.AmericanEnglish, "Message.hash", "The hash of the message. Derived from the header, which includes the data hash")
	MessageBatchID   = ffm(language.AmericanEnglish, "Message.batch", "The UUID of the batch in which the message was pinned/transferred")
	MessageState     = ffm(language.AmericanEnglish, "Message.state", "The current state of the message")
	MessageConfirmed = ffm(language.AmericanEnglish, "Message.confirmed", "The timestamp of when the message was confirmed/rejected")
	MessageData      = ffm(language.AmericanEnglish, "Message.data", "The list of data elements attached to the message")
	MessagePins      = ffm(language.AmericanEnglish, "Message.pins", "For private messages, a unique pin hash:nonce is assigned for each topic")

	// MessageInOut field descriptions
	MessageInOutData  = ffm(language.AmericanEnglish, "MessageInOut.data", "For input allows you to specify data in-line in the message, that will be turned into data attachments. For output when fetchdata is used on API calls, includes the in-line data payloads of all data attachments")
	MessageInOutGroup = ffm(language.AmericanEnglish, "MessageInOut.group", "Allows you to specify details of the private group of recipients in-line in the message. Alternative to using the header.group to specify the hash of a group that has been previously resolved")

	// InputGroup field descriptions
	InputGroupName    = ffm(language.AmericanEnglish, "InputGroup.name", "Optional name for the group. Allows you to have multiple separate groups with the same list of participants")
	InputGroupMembers = ffm(language.AmericanEnglish, "InputGroup.members", "An array of members of the group. If no identities local to the sending node are included, then the organization owner of the local node is added automatically")

	// DataRefOrValue field descriptions
	DataRefOrValueValidator = ffm(language.AmericanEnglish, "DataRefOrValue.validator", "The data validator type to use for in-line data")
	DataRefOrValueDatatype  = ffm(language.AmericanEnglish, "DataRefOrValue.datatype", "The optional datatype to use for validation of the in-line data")
	DataRefOrValueValue     = ffm(language.AmericanEnglish, "DataRefOrValue.value", "The in-line value for the data. Can be any JSON type - object, array, string, number or boolean")
	DataRefOrValueBlob      = ffm(language.AmericanEnglish, "DataRefOrValue.blob", "An optional in-line hash reference to a previously uploaded binary data blob")

	// MessageRef field descriptions
	MessageRefID   = ffm(language.AmericanEnglish, "MessageRef.id", "The UUID of the referenced message")
	MessageRefHash = ffm(language.AmericanEnglish, "MessageRef.hash", "The hash of the referenced message")

	// Group field descriptions
	GroupNamespace = ffm(language.AmericanEnglish, "Group.namespace", "The namespace of the group")
	GroupName      = ffm(language.AmericanEnglish, "Group.name", "The optional name of the group, allowing multiple unique groups to exist with the same list of recipients")
	GroupMembers   = ffm(language.AmericanEnglish, "Group.members", "The list of members in this privacy group")
	GroupMessage   = ffm(language.AmericanEnglish, "Group.message", "The message used to broadcast this group privately to the members")
	GroupHash      = ffm(language.AmericanEnglish, "Group.hash", "The identifier hash of this group. Derived from the name and group members")
	GroupCreated   = ffm(language.AmericanEnglish, "Group.created", "The time when the group was first used to send a message in the network")

	// MemberInput field descriptions
	MemberInputIdentity = ffm(language.AmericanEnglish, "MemberInput.identity", "The DID of the group member. On input can be a UUID or org name, and will be resolved to a DID")
	MemberInputNode     = ffm(language.AmericanEnglish, "MemberInput.node", "The UUID of the node that will receive a copy of the off-chain message for the identity. The first applicable node for the identity will be picked automatically on input if not specified")

	// Member field descriptions
	MemberIdentity = ffm(language.AmericanEnglish, "Member.identity", "The DID of the group member")
	MembertNode    = ffm(language.AmericanEnglish, "Member.node", "The UUID of the node that receives a copy of the off-chain message for the identity")

	// DataRef field descriptions
	DataRefID   = ffm(language.AmericanEnglish, "DataRef.id", "The UUID of the referenced data resource")
	DataRefHash = ffm(language.AmericanEnglish, "DataRef.hash", "The hash of the referenced data")

	// BlobRef field descriptions
	BlobRefHash   = ffm(language.AmericanEnglish, "BlobRef.hash", "The hash of the binary blob data")
	BlobRefSize   = ffm(language.AmericanEnglish, "BlobRef.size", "The size of the binary data")
	BlobRefName   = ffm(language.AmericanEnglish, "BlobRef.name", "The name field from the metadata attached to the blob, commonly used as a path/filename, and indexed for search")
	BlobRefPublic = ffm(language.AmericanEnglish, "BlobRef.public", "If this data has been published to shared storage, this field is the id of the data in the shared storage plugin (IPFS hash etc.)")

	// Data field descriptions
	DataID        = ffm(language.AmericanEnglish, "Data.id", "The UUID of the data resource")
	DataValidator = ffm(language.AmericanEnglish, "Data.validator", "The data validator type")
	DataNamespace = ffm(language.AmericanEnglish, "Data.namespace", "The namespace of the data resource")
	DataHash      = ffm(language.AmericanEnglish, "Data.hash", "The hash of the data resource. Derived from the value and the hash of any binary blob attachment")
	DataCreated   = ffm(language.AmericanEnglish, "Data.created", "The creation time of the data resource")
	DataDatatype  = ffm(language.AmericanEnglish, "Data.datatype", "The optional datatype to use of validation of this data")
	DataValue     = ffm(language.AmericanEnglish, "Data.value", "The value for the data, stored in the FireFly core database. Can be any JSON type - object, array, string, number or boolean. Can be combined with a binary blob attachment")
	DataBlob      = ffm(language.AmericanEnglish, "Data.blob", "An optional hash reference to a binary blob attachment")

	// DatatypeRef field descriptions
	DatatypeRefName    = ffm(language.AmericanEnglish, "DatatypeRef.name", "The name of the datatype")
	DatatypeRefVersion = ffm(language.AmericanEnglish, "DatatypeRef.version", "The version of the datatype. Semantic versioning is encouraged, such as v1.0.1")

	// Datatype field descriptions
	DatatypeID        = ffm(language.AmericanEnglish, "Datatype.id", "The UUID of the datatype")
	DatatypeMessage   = ffm(language.AmericanEnglish, "Datatype.message", "The UUID of the broadcast message that was used to publish this datatype to the network")
	DatatypeValidator = ffm(language.AmericanEnglish, "Datatype.validator", "The validator that should be used to verify this datatype")
	DatatypeNamespace = ffm(language.AmericanEnglish, "Datatype.namespace", "The namespace of the datatype. Data resources can only be created referencing datatypes in the same namespace")
	DatatypeName      = ffm(language.AmericanEnglish, "Datatype.name", "The name of the datatype")
	DatatypeVersion   = ffm(language.AmericanEnglish, "Datatype.version", "The version of the datatype. Multiple versions can exist with the same name. Use of semantic versioning is encourages, such as v1.0.1")
	DatatypeHash      = ffm(language.AmericanEnglish, "Datatype.hash", "The hash of the value, such as the JSON schema. Allows all parties to be confident they have the exact same rules for verifying data created against a datatype")
	DatatypeCreated   = ffm(language.AmericanEnglish, "Datatype.created", "The time the datatype was created")
	DatatypeValue     = ffm(language.AmericanEnglish, "Datatype.value", "The definition of the datatype, in the syntax supported by the validator (such as a JSON Schema definition)")

	// SignerRef field descriptions
	SignerRefAuthor = ffm(language.AmericanEnglish, "SignerRef.author", "The DID of identity of the submitter")
	SignerRefKey    = ffm(language.AmericanEnglish, "SignerRef.key", "The on-chain signing key used to sign the transaction")

	// MessageManifestEntry field descriptions
	MessageManifestEntry = ffm(language.AmericanEnglish, "MessageManifestEntry.topics", "The count of topics in the message")

	// BatchHeader field descriptions
	BatchHeaderID        = ffm(language.AmericanEnglish, "BatchHeader.id", "The UUID of the batch")
	BatchHeaderType      = ffm(language.AmericanEnglish, "BatchHeader.type", "The type of the batch")
	BatchHeaderNamespace = ffm(language.AmericanEnglish, "BatchHeader.namespace", "The namespace of the batch")
	BatchHeaderNode      = ffm(language.AmericanEnglish, "BatchHeader.node", "The UUID of the node that generated the batch")
	BatchHeaderGroup     = ffm(language.AmericanEnglish, "BatchHeader.group", "The privacy group the batch is sent to, for private batches")
	BatchHeaderCreated   = ffm(language.AmericanEnglish, "BatchHeader.created", "The time the batch was sealed")

	// BatchManifest field descriptions
	BatchManifestVersion  = ffm(language.AmericanEnglish, "BatchManifest.version", "The version of the manifest generated")
	BatchManifestID       = ffm(language.AmericanEnglish, "BatchManifest.id", "The UUID of the batch")
	BatchManifestTX       = ffm(language.AmericanEnglish, "BatchManifest.tx", "The FireFly transaction associated with this batch")
	BatchManifestMessages = ffm(language.AmericanEnglish, "BatchManifest.messages", "Array of manifest entries, succinctly summarizing the messages in the batch")
	BatchManifestData     = ffm(language.AmericanEnglish, "BatchManifest.data", "Array of manifest entries, succinctly summarizing the data in the batch")

	// BatchPersisted field descriptions
	BatchPersistedHash       = ffm(language.AmericanEnglish, "BatchPersisted.hash", "The hash of the manifest of the batch")
	BatchPersistedManifest   = ffm(language.AmericanEnglish, "BatchPersisted.manifest", "The manifest of the batch")
	BatchPersistedTX         = ffm(language.AmericanEnglish, "BatchPersisted.tx", "The FireFly transaction associated with this batch")
	BatchPersistedPayloadRef = ffm(language.AmericanEnglish, "BatchPersisted.payloadRef", "For broadcast batches, this is the reference to the binary batch in shared storage")
	BatchPersistedConfirmed  = ffm(language.AmericanEnglish, "BatchPersisted.confirmed", "The time when the batch was confirmed")

	// Transaction field descriptions
	TransactionID            = ffm(language.AmericanEnglish, "Transaction.id", "The UUID of the FireFly transaction")
	TransactionType          = ffm(language.AmericanEnglish, "Transaction.type", "The type of the FireFly transaction")
	TransactionNamespace     = ffm(language.AmericanEnglish, "Transaction.namespace", "The namespace of the FireFly transaction")
	TransactionCreated       = ffm(language.AmericanEnglish, "Transaction.created", "The time the transaction was created on this node. Note the transaction is individually created with the same UUID on each participant in the FireFly transaction")
	TransactionBlockchainID  = ffm(language.AmericanEnglish, "Transaction.blockchainId", "The blockchain transaction ID, in the format specific to the blockchain involved in the transaction. Not all FireFly transactions include a blockchain")
	TransactionBlockchainIDs = ffm(language.AmericanEnglish, "Transaction.blockchainIds", "The blockchain transaction ID, in the format specific to the blockchain involved in the transaction. Not all FireFly transactions include a blockchain. FireFly transactions are extensible to support multiple blockchain transactions")

	// Operation field description
	OperationID          = ffm(language.AmericanEnglish, "Operation.id", "The UUID of the operation")
	OperationNamespace   = ffm(language.AmericanEnglish, "Operation.namespace", "The namespace of the operation")
	OperationTransaction = ffm(language.AmericanEnglish, "Operation.tx", "The UUID of the FireFly transaction the operation is part of")
	OperationType        = ffm(language.AmericanEnglish, "Operation.type", "The type of the operation")
	OperationStatus      = ffm(language.AmericanEnglish, "Operation.status", "The current status of the operation")
	OperationPlugin      = ffm(language.AmericanEnglish, "Operation.plugin", "The plugin responsible for performing the operation")
	OperationInput       = ffm(language.AmericanEnglish, "Operation.input", "The input to this operation")
	OperationOutput      = ffm(language.AmericanEnglish, "Operation.output", "Any output reported back from the plugin for this operation")
	OperationError       = ffm(language.AmericanEnglish, "Operation.error", "Any error reported back from the plugin for this operation")
	OperationCreated     = ffm(language.AmericanEnglish, "Operation.created", "The time the operation was created")
	OperationUpdated     = ffm(language.AmericanEnglish, "Operation.updated", "The last update time of the operation")
	OperationRetry       = ffm(language.AmericanEnglish, "Operation.retry", "If this operation was initiated as a retry to a previous operation, this field points to the UUID of the operation being retried")

	// BlockchainEvent field descriptions
	BlockchainEventID         = ffm(language.AmericanEnglish, "BlockchainEvent.id", "The UUID assigned to the event by FireFly")
	BlockchainEventSource     = ffm(language.AmericanEnglish, "BlockchainEvent.source", "The blockchain plugin or token service that detected the event")
	BlockchainEventNamespace  = ffm(language.AmericanEnglish, "BlockchainEvent.namespace", "The namespace of the listener that detected this blockchain event")
	BlockchainEventName       = ffm(language.AmericanEnglish, "BlockchainEvent.name", "The name of the event in the blockchain smart contract")
	BlockchainEventListener   = ffm(language.AmericanEnglish, "BlockchainEvent.listener", "The UUID of the listener that detected this event, or nil for built-in events in the system namespace")
	BlockchainEventProtocolID = ffm(language.AmericanEnglish, "BlockchainEvent.protocolId", "An alphanumerically sortable string that represents this event uniquely on the blockchain (convention for plugins is zero-padded values BLOCKNUMBER/TXN_INDEX/EVENT_INDEX)")
	BlockchainEventOutput     = ffm(language.AmericanEnglish, "BlockchainEvent.output", "The data output by the event, parsed to JSON according to the interface of the smart contract")
	BlockchainEventInfo       = ffm(language.AmericanEnglish, "BlockchainEvent.info", "Detailed blockchain specific information about the event, as generated by the blockchain connector")
	BlockchainEventTimestamp  = ffm(language.AmericanEnglish, "BlockchainEvent.timestamp", "The time allocated to this event by the blockchain. This is the block timestamp for most blockchain connectors")
	BlockchainEventTX         = ffm(language.AmericanEnglish, "BlockchainEvent.tx", "If this blockchain event is coorelated to FireFly transaction such as a FireFly submitted token transfer, this field is set to the UUID of the FireFly transaction")

	// ChartHistogram field descriptions
	ChartHistogramCount     = ffm(language.AmericanEnglish, "ChartHistogram.count", "Total count of entries in this time bucket within the histogram")
	ChartHistogramTimestamp = ffm(language.AmericanEnglish, "ChartHistogram.timestamp", "Starting timestamp for the bucket")
	ChartHistogramTypes     = ffm(language.AmericanEnglish, "ChartHistogram.types", "Array of separate counts for individual types of record within the bucket")
	ChartHistogramIsCapped  = ffm(language.AmericanEnglish, "ChartHistogram.isCapped", "Indicates whether there are more results in this bucket that are not being displayed")

	// ChartHistogramType field descriptions
	ChartHistogramTypeCount = ffm(language.AmericanEnglish, "ChartHistogramType.count", "Count of entries of a given type within a bucket")
	ChartHistogramTypeType  = ffm(language.AmericanEnglish, "ChartHistogramType.type", "Name of the type")

	// ContractAPI field descriptions
	ContractAPIID        = ffm(language.AmericanEnglish, "ContractAPI.id", "The UUID of the contract API")
	ContractAPINamespace = ffm(language.AmericanEnglish, "ContractAPI.namespace", "The namespace of the contract API")
	ContractAPIInterface = ffm(language.AmericanEnglish, "ContractAPI.interface", "Reference to the FireFly Interface definition associated with the contract API")
	ContractAPILocation  = ffm(language.AmericanEnglish, "ContractAPI.location", "If this API is tied to an individual instance of a smart contract, this field can include a blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel")
	ContractAPIName      = ffm(language.AmericanEnglish, "ContractAPI.name", "The name that is used in the URL to access the API")
	ContractAPIMessage   = ffm(language.AmericanEnglish, "ContractAPI.message", "The UUID of the broadcast message that was used to publish this API to the network")
	ContractAPIURLs      = ffm(language.AmericanEnglish, "ContractAPI.urls", "The URLs to use to access the API")

	// ContractURLs field descriptions
	ContractURLsOpenAPI = ffm(language.AmericanEnglish, "ContractURLs.openapi", "The URL to download the OpenAPI v3 (Swagger) description for the API generated in JSON or YAML format")
	ContractURLsUI      = ffm(language.AmericanEnglish, "ContractURLs.ui", "The URL to use in a web browser to access the SwaggerUI explorer/exerciser for the API")

	// FFIReference field descriptions
	FFIReferenceID      = ffm(language.AmericanEnglish, "FFIReference.id", "The UUID of the FireFly interface")
	FFIReferenceName    = ffm(language.AmericanEnglish, "FFIReference.name", "The name of the FireFly interface")
	FFIReferenceVersion = ffm(language.AmericanEnglish, "FFIReference.version", "The version of the FireFly interface")

	// FFI field descriptions
	FFIID          = ffm(language.AmericanEnglish, "FFI.id", "The UUID of the FireFly interface (FFI) smart contract definition")
	FFIMessage     = ffm(language.AmericanEnglish, "FFI.message", "The UUID of the broadcast message that was used to publish this FFI to the network")
	FFINamespace   = ffm(language.AmericanEnglish, "FFI.namespace", "The namespace of the FFI")
	FFIName        = ffm(language.AmericanEnglish, "FFI.name", "The name of the FFI - usually matching the smart contract name")
	FFIDescription = ffm(language.AmericanEnglish, "FFI.description", "A description of the smart contract this FFI represents")
	FFIVersion     = ffm(language.AmericanEnglish, "FFI.version", "A version for the FFI - use of semantic versioning such as 'v1.0.1' is encouraged")
	FFIMethods     = ffm(language.AmericanEnglish, "FFI.methods", "An array of smart contract method definitions")
	FFIEvents      = ffm(language.AmericanEnglish, "FFI.events", "An array of smart contract event definitions")

	// FFIMethod field descriptions
	FFIMethodID          = ffm(language.AmericanEnglish, "FFIMethod.id", "The UUID of the FFI method definition")
	FFIMethodInterface   = ffm(language.AmericanEnglish, "FFIMethod.interface", "The UUID of the FFI smart contract definition that this method is part of")
	FFIMethodName        = ffm(language.AmericanEnglish, "FFIMethod.name", "The name of the method")
	FFIMethodNamespace   = ffm(language.AmericanEnglish, "FFIMethod.namespace", "The namespace of the FFI")
	FFIMethodPathname    = ffm(language.AmericanEnglish, "FFIMethod.pathname", "The unique name allocated to this method within the FFI for use on URL paths. Supports contracts that have multiple method overrides with the same name")
	FFIMethodDescription = ffm(language.AmericanEnglish, "FFIMethod.description", "A description of the smart contract method")
	FFIMethodParams      = ffm(language.AmericanEnglish, "FFIMethod.params", "An array of method parameter/argument definitions")
	FFIMethodReturns     = ffm(language.AmericanEnglish, "FFIMethod.returns", "An array of method return definitions")

	// FFIEvent field descriptions
	FFIEventID          = ffm(language.AmericanEnglish, "FFIEvent.id", "The UUID of the FFI event definition")
	FFIEventInterface   = ffm(language.AmericanEnglish, "FFIEvent.interface", "The UUID of the FFI smart contract definition that this event is part of")
	FFIEventName        = ffm(language.AmericanEnglish, "FFIEvent.name", "The name of the event")
	FFIEventNamespace   = ffm(language.AmericanEnglish, "FFIEvent.namespace", "The namespace of the FFI")
	FFIEventPathname    = ffm(language.AmericanEnglish, "FFIEvent.pathname", "The unique name allocated to this event within the FFI for use on URL paths. Supports contracts that have multiple event overrides with the same name")
	FFIEventDescription = ffm(language.AmericanEnglish, "FFIEvent.description", "A description of the smart contract event")
	FFIEventParams      = ffm(language.AmericanEnglish, "FFIEvent.params", "An array of event parameter/argument definitions")
	FFIEventSignature   = ffm(language.AmericanEnglish, "FFIEvent.signature", "The stringified signature of the event, as computed by the blockchain plugin")

	// FFIParam field descriptions
	FFIParamName   = ffm(language.AmericanEnglish, "FFIParam.name", "The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract")
	FFIParamSchema = ffm(language.AmericanEnglish, "FFIParam.schema", "FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail")

	// FFIGenerationRequest field descriptions
	FFIGenerationRequestNamespace   = ffm(language.AmericanEnglish, "FFIGenerationRequest.namespace", "The namespace into which the FFI will be generated")
	FFIGenerationRequestName        = ffm(language.AmericanEnglish, "FFIGenerationRequest.name", "The name of the FFI to generate")
	FFIGenerationRequestDescription = ffm(language.AmericanEnglish, "FFIGenerationRequest.description", "The description of the FFI to be generated. Defaults to the description extracted by the blockchain specific converter utility")
	FFIGenerationRequestVersion     = ffm(language.AmericanEnglish, "FFIGenerationRequest.version", "The version of the FFI to generate")
	FFIGenerationRequestInput       = ffm(language.AmericanEnglish, "FFIGenerationRequest.input", "A blockchain connector specific payload. For example in Ethereum this is a JSON structure containing an 'abi' array, and optionally a 'devdocs' array.")

	// ContractListener field descriptions
	ContractListenerID        = ffm(language.AmericanEnglish, "ContractListener.id", "The UUID of the smart contract listener")
	ContractListenerInterface = ffm(language.AmericanEnglish, "ContractListener.interface", "A reference to an existing FFI, containing pre-registered type information for the event")
	ContractListenerNamespace = ffm(language.AmericanEnglish, "ContractListener.namespace", "The namespace of the listener, which defines the namespace of all blockchain events detected by this listener")
	ContractListenerName      = ffm(language.AmericanEnglish, "ContractListener.name", "A descriptive name for the listener")
	ContractListenerBackendID = ffm(language.AmericanEnglish, "ContractListener.backendId", "An ID assigned by the blockchain connector to this listener")
	ContractListenerLocation  = ffm(language.AmericanEnglish, "ContractListener.location", "A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel")
	ContractListenerCreated   = ffm(language.AmericanEnglish, "ContractListener.created", "The creation time of the listener")
	ContractListenerEvent     = ffm(language.AmericanEnglish, "ContractListener.event", "The definition of the event, either provided in-line when creating the listener, or extracted from the referenced FFI")
	ContractListenerTopic     = ffm(language.AmericanEnglish, "ContractListener.topic", "A topic to set on the FireFly event that is emitted each time a blockchain event is detected from the blockchain. Setting this topic on a number of listeners allows applications to easily subscribe to all events they need")
	ContractListenerOptions   = ffm(language.AmericanEnglish, "ContractListener.options", "Options that control how the listener subscribes to events from the underlying blockchain")
	ContractListenerEventPath = ffm(language.AmericanEnglish, "ContractListener.eventPath", "When creating a listener from an existing FFI, this is the pathname of the event on that FFI to be detected by this listener")
	ContractListenerSignature = ffm(language.AmericanEnglish, "ContractListener.signature", "The stringified signature of the event, as computed by the blockchain plugin")

	// ContractListenerOptions field descriptions
	ContractListenerOptionsFirstEvent = ffm(language.AmericanEnglish, "ContractListenerOptions.firstEvent", "A blockchain specific string, such as a block number, to start listening from. The special strings 'oldest' and 'newest' are supported by all blockchain connectors. Default is 'newest'")

	// DIDDocument field descriptions
	DIDDocumentContext            = ffm(language.AmericanEnglish, "DIDDocument.@context", "See https://www.w3.org/TR/did-core/#json-ld")
	DIDDocumentID                 = ffm(language.AmericanEnglish, "DIDDocument.id", "See https://www.w3.org/TR/did-core/#did-document-properties")
	DIDDocumentAuthentication     = ffm(language.AmericanEnglish, "DIDDocument.authentication", "See https://www.w3.org/TR/did-core/#did-document-properties")
	DIDDocumentVerificationMethod = ffm(language.AmericanEnglish, "DIDDocument.verificationMethod", "See https://www.w3.org/TR/did-core/#did-document-properties")

	// DIDVerificationMethod field descriptions
	DIDVerificationMethodID                  = ffm(language.AmericanEnglish, "DIDVerificationMethod.id", "See https://www.w3.org/TR/did-core/#service-properties")
	DIDVerificationMethodController          = ffm(language.AmericanEnglish, "DIDVerificationMethod.controller", "See https://www.w3.org/TR/did-core/#service-properties")
	DIDVerificationMethodType                = ffm(language.AmericanEnglish, "DIDVerificationMethod.type", "See https://www.w3.org/TR/did-core/#service-properties")
	DIDVerificationMethodBlockchainAccountID = ffm(language.AmericanEnglish, "DIDVerificationMethod.blockchainAcountId", "For blockchains like Ethereum that represent signing identities directly by their public key summarized in an account string")
	DIDVerificationMethodMSPIdentityString   = ffm(language.AmericanEnglish, "DIDVerificationMethod.mspIdentityString", "For Hyperledger Fabric where the signing identity is represented by an MSP identifier (containing X509 certificate DN strings) that were validated by your local MSP")
	DIDVerificationMethodDataExchangePeerID  = ffm(language.AmericanEnglish, "DIDVerificationMethod.dataExchangePeerID", "A string provided by your Data Exchange plugin, that it uses a technology specific mechanism to validate against when messages arrive from this identity")

	// Event field descriptions
	EventID          = ffm(language.AmericanEnglish, "Event.id", "The UUID assigned to this event by your local FireFly node")
	EventSequence    = ffm(language.AmericanEnglish, "Event.sequence", "A sequence indicating the order in which events are delivered to your application. Assure to be unique per event in your local FireFly database (unlike the created timestamp)")
	EventType        = ffm(language.AmericanEnglish, "Event.type", "All interesting activity in FireFly is emitted as a FireFly event, of a given type. The 'type' combined with the 'reference' can be used to determine how to process the event within your application")
	EventNamespace   = ffm(language.AmericanEnglish, "Event.namespace", "The namespace of the event. Your application must subscribe to events within a namespace")
	EventReference   = ffm(language.AmericanEnglish, "Event.reference", "The UUID of an resource that is the subject of this event. The event type determines what type of resource is referenced, and whether this field might be unset")
	EventCorrelator  = ffm(language.AmericanEnglish, "Event.correlator", "For message events, this is the 'header.cid' field from the referenced message. For certain other event types, a secondary object is referenced such as a token pool")
	EventTransaction = ffm(language.AmericanEnglish, "Event.tx", "The UUID of a transaction that is event is part of. Not all events are part of a transaction")
	EventTopic       = ffm(language.AmericanEnglish, "Event.topic", "A stream of information this event relates to. For message confirmation events, a separate event is emitted for each topic in the message. For blockchain events, the listener specifies the topic. Rules exist for how the topic is set for other event types")
	EventCreated     = ffm(language.AmericanEnglish, "Event.created", "The time the event was emitted. Not guaranteed to be unique, or to increase between events in the same order as the final sequence events are delivered to your application. As such, the 'sequence' field should be used instead of the 'created' field for querying events in the exact order they are delivered to applications")

	// EnrichedEvent field descriptions
	EnrichedEventBlockchainEvent   = ffm(language.AmericanEnglish, "EnrichedEvent.blockchainEvent", "A blockchain event if referenced by the FireFly event")
	EnrichedEventContractAPI       = ffm(language.AmericanEnglish, "EnrichedEvent.contractAPI", "A Contract API if referenced by the FireFly event")
	EnrichedEventContractInterface = ffm(language.AmericanEnglish, "EnrichedEvent.contractInterface", "A Contract Interface (FFI) if referenced by the FireFly event")
	EnrichedEventDatatype          = ffm(language.AmericanEnglish, "EnrichedEvent.datatype", "A Datatype if referenced by the FireFly event")
	EnrichedEventIdentity          = ffm(language.AmericanEnglish, "EnrichedEvent.identity", "An Identity if referenced by the FireFly event")
	EnrichedEventMessage           = ffm(language.AmericanEnglish, "EnrichedEvent.message", "A Message if  referenced by the FireFly event")
	EnrichedEventNamespaceDetails  = ffm(language.AmericanEnglish, "EnrichedEvent.namespaceDetails", "Full resource detail of a Namespace if referenced by the FireFly event")
	EnrichedEventTokenApproval     = ffm(language.AmericanEnglish, "EnrichedEvent.tokenApproval", "A Token Approval if referenced by the FireFly event")
	EnrichedEventTokenPool         = ffm(language.AmericanEnglish, "EnrichedEvent.tokenPool", "A Token Pool if referenced by the FireFly event")
	EnrichedEventTokenTransfer     = ffm(language.AmericanEnglish, "EnrichedEvent.tokenTransfer", "A Token Transfer if referenced by the FireFly event")
	EnrichedEventTransaction       = ffm(language.AmericanEnglish, "EnrichedEvent.transaction", "A Transaction if associated with the FireFly event")

	// IdentityMessages field descriptions
	IdentityMessagesClaim        = ffm(language.AmericanEnglish, "IdentityMessages.claim", "The UUID of claim message")
	IdentityMessagesVerification = ffm(language.AmericanEnglish, "IdentityMessages.verification", "The UUID of claim message. Unset for root organization identities")
	IdentityMessagesUpdate       = ffm(language.AmericanEnglish, "IdentityMessages.update", "The UUID of the most recently applied update message. Unset if no updates have been confirmed")

	// Identity field descriptions
	IdentityID        = ffm(language.AmericanEnglish, "Identity.id", "The UUID of the identity")
	IdentityDID       = ffm(language.AmericanEnglish, "Identity.did", "The DID of the identity. Unique across namespaces within a FireFly network")
	IdentityType      = ffm(language.AmericanEnglish, "Identity.type", "The type of the identity")
	IdentityParent    = ffm(language.AmericanEnglish, "Identity.parent", "The UUID of the parent identity. Unset for root organization identities")
	IdentityNamespace = ffm(language.AmericanEnglish, "Identity.namespace", "The namespace of the identity. Organization and node identities are always defined in the ff_system namespace")
	IdentityName      = ffm(language.AmericanEnglish, "Identity.name", "The name of the identity. The name must be unique within the type and namespace")
	IdentityMessages  = ffm(language.AmericanEnglish, "Identity.messages", "References to the broadcast messages that established this identity and proved ownership of the associated verifiers (keys)")
	IdentityCreated   = ffm(language.AmericanEnglish, "Identity.created", "The creation time of the identity")
	IdentityUpdated   = ffm(language.AmericanEnglish, "Identity.updated", "The last update time of the identity profile")

	// IdentityProfile field descriptions
	IdentityProfileProfile     = ffm(language.AmericanEnglish, "IdentityProfile.profile", "A set of metadata for the identity. Part of the updatable profile information of an identity")
	IdentityProfileDescription = ffm(language.AmericanEnglish, "IdentityProfile.description", "A description of the identity. Part of the updatable profile information of an identity")

	// IdentityWithVerifiers field descriptions
	IdentityWithVerifiersVerifiers = ffm(language.AmericanEnglish, "IdentityWithVerifiers.verifiers", "The verifiers, such as blockchain signing keys, that have been bound to this identity and can be used to prove data orignates from that identity")

	// IdentityCreateDTO field descriptions
	IdentityCreateDTOParent = ffm(language.AmericanEnglish, "IdentityCreateDTO.parent", "On input the parent can be specified directly as the UUID of and existing identity, or as a DID to resolve to that identity, or an organization name. The parent must already have been registered, and its blockchain signing key must be available to the local node to sign the verification")
	IdentityCreateDTOKey    = ffm(language.AmericanEnglish, "IdentityCreateDTO.key", "The blockchain signing key to use to make the claim to the identity. Must be available to the local node to sign the identity claim. Will become a verifier on the established identity")

	// IdentityClaim field descriptions
	IdentityClaimIdentity = ffm(language.AmericanEnglish, "IdentityClaim.identity", "The identity being claimed")

	// IdentityVerification field descriptions
	IdentityVerificationClaim    = ffm(language.AmericanEnglish, "IdentityVerification.claim", "The UUID of the message containing the identity claim being verified")
	IdentityVerificationIdentity = ffm(language.AmericanEnglish, "IdentityVerification.identity", "The identity being verified")

	// IdentityUpdate field descriptions
	IdentityUpdateIdentity = ffm(language.AmericanEnglish, "IdentityUpdate.identity", "The identity being updated")
	IdentityUpdateProfile  = ffm(language.AmericanEnglish, "IdentityUpdate.profile", "The new profile, which is replaced in its entirety when the update is confirmed")

	// Verifier field descriptions
	VerifierHash      = ffm(language.AmericanEnglish, "Verifier.hash", "Hash used as a globally consistent identifier for this namespace + type + value combination on every node in the network")
	VerifierIdentity  = ffm(language.AmericanEnglish, "Verifier.identity", "The UUID of the parent identity that has claimed this verifier")
	VerifierType      = ffm(language.AmericanEnglish, "Verifier.type", "The type of the verifier")
	VerifierValue     = ffm(language.AmericanEnglish, "Verifier.value", "The verifier string, such as an Ethereum address, or Fabric MSP identifier")
	VerifierNamespace = ffm(language.AmericanEnglish, "Verifier.namespace", "The namespace of the verifier")
	VerifierCreated   = ffm(language.AmericanEnglish, "Verifier.created", "The time this verifier was created on this node")

	// Namespace field descriptions
	NamespaceID          = ffm(language.AmericanEnglish, "Namespace.id", "The UUID of the namespace. For locally established namespaces will be different on each node in the network. For broadcast namespaces, will be the same on every node")
	NamespaceMessage     = ffm(language.AmericanEnglish, "Namespace.message", "The UUID of broadcast message used to establish the namespace. Unset for local namespaces")
	NamespaceName        = ffm(language.AmericanEnglish, "Namespace.name", "The namespace name")
	NamespaceDescription = ffm(language.AmericanEnglish, "Namespace.description", "A description of the namespace")
	NamespaceType        = ffm(language.AmericanEnglish, "Namespace.type", "The type of the namespace")
	NamespaceCreated     = ffm(language.AmericanEnglish, "Namespace.created", "The time the namespace was created")

	// NodeStatus field descriptions
	NodeStatusNode = ffm(language.AmericanEnglish, "NodeStatus.node", "Details of the local node")
	NodeStatusOrg  = ffm(language.AmericanEnglish, "NodeStatus.org", "Details of the organization identity that operates this node")
	NodeDefaults   = ffm(language.AmericanEnglish, "NodeStatus.defaults", "Information about defaults configured on this node that appplications might need to query on startup")
	NodePlugins    = ffm(language.AmericanEnglish, "NodeStatus.plugins", "Information about plugins configured on this node")

	// NodeStatusNode field descriptions
	NodeStatusNodeName       = ffm(language.AmericanEnglish, "NodeStatusNode.name", "The name of this node, as specified in the local configuration")
	NodeStatusNodeRegistered = ffm(language.AmericanEnglish, "NodeStatusNode.registered", "Whether the node has been successfully registered")
	NodeStatusNodeID         = ffm(language.AmericanEnglish, "NodeStatusNode.id", "The UUID of the node, if registered")

	// NodeStatusOrg field descriptions
	NodeStatusOrgName       = ffm(language.AmericanEnglish, "NodeStatusOrg.name", "The name of the node operator organization, as specified in the local configuration")
	NodeStatusOrgRegistered = ffm(language.AmericanEnglish, "NodeStatusOrg.registered", "Whether the organization has been successfully registered")
	NodeStatusOrgDID        = ffm(language.AmericanEnglish, "NodeStatusOrg.did", "The DID of the organization identity, if registered")
	NodeStatusOrgID         = ffm(language.AmericanEnglish, "NodeStatusOrg.id", "The UUID of the organization, if registered")
	NodeStatusOrgVerifiers  = ffm(language.AmericanEnglish, "NodeStatusOrg.verifiers", "Array of verifiers (blockchain keys) owned by this identity")

	// NodeStatusDefaults field descriptions
	NodeStatusDefaultsNamespace = ffm(language.AmericanEnglish, "NodeStatusDefaults.namespace", "The default namespace on this node")

	// NodeStatusPlugins field descriptions
	NodeStatusPluginsBlockchain    = ffm(language.AmericanEnglish, "NodeStatusPlugins.blockchain", "The blockchain plugins on this node")
	NodeStatusPluginsDatabase      = ffm(language.AmericanEnglish, "NodeStatusPlugins.database", "The database plugins on this node")
	NodeStatusPluginsDataExchange  = ffm(language.AmericanEnglish, "NodeStatusPlugins.dataExchange", "The data exchange plugins on this node")
	Events                         = ffm(language.AmericanEnglish, "NodeStatusPlugins.events", "The event plugins on this node")
	NodeStatusPluginsIdentity      = ffm(language.AmericanEnglish, "NodeStatusPlugins.identity", "The identity plugins on this node")
	NodeStatusPluginsSharedStorage = ffm(language.AmericanEnglish, "NodeStatusPlugins.sharedStorage", "The shared storage plugins on this node")
	NodeStatusPluginsTokens        = ffm(language.AmericanEnglish, "NodeStatusPlugins.tokens", "The token plugins on this node")

	// NodeStatusPlugin field descriptions
	NodeStatusPluginName = ffm(language.AmericanEnglish, "NodeStatusPlugin.name", "The name of the plugin")
	NodeStatusPluginType = ffm(language.AmericanEnglish, "NodeStatusPlugin.pluginType", "The type of the plugin")

	// BatchManagerStatus field descriptions
	BatchManagerStatusProcessors = ffm(language.AmericanEnglish, "BatchManagerStatus.processors", "An array of currently active batch processors")

	// BatchProcessorStatus field descriptions
	BatchProcessorStatusDispatcher = ffm(language.AmericanEnglish, "BatchProcessorStatus.dispatcher", "The type of dispatcher for this processor")
	BatchProcessorStatusName       = ffm(language.AmericanEnglish, "BatchProcessorStatus.name", "The name of the processor, which includes details of the attributes of message are allocated to this processor")
	BatchProcessorStatusStatus     = ffm(language.AmericanEnglish, "BatchProcessorStatus.status", "The flush status for this batch processor")

	// BatchFlushStatus field descriptions
	BatchFlushStatusLastFlushTime        = ffm(language.AmericanEnglish, "BatchFlushStatus.lastFlushStartTime", "The last time a flush was performed")
	BatchFlushStatusFlushing             = ffm(language.AmericanEnglish, "BatchFlushStatus.flushing", "If a flush is in progress, this is the UUID of the batch being flushed")
	BatchFlushStatusBlocked              = ffm(language.AmericanEnglish, "BatchFlushStatus.blocked", "True if the batch flush is in a retry loop, due to errors being returned by the plugins")
	BatchFlushStatusLastFlushError       = ffm(language.AmericanEnglish, "BatchFlushStatus.lastFlushError", "The last error received by this batch processor while flushing")
	BatchFlushStatusLastFlushErrorTime   = ffm(language.AmericanEnglish, "BatchFlushStatus.lastFlushErrorTime", "The time of the last flush")
	BatchFlushStatusAverageBatchBytes    = ffm(language.AmericanEnglish, "BatchFlushStatus.averageBatchBytes", "The average byte size of each batch")
	BatchFlushStatusAverageBatchMessages = ffm(language.AmericanEnglish, "BatchFlushStatus.averageBatchMessages", "The average number of messages included in each batch")
	BatchFlushStatusAverageBatchData     = ffm(language.AmericanEnglish, "BatchFlushStatus.averageBatchData", "The average number of data attachments included in each batch")
	BatchFlushStatusAverageFlushTimeMS   = ffm(language.AmericanEnglish, "BatchFlushStatus.averageFlushTimeMS", "The average amount of time spent flushing each batch")
	BatchFlushStatusTotalBatches         = ffm(language.AmericanEnglish, "BatchFlushStatus.totalBatches", "The total count of batches flushed by this processor since it started")
	BatchFlushStatusTotalErrors          = ffm(language.AmericanEnglish, "BatchFlushStatus.totalErrors", "The total count of error flushed encountered by this processor since it started")

	// Pin field descriptions
	PinSequence   = ffm(language.AmericanEnglish, "Pin.sequence", "The order of the pin in the local FireFly database, which matches the order in which pins were delivered to FireFly by the blockchain connector event stream")
	PinMasked     = ffm(language.AmericanEnglish, "Pin.masked", "True if the pin is for a private message, and hence is masked with the group ID and salted with a nonce so observers of the blockchain cannot use pin hash to match this transaction to other transactions or participants")
	PinHash       = ffm(language.AmericanEnglish, "Pin.hash", "The hash represents a topic within a message in the batch. If a message has multiple topics, then multiple pins are created. If the message is private, the hash is masked for privacy")
	PinBatch      = ffm(language.AmericanEnglish, "Pin.batch", "The UUID of the batch of messages this pin is part of")
	PinBatchHash  = ffm(language.AmericanEnglish, "Pin.batchHash", "The manifest hash batch of messages this pin is part of")
	PinIndex      = ffm(language.AmericanEnglish, "Pin.index", "The index of this pin within the batch. One pin is created for each topic, of each message in the batch")
	PinDispatched = ffm(language.AmericanEnglish, "Pin.dispatched", "Once true, this pin has been processed and will not be processed again")
	PinSigner     = ffm(language.AmericanEnglish, "Pin.signer", "The blockchain signing key that submitted this transaction, as passed through to FireFly by the smart contract that emitted the blockchain event")
	PinCreated    = ffm(language.AmericanEnglish, "Pin.created", "The time the FireFly node created the pin")

	// Subscription field descriptions
	SubscriptionID        = ffm(language.AmericanEnglish, "Subscription.id", "The UUID of the subscription")
	SubscriptionNamespace = ffm(language.AmericanEnglish, "Subscription.namespace", "The namespace of the subscription. A subscription will only receive events generated in the namespace of the subscription")
	SubscriptionName      = ffm(language.AmericanEnglish, "Subscription.name", "The name of the subscription. The application specifies this name when it connects, in order to attach to the subscription and receive events that arrived while it was disconnected. If multiple apps connect to the same subscription, events are workload balanced across the connected application instances")
	SubscriptionTransport = ffm(language.AmericanEnglish, "Subscription.transport", "The transport plugin responsible for event delivery (WebSockets, Webhooks, JMS, NATS etc.)")
	SubscriptionFilter    = ffm(language.AmericanEnglish, "Subscription.filter", "Server-side filter to apply to events")
	SubscriptionOptions   = ffm(language.AmericanEnglish, "Subscription.options", "Subscription options")
	SubscriptionEphemeral = ffm(language.AmericanEnglish, "Subscription.ephemeral", "Ephemeral subscriptions only exist as long as the application is connected, and as such will miss events that occur while the application is disconnected, and cannot be created administratively. You can create one over over a connected WebSocket connection")
	SubscriptionCreated   = ffm(language.AmericanEnglish, "Subscription.created", "Creation time of the subscription")
	SubscriptionUpdated   = ffm(language.AmericanEnglish, "Subscription.updated", "Last time the subscription was updated")

	// SubscriptionFilter field descriptions
	SubscriptionFilterEvents           = ffm(language.AmericanEnglish, "SubscriptionFilter.events", "Regular expression to apply to the event type, to subscribe to a subset of event types")
	SubscriptionFilterTopic            = ffm(language.AmericanEnglish, "SubscriptionFilter.topic", "Regular expression to apply to the topic of the event, to subscribe to a subset of topics. Note for messages sent with multiple topics, a separate event is emitted for each topic")
	SubscriptionFilterMessage          = ffm(language.AmericanEnglish, "SubscriptionFilter.message", "Filters specific to message events. If an event is not a message event, these filters are ignored")
	SubscriptionFilterTransaction      = ffm(language.AmericanEnglish, "SubscriptionFilter.transaction", "Filters specific to events with a transaction. If an event is not associated with a transaction, this filter is ignored")
	SubscriptionFilterBlockchainEvent  = ffm(language.AmericanEnglish, "SubscriptionFilter.blockchainevent", "Filters specific to blockchain events. If an event is not a blockchain event, these filters are ignored")
	SubscriptionFilterDeprecatedTopics = ffm(language.AmericanEnglish, "SubscriptionFilter.topics", "Deprecated: Please use 'topic' instead")
	SubscriptionFilterDeprecatedTag    = ffm(language.AmericanEnglish, "SubscriptionFilter.tag", "Deprecated: Please use 'message.tag' instead")
	SubscriptionFilterDeprecatedGroup  = ffm(language.AmericanEnglish, "SubscriptionFilter.group", "Deprecated: Please use 'message.group' instead")
	SubscriptionFilterDeprecatedAuthor = ffm(language.AmericanEnglish, "SubscriptionFilter.author", "Deprecated: Please use 'message.author' instead")

	// SubscriptionMessageFilter field descriptions
	SubscriptionMessageFilterTag    = ffm(language.AmericanEnglish, "SubscriptionMessageFilter.tag", "Regular expression to apply to the message 'header.tag' field")
	SubscriptionMessageFilterGroup  = ffm(language.AmericanEnglish, "SubscriptionMessageFilter.group", "Regular expression to apply to the message 'header.group' field")
	SubscriptionMessageFilterAuthor = ffm(language.AmericanEnglish, "SubscriptionMessageFilter.author", "Regular expression to apply to the message 'header.author' field")

	// SubscriptionTransactionFilter field descriptions
	SubscriptionTransactionFilterType = ffm(language.AmericanEnglish, "SubscriptionTransactionFilter.type", "Regular expression to apply to the transaction 'type' field")

	// SubscriptionBlockchainEventFilter field descriptions
	SubscriptionBlockchainEventFilterName     = ffm(language.AmericanEnglish, "SubscriptionBlockchainEventFilter.name", "Regular expression to apply to the blockchain event 'name' field, which is the name of the event in the underlying blockchain smart contract")
	SubscriptionBlockchainEventFilterListener = ffm(language.AmericanEnglish, "SubscriptionBlockchainEventFilter.listener", "Regular expression to apply to the blockchain event 'listener' field, which is the UUID of the event listener. So you can restrict your subscription to certain blockchain listeners. Alternatively to avoid your application need to know listener UUIDs you can set the 'topic' field of blockchain event listeners, and use a topic filter on your subscriptions")

	// SubscriptionCoreOptions field descriptions
	SubscriptionCoreOptionsFirstEvent = ffm(language.AmericanEnglish, "SubscriptionCoreOptions.firstEvent", "Whether your appplication would like to receive events from the 'oldest' event emitted by your FireFly node (from the beginning of time), or the 'newest' event (from now), or a specific event sequence. Default is 'newest'")
	SubscriptionCoreOptionsReadAhead  = ffm(language.AmericanEnglish, "SubscriptionCoreOptions.readAhead", "The number of events to stream ahead to your application, while waiting for confirmation of consumption of those events. At least once delivery semantics are used in FireFly, so if your application crashes/reconnects this is the maximum number of events you would expect to be redelivered after it restarts")
	SubscriptionCoreOptionsWithData   = ffm(language.AmericanEnglish, "SubscriptionCoreOptions.withData", "Whether message events delivered over the subscription, should be packaged with the full data of those messages in-line as part of the event JSON payload. Or if the application should make separate REST calls to download that data. May not be supported on some transports.")

	// TokenApproval field descriptions
	TokenApprovalLocalID         = ffm(language.AmericanEnglish, "TokenApproval.localId", "The UUID of this token approval, in the local FireFly node")
	TokenApprovalPool            = ffm(language.AmericanEnglish, "TokenApproval.pool", "The UUID the token pool this approval applies to")
	TokenApprovalConnector       = ffm(language.AmericanEnglish, "TokenApproval.connector", "The name of the token connector, as specified in the FireFly core configuration file. Required on input when there are more than one token connectors configured")
	TokenApprovalKey             = ffm(language.AmericanEnglish, "TokenApproval.key", "The blockchain signing key for the approval request. On input defaults to the first signing key of the organization that operates the node")
	TokenApprovalOperator        = ffm(language.AmericanEnglish, "TokenApproval.operator", "The blockchain identity that is granted the approval")
	TokenApprovalApproved        = ffm(language.AmericanEnglish, "TokenApproval.approved", "Whether this record grants permission for an operator to perform actions on the token balance (true), or revokes permission (false)")
	TokenApprovalInfo            = ffm(language.AmericanEnglish, "TokenApproval.info", "Token connector specific information about the approval operation, such as whether it applied to a limited balance of a fungible token. See your chosen token connector documentation for details")
	TokenApprovalNamespace       = ffm(language.AmericanEnglish, "TokenApproval.namespace", "The namespace for the approval, which must match the namespace of the token pool")
	TokenApprovalProtocolID      = ffm(language.AmericanEnglish, "TokenApproval.protocolId", "An alphanumerically sortable string that represents this event uniquely with respect to the blockchain")
	TokenApprovalSubject         = ffm(language.AmericanEnglish, "TokenApproval.subject", "A string identifying the parties and entities in the scope of this approval, as provided by the token connector")
	TokenApprovalActive          = ffm(language.AmericanEnglish, "TokenApproval.active", "Indicates if this approval is currently active (only one approval can be active per subject)")
	TokenApprovalCreated         = ffm(language.AmericanEnglish, "TokenApproval.created", "The creation time of the token approval")
	TokenApprovalTX              = ffm(language.AmericanEnglish, "TokenApproval.tx", "If submitted via FireFly, this will reference the UUID of the FireFly transaction (if the token connector in use supports attaching data)")
	TokenApprovalBlockchainEvent = ffm(language.AmericanEnglish, "TokenApproval.blockchainEvent", "The UUID of the blockchain event")
	TokenApprovalConfig          = ffm(language.AmericanEnglish, "TokenApproval.config", "Input only field, with token connector specific configuration of the approval.  See your chosen token connector documentation for details")

	// TokenApprovalInputInput field descriptions
	TokenApprovalInputInputPool = ffm(language.AmericanEnglish, "TokenApprovalInput.pool", "The name or UUID of a token pool. Required if more than one pool exists.")

	// TokenBalance field descriptions
	TokenBalancePool       = ffm(language.AmericanEnglish, "TokenBalance.pool", "The UUID the token pool this balance entry applies to")
	TokenBalanceTokenIndex = ffm(language.AmericanEnglish, "TokenBalance.tokenIndex", "The index of the token within the pool that this balance applies to")
	TokenBalanceURI        = ffm(language.AmericanEnglish, "TokenBalance.uri", "The URI of the token this balance entry applies to")
	TokenBalanceConnector  = ffm(language.AmericanEnglish, "TokenBalance.connector", "The token connector that is responsible for the token pool of this balance entry")
	TokenBalanceNamespace  = ffm(language.AmericanEnglish, "TokenBalance.namespace", "The namespace of the token pool for this balance entry")
	TokenBalanceKey        = ffm(language.AmericanEnglish, "TokenBalance.key", "The blockchain signing identity this balance applies to")
	TokenBalanceBalance    = ffm(language.AmericanEnglish, "TokenBalance.balance", "The numeric balance. For non-fungible tokens will always be 1. For fungible tokens, the number of decimals for the token pool should be considered when interpreting the balance. For example, with 18 decimals a fractional balance of 10.234 will be returned as 10,234,000,000,000,000,000")
	TokenBalanceUpdated    = ffm(language.AmericanEnglish, "TokenBalance.updated", "The last time the balance was updated by applying a transfer event")

	// TokenBalance field descriptions
	TokenConnectorName = ffm(language.AmericanEnglish, "TokenConnector.name", "The name of the token connector, as configured in the FireFly core configuration file")

	// TokenPool field descriptions
	TokenPoolID        = ffm(language.AmericanEnglish, "TokenPool.id", "The UUID of the token pool")
	TokenPoolType      = ffm(language.AmericanEnglish, "TokenPool.type", "The type of token the pool contains, such as fungible/non-fungible")
	TokenPoolNamespace = ffm(language.AmericanEnglish, "TokenPool.namespace", "The namespace for the token pool")
	TokenPoolName      = ffm(language.AmericanEnglish, "TokenPool.name", "The name of the token pool. Note the name is not validated against the description of the token on the blockchain")
	TokenPoolStandard  = ffm(language.AmericanEnglish, "TokenPool.standard", "The ERC standard the token pool conforms to, as reported by the token connector")
	TokenPoolLocator   = ffm(language.AmericanEnglish, "TokenPool.locator", "A unique identifier for the pool, as provided by the token connector")
	TokenPoolKey       = ffm(language.AmericanEnglish, "TokenPool.key", "The signing key used to create the token pool. On input for token connectors that support on-chain deployment of new tokens (vs. only index existing ones) this determines the signing key used to create the token on-chain")
	TokenPoolSymbol    = ffm(language.AmericanEnglish, "TokenPool.symbol", "The token symbol. If supplied on input for an existing on-chain token, this must match the on-chain information")
	TokenPoolDecimals  = ffm(language.AmericanEnglish, "TokenPool.decimals", "Number of decimal places that this token has")
	TokenPoolConnector = ffm(language.AmericanEnglish, "TokenPool.connector", "The name of the token connector, as specified in the FireFly core configuration file that is responsible for the token pool. Required on input when multiple token connectors are configured")
	TokenPoolMessage   = ffm(language.AmericanEnglish, "TokenPool.message", "The UUID of the broadcast message used to inform the network to index this pool")
	TokenPoolState     = ffm(language.AmericanEnglish, "TokenPool.state", "The current state of the token pool")
	TokenPoolCreated   = ffm(language.AmericanEnglish, "TokenPool.created", "The creation time of the pool")
	TokenPoolConfig    = ffm(language.AmericanEnglish, "TokenPool.config", "Input only field, with token connector specific configuration of the pool, such as an existing Ethereum address and block number to used to index the pool. See your chosen token connector documentation for details")
	TokenPoolInfo      = ffm(language.AmericanEnglish, "TokenPool.info", "Token connector specific information about the pool. See your chosen token connector documentation for details")
	TokenPoolTX        = ffm(language.AmericanEnglish, "TokenPool.tx", "Reference to the FireFly transaction used to create and broadcast this pool to the network")

	// TokenTransfer field descriptions
	TokenTransferType            = ffm(language.AmericanEnglish, "TokenTransfer.type", "The type of transfer such as mint/burn/transfer")
	TokenTransferLocalID         = ffm(language.AmericanEnglish, "TokenTransfer.localId", "The UUID of this token transfer, in the local FireFly node")
	TokenTransferPool            = ffm(language.AmericanEnglish, "TokenTransfer.pool", "The UUID the token pool this transfer applies to")
	TokenTransferTokenIndex      = ffm(language.AmericanEnglish, "TokenTransfer.tokenIndex", "The index of the token within the pool that this transfer applies to")
	TokenTransferURI             = ffm(language.AmericanEnglish, "TokenTransfer.uri", "The URI of the token this transfer applies to")
	TokenTransferConnector       = ffm(language.AmericanEnglish, "TokenTransfer.connector", "The name of the token connector, as specified in the FireFly core configuration file. Required on input when there are more than one token connectors configured")
	TokenTransferNamespace       = ffm(language.AmericanEnglish, "TokenTransfer.namespace", "The namespace for the transfer, which must match the namespace of the token pool")
	TokenTransferKey             = ffm(language.AmericanEnglish, "TokenTransfer.key", "The blockchain signing key for the transfer. On input defaults to the first signing key of the organization that operates the node")
	TokenTransferFrom            = ffm(language.AmericanEnglish, "TokenTransfer.from", "The source account for the transfer. On input defaults to the value of 'key'")
	TokenTransferTo              = ffm(language.AmericanEnglish, "TokenTransfer.to", "The target account for the transfer. On input defaults to the value of 'key'")
	TokenTransferAmount          = ffm(language.AmericanEnglish, "TokenTransfer.amount", "The amount for the transfer. For non-fungible tokens will always be 1. For fungible tokens, the number of decimals for the token pool should be considered when inputting the amount. For example, with 18 decimals a fractional balance of 10.234 will be specified as 10,234,000,000,000,000,000")
	TokenTransferProtocolID      = ffm(language.AmericanEnglish, "TokenTransfer.protocolId", "An alphanumerically sortable string that represents this event uniquely with respect to the blockchain")
	TokenTransferMessage         = ffm(language.AmericanEnglish, "TokenTransfer.message", "The UUID of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector")
	TokenTransferMessageHash     = ffm(language.AmericanEnglish, "TokenTransfer.messageHash", "The hash of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector")
	TokenTransferCreated         = ffm(language.AmericanEnglish, "TokenTransfer.created", "The creation time of the transfer")
	TokenTransferTX              = ffm(language.AmericanEnglish, "TokenTransfer.tx", "If submitted via FireFly, this will reference the UUID of the FireFly transaction (if the token connector in use supports attaching data)")
	TokenTransferBlockchainEvent = ffm(language.AmericanEnglish, "TokenTransfer.blockchainEvent", "The UUID of the blockchain event")

	// TokenTransferInput field descriptions
	TokenTransferInputMessage = ffm(language.AmericanEnglish, "TokenTransferInput.message", "You can specify a message to correlate with the transfer, which can be of type broadcast or private. Your chosen token connector and on-chain smart contract must support on-chain/off-chain correlation by taking a `data` input on the transfer")
	TokenTransferInputPool    = ffm(language.AmericanEnglish, "TokenTransferInput.pool", "The name or UUID of a token pool")

	// TransactionStatus field descriptions
	TransactionStatusStatus  = ffm(language.AmericanEnglish, "TransactionStatus.status", "The overall computed status of the transaction, after analyzing the details during the API call")
	TransactionStatusDetails = ffm(language.AmericanEnglish, "TransactionStatus.details", "A set of records describing the activities within the transaction known by the local FireFly node")

	// TransactionStatusDetails field descriptions
	TransactionStatusDetailsType      = ffm(language.AmericanEnglish, "TransactionStatusDetails.type", "The type of the transaction status detail record")
	TransactionStatusDetailsSubType   = ffm(language.AmericanEnglish, "TransactionStatusDetails.subtype", "A sub-type, such as an operation type, or an event type")
	TransactionStatusDetailsStatus    = ffm(language.AmericanEnglish, "TransactionStatusDetails.status", "The status of the detail record. Cases where an event is required for completion, but has not arrived yet are marked with a 'pending' record")
	TransactionStatusDetailsTimestamp = ffm(language.AmericanEnglish, "TransactionStatusDetails.timestamp", "The time relevant to when the record was updated, such as the time an event was created, or the last update time of an operation")
	TransactionStatusDetailsID        = ffm(language.AmericanEnglish, "TransactionStatusDetails.id", "The UUID of the entry referenced by this detail. The type of this record can be inferred from the entry type")
	TransactionStatusDetailsError     = ffm(language.AmericanEnglish, "TransactionStatusDetails.error", "If an error occurred related to the detail entry, it is included here")
	TransactionStatusDetailsInfo      = ffm(language.AmericanEnglish, "TransactionStatusDetails.info", "Output details for this entry")

	// ContractCallRequest field descriptions
	ContractCallRequestType       = ffm(language.AmericanEnglish, "ContractCallRequest.type", "Invocations cause transactions on the blockchain. Whereas queries simply execute logic in your local node to query data at a given current/historical block")
	ContractCallRequestInterface  = ffm(language.AmericanEnglish, "ContractCallRequest.interface", "The UUID of a method within a pre-configured FireFly interface (FFI) definition for a smart contract. Required if the 'method' is omitted. Also see Contract APIs as a way to configure a dedicated API for your FFI, including all methods and an OpenAPI/Swagger interface")
	ContractCallRequestLocation   = ffm(language.AmericanEnglish, "ContractCallRequest.location", "A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel")
	ContractCallRequestKey        = ffm(language.AmericanEnglish, "ContractCallRequest.key", "The blockchain signing key that will sign the invocation. Defaults to the first signing key of the organization that operates the node")
	ContractCallRequestMethod     = ffm(language.AmericanEnglish, "ContractCallRequest.method", "An in-line FFI method definition for the method to invoke. Required when FFI is not specified")
	ContractCallRequestMethodPath = ffm(language.AmericanEnglish, "ContractCallRequest.methodPath", "The pathname of the method on the specified FFI")
	ContractCallRequestInput      = ffm(language.AmericanEnglish, "ContractCallRequest.input", "A map of named inputs. The name and type of each input must be compatible with the FFI description of the method, so that FireFly knows how to serialize it to the blockchain via the connector")

	// WebSocketStatus field descriptions
	WebSocketStatusEnabled     = ffm(language.AmericanEnglish, "WebSocketStatus.enabled", "Indicates whether the websockets plugin is enabled")
	WebSocketStatusConnections = ffm(language.AmericanEnglish, "WebSocketStatus.connections", "List of currently active websocket client connections")

	// WSConnectionStatus field descriptions
	WSConnectionStatusID            = ffm(language.AmericanEnglish, "WSConnectionStatus.id", "The unique ID assigned to this client connection")
	WSConnectionStatusRemoteAddress = ffm(language.AmericanEnglish, "WSConnectionStatus.remoteAddress", "The remote address of the connected client (if available)")
	WSConnectionStatusUserAgent     = ffm(language.AmericanEnglish, "WSConnectionStatus.userAgent", "The user agent of the connected client (if available)")
	WSConnectionStatusSubscriptions = ffm(language.AmericanEnglish, "WSConnectionStatus.subscriptions", "List of subscriptions currently started by this client")

	// WSSubscriptionStatus field descriptions
	WSSubscriptionStatusEphemeral = ffm(language.AmericanEnglish, "WSSubscriptionStatus.ephemeral", "Indicates whether the subscription is ephemeral (vs durable)")
	WSSubscriptionStatusNamespace = ffm(language.AmericanEnglish, "WSSubscriptionStatus.namespace", "The subscription namespace")
	WSSubscriptionStatusName      = ffm(language.AmericanEnglish, "WSSubscriptionStatus.name", "The subscription name (for durable subscriptions only)")
)
