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
	MessageBatchID   = ffm("Message.batch", "The UUID of the batch in which the message was pinned/transferred")
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

	// MemberInput field descriptions
	MemberInputIdentity = ffm("MemberInput.identity", "The DID of the group member. On input can be a UUID or org name, and will be resolved to a DID")
	MemberInputNode     = ffm("MemberInput.node", "The UUID of the node that will receive a copy of the off-chain message for the identity. The first applicable node for the identity will be picked automatically on input if not specified")

	// Member field descriptions
	MemberIdentity = ffm("Member.identity", "The DID of the group member")
	MembertNode    = ffm("Member.node", "The UUID of the node that receives a copy of the off-chain message for the identity")

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

	// DatatypeRef field descriptions
	DatatypeRefName    = ffm("DatatypeRef.name", "The name of the datatype")
	DatatypeRefVersion = ffm("DatatypeRef.version", "The version of the datatype. Semantic versioning is encouraged, such as v1.0.1")

	// Datatype field descriptions
	DatatypeID        = ffm("Datatype.id", "The UUID of the datatype")
	DatatypeMessage   = ffm("Datatype.message", "The UUID of the broadcast message that was used to publish this datatype to the network")
	DatatypeValidator = ffm("Datatype.validator", "The validator that should be used to verify this datatype")
	DatatypeNamespace = ffm("Datatype.namespace", "The namespace of the datatype. Data resources can only be created referencing datatypes in the same namespace")
	DatatypeName      = ffm("Datatype.name", "The name of the datatype")
	DatatypeVersion   = ffm("Datatype.version", "The version of the datatype. Multiple versions can exist with the same name. Use of semantic versioning is encourages, such as v1.0.1")
	DatatypeHash      = ffm("Datatype.hash", "The hash of the value, such as the JSON schema. Allows all parties to be confident they have the exact same rules for verifying data created against a datatype")
	DatatypeCreated   = ffm("Datatype.created", "The time the datatype was created")
	DatatypeValue     = ffm("Datatype.value", "The definition of the datatype, in the syntax supported by the validator. Such as a JSON Schema definition")

	// SignerRef field descriptions
	SignerRefAuthor = ffm("SignerRef.author", "The DID of identity of the submitter")
	SignerRefKey    = ffm("SignerRef.key", "The on-chain signing key used to sign the transaction")

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

	// Transaction field descriptions
	TransactionID            = ffm("Transaction.id", "The UUID of the FireFly transaction")
	TransactionType          = ffm("Transaction.type", "The type of the FireFly transaction")
	TransactionNamespace     = ffm("Transaction.namespace", "The namespace of the FireFly transaction")
	TransactionCreated       = ffm("Transaction.created", "The time the transaction was created on this node. Note the transaction is individually created with the same UUID on each participant in the FireFly transaction")
	TransactionBlockchainIDs = ffm("Transaction.blockchainIds", "The blockchain transaction ID, in the format specific to the blockchain involved in the transaction. Not all FireFly transactions include a blockchain. FireFly transactions are extensible to support multiple blockchain transactions")

	// Operation field description
	OperationID          = ffm("Operation.id", "The UUID of the operation")
	OperationNamespace   = ffm("Operation.namespace", "The namespace of the operation")
	OperationTransaction = ffm("Operation.tx", "The UUID of the FireFly transaction the operation is part of")
	OperationType        = ffm("Operation.type", "The type of the operation")
	OperationStatus      = ffm("Operation.status", "The current status of the operation")
	OperationPlugin      = ffm("Operation.plugin", "The plugin responsible for performing the operation")
	OperationInput       = ffm("Operation.input", "The input to this operation")
	OperationOutput      = ffm("Operation.output", "Any output reported back from the plugin for this operation")
	OperationError       = ffm("Operation.error", "Any error reported back from the plugin for this operation")
	OperationCreated     = ffm("Operation.created", "The time the operation was created")
	OperationUpdated     = ffm("Operation.updated", "The last update time of the operation")
	OperationRetry       = ffm("Operation.retry", "If this operation was initiated as a retry to a previous operation, this field points to the UUID of the operation being retried")

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
	ContractAPILocation  = ffm("ContractAPI.location", "If this API is tied to an individual instance of a smart contract, this field can include a blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel")
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

	// ContractListener field descriptions
	ContractListenerID         = ffm("ContractListener.id", "The UUID of the smart contract listener")
	ContractListenerInterface  = ffm("ContractListener.interface", "A reference to an existing FFI, containing pre-registered type information for the event")
	ContractListenerNamespace  = ffm("ContractListener.namespace", "The namespace of the listener, which defines the namespace of all blockchain events detected by this listener")
	ContractListenerName       = ffm("ContractListener.name", "A descriptive name for the listener")
	ContractListenerProtocolID = ffm("ContractListener.protocolId", "An ID assigned by the blockchain connector to this listener")
	ContractListenerLocation   = ffm("ContractListener.location", "A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel")
	ContractListenerCreated    = ffm("ContractListener.created", "The creation time of the listener")
	ContractListenerEvent      = ffm("ContractListener.event", "The definition of the event, either provided in-line when creating the listener, or extracted from the referenced FFI")
	ContractListenerTopic      = ffm("ContractListener.topic", "A topic to set on the FireFly event that is emitted each time a blockchain event is detected from the blockchain. Setting this topic on a number of listeners allows applications to easily subscribe to all events they need")
	ContractListenerOptions    = ffm("ContractListener.options", "Options that control how the listener subscribes to events from the underlying blockchain")
	ContractListenerEventID    = ffm("ContractListener.eventId", "When creating a listener from an existing FFI, this is the UUID of the event on that FFI that will be detected by this listener")

	// ContractListenerOptions field descriptions
	ContractListenerOptionsFirstEvent = ffm("ContractListenerOptions.firstEvent", "A blockchain specific string, such as a block number, to start listening from. The special strings 'oldest' and 'newest' are supported by all blockchain connectors. Default is 'newest'")

	// DIDDocument field descriptions
	DIDDocumentContext            = ffm("DIDDocument.@context", "See https://www.w3.org/TR/did-core/#json-ld")
	DIDDocumentID                 = ffm("DIDDocument.id", "See https://www.w3.org/TR/did-core/#did-document-properties")
	DIDDocumentAuthentication     = ffm("DIDDocument.authentication", "See https://www.w3.org/TR/did-core/#did-document-properties")
	DIDDocumentVerificationMethod = ffm("DIDDocument.verificationMethod", "See https://www.w3.org/TR/did-core/#did-document-properties")

	// DIDVerificationMethod field descriptions
	DIDVerificationMethodID                  = ffm("DIDVerificationMethod.id", "See https://www.w3.org/TR/did-core/#service-properties")
	DIDVerificationMethodController          = ffm("DIDVerificationMethod.controller", "See https://www.w3.org/TR/did-core/#service-properties")
	DIDVerificationMethodType                = ffm("DIDVerificationMethod.type", "See https://www.w3.org/TR/did-core/#service-properties")
	DIDVerificationMethodBlockchainAccountID = ffm("DIDVerificationMethod.blockchainAcountId", "For blockchains like Ethereum that represent signing identities directly by their public key summarized in an account string")
	DIDVerificationMethodMSPIdentityString   = ffm("DIDVerificationMethod.mspIdentityString", "For Hyperledger Fabric where the signing identity is represented by an MSP identifier (containing X509 certificate DN strings) that were validated by your local MSP")
	DIDVerificationMethodDataExchangePeerID  = ffm("DIDVerificationMethod.dataExchangePeerID", "A string provided by your Data Exchange plugin, that it uses a technology specific mechanism to validate against when messages arrive from this identity")

	// Event field descriptions
	EventID          = ffm("Event.id", "The UUID assigned to this event by your local FireFly node")
	EventSequence    = ffm("Event.sequence", "A sequence indicating the order in which events are delivered to your application. Assure to be unique per event in your local FireFly database (unlike the created timestamp)")
	EventType        = ffm("Event.type", "All interesting activity in FireFly is emitted as a FireFly event, of a given type. The 'type' combined with the 'reference' can be used to determine how to process the event within your application")
	EventNamespace   = ffm("Event.namespace", "The namespace of the event. Your application must subscribe to events within a namespace")
	EventReference   = ffm("Event.reference", "The UUID of an resource that is the subject of this event. The event type determines what type of resource is referenced, and whether this field might be unset")
	EventCorrelator  = ffm("Event.correlator", "For message events, this is the 'header.cid' field from the referenced message. For certain other event types, a secondary object is referenced such as a token pool")
	EventTransaction = ffm("Event.tx", "The UUID of a transaction that is event is part of. Not all events are part of a transaction")
	EventTopic       = ffm("Event.topic", "A stream of information this event relates to. For message confirmation events, a separate event is emitted for each topic in the message. For blockchain events, the listener specifies the topic. Rules exist for how the topic is set for other event types")
	EventCreated     = ffm("Event.created", "The time the event was emitted. Not guaranteed to be unique, or to increase between events in the same order as the final sequence events are delivered to your application. As such, the 'sequence' field should be used instead of the 'created' field for querying events in the exact order they are delivered to applications")

	// EnrichedEvent field descriptions
	EnrichedEventBlockchainEvent   = ffm("EnrichedEvent.blockchainEvent", "A blockchain event if referenced by the FireFly event")
	EnrichedEventContractAPI       = ffm("EnrichedEvent.contractAPI", "A Contract API if referenced by the FireFly event")
	EnrichedEventContractInterface = ffm("EnrichedEvent.contractInterface", "A Contract Interface (FFI) if referenced by the FireFly event")
	EnrichedEventDatatype          = ffm("EnrichedEvent.datatype", "A Datatype if referenced by the FireFly event")
	EnrichedEventIdentity          = ffm("EnrichedEvent.identity", "An Identity if referenced by the FireFly event")
	EnrichedEventMessage           = ffm("EnrichedEvent.message", "A Message if  referenced by the FireFly event")
	EnrichedEventNamespaceDetails  = ffm("EnrichedEvent.namespaceDetails", "Full resource detail of a Namespace if referenced by the FireFly event")
	EnrichedEventTokenApproval     = ffm("EnrichedEvent.tokenApproval", "A Token Approval if referenced by the FireFly event")
	EnrichedEventTokenPool         = ffm("EnrichedEvent.tokenPool", "A Token Pool if referenced by the FireFly event")
	EnrichedEventTokenTransfer     = ffm("EnrichedEvent.tokenTransfer", "A Token Transfer if referenced by the FireFly event")
	EnrichedEventTransaction       = ffm("EnrichedEvent.transaction", "A Transaction if associated with the FireFly event")

	// IdentityMessages field descriptions
	IdentityMessagesClaim        = ffm("IdentityMessages.claim", "The UUID of claim message")
	IdentityMessagesVerification = ffm("IdentityMessages.verification", "The UUID of claim message. Unset for root organization identities")
	IdentityMessagesUpdate       = ffm("IdentityMessages.update", "The UUID of the most recently applied update message. Unset if no updates have been confirmed")

	// Identity field descriptions
	IdentityID        = ffm("Identity.id", "The UUID of the identity")
	IdentityDID       = ffm("Identity.did", "The DID of the identity. Unique across namespaces within a FireFly network")
	IdentityType      = ffm("Identity.type", "The type of the identity")
	IdentityParent    = ffm("Identity.parent", "The UUID of the parent identity. Unset for root organization identities")
	IdentityNamespace = ffm("Identity.namespace", "The namespace of the identity. Organization and node identities are always defined in the ff_system namespace")
	IdentityName      = ffm("Identity.name", "The name of the identity. The name must be unique within the type and namespace")
	IdentityMessages  = ffm("Identity.messages", "References to the broadcast messages that established this identity and proved ownership of the associated verifiers (keys)")
	IdentityCreated   = ffm("Identity.created", "The creation time of the identity")
	IdentityUpdated   = ffm("Identity.updated", "The last update time of the identity profile")

	// IdentityProfile field descriptions
	IdentityProfileProfile     = ffm("IdentityProfile.profile", "A set of metadata for the identity. Part of the updatable profile information of an identity")
	IdentityProfileDescription = ffm("IdentityProfile.description", "A description of the identity. Part of the updatable profile information of an identity")

	// IdentityWithVerifiers field descriptions
	IdentityWithVerifiersVerifiers = ffm("IdentityWithVerifiers.verifiers", "The verifiers, such as blockchain signing keys, that have been bound to this identity and can be used to prove data orignates from that identity")

	// IdentityCreateDTO field descriptions
	IdentityCreateDTOParent = ffm("IdentityCreateDTO.parent", "On input the parent can be specified directly as the UUID of and existing identity, or as a DID to resolve to that identity, or an organization name. The parent must already have been registered, and its blockchain signing key must be available to the local node to sign the verification")
	IdentityCreateDTOKey    = ffm("IdentityCreateDTO.key", "The blockchain signing key to use to make the claim to the identity. Must be available to the local node to sign the identity claim. Will become a verifier on the established identity")

	// IdentityClaim field descriptions
	IdentityClaimIdentity = ffm("IdentityClaim.identity", "The identity being claimed")

	// IdentityVerification field descriptions
	IdentityVerificationClaim    = ffm("IdentityVerification.claim", "The UUID of the message containing the identity claim being verified")
	IdentityVerificationIdentity = ffm("IdentityVerification.identity", "The identity being verified")

	// IdentityUpdate field descriptions
	IdentityUpdateIdentity = ffm("IdentityUpdate.identity", "The identity being updated")
	IdentityUpdateProfile  = ffm("IdentityUpdate.profile", "The new profile, which is replaced in its entirety when the update is confirmed")

	// Verifier field descriptions
	VerifierHash      = ffm("Verifier.hash", "Hash used as a globally consistent identifier for this namespace + type + value combination on every node in the network")
	VerifierIdentity  = ffm("Verifier.identity", "The UUID of the parent identity that has claimed this verifier")
	VerifierType      = ffm("Verifier.type", "The type of the verifier")
	VerifierValue     = ffm("Verifier.value", "The verifier string, such as an Ethereum address, or Fabric MSP identifier")
	VerifierNamespace = ffm("Verifier.namespace", "The namespace of the verifier")
	VerifierCreated   = ffm("Verifier.created", "The time this verifier was created on this node")

	// Namespace field descriptions
	NamespaceID          = ffm("Namespace.id", "The UUID of the namespace. For locally established namespaces will be different on each node in the network. For broadcast namespaces, will be the same on every node")
	NamespaceMessage     = ffm("Namespace.message", "The UUID of broadcast message used to establish the namespace. Unset for local namespaces")
	NamespaceName        = ffm("Namespace.name", "The namespace name")
	NamespaceDescription = ffm("Namespace.description", "A description of the namespace")
	NamespaceType        = ffm("Namespace.type", "The type of the namespace")
	NamespaceCreated     = ffm("Namespace.created", "The time the namespace was created")

	// NodeStatus field descriptions
	NodeStatusNode = ffm("NodeStatus.node", "Details of the local node")
	NodeStatusOrg  = ffm("NodeStatus.org", "Details of the organization identity that operates this node")
	NodeDefaults   = ffm("NodeStatus.defaults", "Information about defaults configured on this node that appplications might need to query on startup")

	// NodeStatusNode field descriptions
	NodeStatusNodeName       = ffm("NodeStatusNode.name", "The name of this node, as specified in the local configuration")
	NodeStatusNodeRegistered = ffm("NodeStatusNode.registered", "Whether the node has been successfully registered")
	NodeStatusNodeID         = ffm("NodeStatusNode.id", "The UUID of the node, if registered")

	// NodeStatusOrg field descriptions
	NodeStatusOrgName       = ffm("NodeStatusOrg.name", "The name of the node operator organization, as specified in the local configuration")
	NodeStatusOrgRegistered = ffm("NodeStatusOrg.registered", "Whether the organization has been successfully registered")
	NodeStatusOrgDID        = ffm("NodeStatusOrg.did", "The DID of the organization identity, if registered")
	NodeStatusOrgID         = ffm("NodeStatusOrg.id", "The UUID of the organization, if registered")
	NodeStatusOrgVerifiers  = ffm("NodeStatusOrg.verifiers", "Array of verifiers (blockchain keys) owned by this identity")

	// NodeStatusDefaults field descriptions
	NodeStatusDefaultsNamespace = ffm("NodeStatusDefaults.namespace", "The default namespace on this node")

	// BatchManagerStatus field descriptions
	BatchManagerStatusProcessors = ffm("BatchManagerStatus.processors", "An array of currently active batch processors")

	// BatchProcessorStatus field descriptions
	BatchProcessorStatusDispatcher = ffm("BatchProcessorStatus.dispatcher", "The type of dispatcher for this processor")
	BatchProcessorStatusName       = ffm("BatchProcessorStatus.name", "The name of the processor, which includes details of the attributes of message are allocated to this processor")
	BatchProcessorStatusStatus     = ffm("BatchProcessorStatus.status", "The flush status for this batch processor")

	// BatchFlushStatus field descriptions
	BatchFlushStatusLastFlushTime        = ffm("BatchFlushStatus.lastFlushStartTime", "The last time a flush was performed")
	BatchFlushStatusFlushing             = ffm("BatchFlushStatus.flushing", "If a flush is in progress, this is the UUID of the batch being flushed")
	BatchFlushStatusBlocked              = ffm("BatchFlushStatus.blocked", "True if the batch flush is in a retry loop, due to errors being returned by the plugins")
	BatchFlushStatusLastFlushError       = ffm("BatchFlushStatus.lastFlushError", "The last error received by this batch processor while flushing")
	BatchFlushStatusLastFlushErrorTime   = ffm("BatchFlushStatus.lastFlushErrorTime", "The time of the last flush")
	BatchFlushStatusAverageBatchBytes    = ffm("BatchFlushStatus.averageBatchBytes", "The average byte size of each batch")
	BatchFlushStatusAverageBatchMessages = ffm("BatchFlushStatus.averageBatchMessages", "The average number of messages included in each batch")
	BatchFlushStatusAverageBatchData     = ffm("BatchFlushStatus.averageBatchData", "The average number of data attachments included in each batch")
	BatchFlushStatusAverageFlushTimeMS   = ffm("BatchFlushStatus.averageFlushTimeMS", "The average amount of time spent flushing each batch")
	BatchFlushStatusTotalBatches         = ffm("BatchFlushStatus.totalBatches", "The total count of batches flushed by this processor since it started")
	BatchFlushStatusTotalErrors          = ffm("BatchFlushStatus.totalErrors", "The total count of error flushed encountered by this processor since it started")

	// Pin field descriptions
	PinSequence   = ffm("Pin.sequence", "The order of the pin in the local FireFly database, which matches the order in which pins were delivered to FireFly by the blockchain connector event stream")
	PinMasked     = ffm("Pin.masked", "True if the pin is for a private message, and hence is masked with the group ID and salted with a nonce so observers of the blockchain cannot use pin hash to match this transaction to other transactions or participants")
	PinHash       = ffm("Pin.hash", "The hash represents a topic within a message in the batch. If a message has multiple topics, then multiple pins are created. If the message is private, the hash is masked for privacy")
	PinBatch      = ffm("Pin.batch", "The UUID of the batch of messages this pin is part of")
	PinBatchHash  = ffm("Pin.batchHash", "The manifest hash batch of messages this pin is part of")
	PinIndex      = ffm("Pin.index", "The index of this pin within the batch. One pin is created for each topic, of each mesage in the batch")
	PinDispatched = ffm("Pin.dispatched", "Once true, this pin as been processed and will not be processed again")
	PinSigner     = ffm("Pin.signer", "The blockchain signing key that submitted this transaction, as passed through to FireFly by the smart contract that emitted the blockchain event")
	PinCreated    = ffm("Pin.created", "The time the FireFly node created the pin")

	// Subscription field descriptions
	SubscriptionID        = ffm("Subscription.id", "The UUID of the subscription")
	SubscriptionNamespace = ffm("Subscription.namespace", "The namespace of the subscription. A subscription will only receive events generated in the namespace of the subscription")
	SubscriptionName      = ffm("Subscription.name", "The name of the subscription. The application specifies this name when it connects, in order to attach to the subscription and receive events that arrived while it was disconnected. If multiple apps connect to the same subscription, events are workload balanced across the connected application instances")
	SubscriptionTransport = ffm("Subscription.transport", "The transport plugin responsible for event delivery (WebSockets, Webhooks, JMS, NATS etc.)")
	SubscriptionFilter    = ffm("Subscription.filter", "Server-side filter to apply to events")
	SubscriptionOptions   = ffm("Subscription.options", "Subscription options")
	SubscriptionEphemeral = ffm("Subscription.ephemeral", "Ephemeral subscriptions only exist as long as the application is connected, and as such will miss events that occur while the application is disconnected, and cannot be created administratively. You can create one over over a connected WebSocket connection")
	SubscriptionCreated   = ffm("Subscription.created", "Creation time of the subscription")
	SubscriptionUpdated   = ffm("Subscription.updated", "Last time the subscription was updated")

	// SubscriptionFilter field descriptions
	SubscriptionFilterEvents           = ffm("SubscriptionFilter.events", "Regular expression to apply to the event type, to subscribe to a subset of event types")
	SubscriptionFilterTopic            = ffm("SubscriptionFilter.topic", "Regular expression to apply to the topic of the event, to subscribe to a subset of topics. Note for messages sent with multiple topics, a separate event is emitted for each topic")
	SubscriptionFilterMessage          = ffm("SubscriptionFilter.message", "Filters specific to message events. If an event is not a message event, these filters are ignored")
	SubscriptionFilterTransaction      = ffm("SubscriptionFilter.transaction", "Filters specific to events with a transaction. If an event is not associated with a transaction, this filter is ignored")
	SubscriptionFilterBlockchainEvent  = ffm("SubscriptionFilter.blockchainevent", "Filters specific to blockchain events. If an event is not a blockchain event, these filters are ignored")
	SubscriptionFilterDeprecatedTopics = ffm("SubscriptionFilter.topics", "Deprecated: Please use 'topic' instead")
	SubscriptionFilterDeprecatedTag    = ffm("SubscriptionFilter.tag", "Deprecated: Please use 'message.tag' instead")
	SubscriptionFilterDeprecatedGroup  = ffm("SubscriptionFilter.group", "Deprecated: Please use 'message.group' instead")
	SubscriptionFilterDeprecatedAuthor = ffm("SubscriptionFilter.author", "Deprecated: Please use 'message.author' instead")

	// SubscriptionMessageFilter field descriptions
	SubscriptionMessageFilterTag    = ffm("SubscriptionMessageFilter.tag", "Regular expression to apply to the message 'header.tag' field")
	SubscriptionMessageFilterGroup  = ffm("SubscriptionMessageFilter.group", "Regular expression to apply to the message 'header.group' field")
	SubscriptionMessageFilterAuthor = ffm("SubscriptionMessageFilter.author", "Regular expression to apply to the message 'header.author' field")

	// SubscriptionTransactionFilter field descriptions
	SubscriptionTransactionFilterType = ffm("SubscriptionTransactionFilter.type", "Regular expression to apply to the transaction 'type' field")

	// SubscriptionBlockchainEventFilter field descriptions
	SubscriptionBlockchainEventFilterName     = ffm("SubscriptionBlockchainEventFilter.name", "Regular expression to apply to the blockchain event 'name' field, which is the name of the event in the underlying blockchain smart contract")
	SubscriptionBlockchainEventFilterListener = ffm("SubscriptionBlockchainEventFilter.listener", "Regular expression to apply to the blockchain event 'listener' field, which is the UUID of the event listener. So you can restrict your subscription to certain blockchain listeners. Alternatively to avoid your application need to know listener UUIDs you can set the 'topic' field of blockchain event listeners, and use a topic filter on your subscriptions")

	// SubscriptionCoreOptions field descriptions
	SubscriptionCoreOptionsFirstEvent = ffm("SubscriptionCoreOptions.firstEvent", "Whether your appplication would like to receive events from the 'earliest' event emitted by your FireFly node (from the beginning of time), the 'latest' event (from now), or a specific event sequence. Default is 'latest'")
	SubscriptionCoreOptionsReadAhead  = ffm("SubscriptionCoreOptions.readAhead", "The number of events to stream ahead to your application, while waiting for confirmation of consumption of those events. At least once delivery semantics are used in FireFly, so if your application crashes/reconnects this is the maximum number of events you would expect to be redelivered after it restarts")
	SubscriptionCoreOptionsWithData   = ffm("SubscriptionCoreOptions.withData", "Whether message events delivered over the subscription, should be packaged with the full data of those messages in-line as part of the event JSON payload. Or if the application should make separate REST calls to download that data")

	// TokenApproval field descriptions
	TokenApprovalLocalID         = ffm("TokenApproval.localId", "The UUID of this token approval, in the local FireFly node")
	TokenApprovalPool            = ffm("TokenApproval.pool", "The UUID the token pool this approval applies to")
	TokenApprovalTokenIndex      = ffm("TokenApproval.tokenIndex", "The index of the token within the pool that this approval applies to")
	TokenApprovalConnector       = ffm("TokenApproval.connector", "The token connector. Required on input when there are more than one token connectors configured")
	TokenApprovalKey             = ffm("TokenApproval.key", "The blockchain signing key for the approval request. On input defaults to the first signing key of the organization that operates the node")
	TokenApprovalOperator        = ffm("TokenApproval.operator", "The blockchain identity that is granted the approval")
	TokenApprovalApproved        = ffm("TokenApproval.approved", "Whether this record grants permission for an operator to perform actions on the token balance (true), or revokes permission (false)")
	TokenApprovalInfo            = ffm("TokenApproval.info", "Token connector specific information about the approval operation, such as whether it applied to a limited balance of a fungible token. See your chosen token connector documentation for details")
	TokenApprovalNamespace       = ffm("TokenApproval.namespace", "The namespace for the approval, which must match the namespace of the token pool")
	TokenApprovalProtocolID      = ffm("TokenApproval.protocolId", "The protocol identifier of the blockchain event for the token approval. See the 'protocolId' field on blockchain events for more information")
	TokenApprovalCreated         = ffm("TokenApproval.created", "The creation time of the token approval")
	TokenApprovalTX              = ffm("TokenApproval.tx", "If submitted via FireFly through the local node, this will reference the UUID of the FireFly transaction")
	TokenApprovalBlockchainEvent = ffm("TokenApproval.blockchainEvent", "The UUID of the blockchain event")
	TokenApprovalConfig          = ffm("TokenApproval.config", "Input only field, with token connector specific configuration of the approval.  See your chosen token connector documentation for details")

	// TokenBalance field descriptions
	TokenBalancePool       = ffm("TokenBalance.pool", "The UUID the token pool this balance entry applies to")
	TokenBalanceTokenIndex = ffm("TokenBalance.tokenIndex", "The index of the token within the pool that this balance applies to")
	TokenBalanceURI        = ffm("TokenBalance.uri", "The URI of the token this balance entry applies to")
	TokenBalanceConnector  = ffm("TokenBalance.connector", "The token connector that is responsible for the token pool of this balance entry")
	TokenBalanceNamespace  = ffm("TokenBalance.namespace", "The namespace of the token pool for this balance entry")
	TokenBalanceKey        = ffm("TokenBalance.key", "The blockchain signing identity this balance applies to")
	TokenBalanceBalance    = ffm("TokenBalance.balance", "The numeric balance. For non-fungible tokens will always be 1. For fungible tokens, the number of decimals for the token pool should be considered when interpreting the balance. For example, with 18 decimals a fractional balance of 10.234 will be returned as 10,234,000,000,000,000,000")
	TokenBalanceUpdated    = ffm("TokenBalance.updated", "The last time the balance was updated by applying a transfer event")

	// TokenBalance field descriptions
	TokenConnectorName = ffm("TokenConnector.name", "The name of the token connector, as configured in the FireFly core configuration file")

	// TokenPool field descriptions
	TokenPoolID         = ffm("TokenPool.id", "The UUID of the token pool")
	TokenPoolType       = ffm("TokenPool.type", "The type of token the pool contains, such as fungible/non-fungible")
	TokenPoolNamespace  = ffm("TokenPool.namespace", "The namespace for the token pool")
	TokenPoolName       = ffm("TokenPool.name", "The name of the token pool. Note the name is not validated against the description of the token on the blockchain")
	TokenPoolStandard   = ffm("TokenPool.standard", "The ERC standard the token pool conforms to, as reported by the token connector")
	TokenPoolProtocolID = ffm("TokenPool.protocolId", "The protocol identifier of the blockchain event that established the existence of the pool on-chain. See the 'protocolId' field on blockchain events for more information")
	TokenPoolKey        = ffm("TokenPool.key", "The signing key used to create the token pool. On input for token connectors that support on-chain deployment of new tokens (vs. only index existing ones) this determines the signing key used to create the token on-chain")
	TokenPoolSymbol     = ffm("TokenPool.symbol", "The token symbol. If supplied on input for an existing on-chain token, this must match the on-chain information")
	TokenPoolConnector  = ffm("TokenPool.connector", "The token connector that is responsible for the token pool. Required on input when multiple token connectors are configured")
	TokenPoolMessage    = ffm("TokenPool.message", "The UUID of the broadcast message used to inform the network to index this pool")
	TokenPoolState      = ffm("TokenPool.state", "The current state of the token pool")
	TokenPoolCreated    = ffm("TokenPool.created", "The creation time of the pool")
	TokenPoolConfig     = ffm("TokenPool.config", "Input only field, with token connector specific configuration of the pool, such as an existing Ethereum address and block number to used to index the pool. See your chosen token connector documentation for details")
	TokenPoolInfo       = ffm("TokenPool.info", "Token connector specific information about the pool. See your chosen token connector documentation for details")
	TokenPoolTX         = ffm("TokenPool.tx", "Reference to the FireFly transaction used to create and broadcast this pool to the network")

	// TokenTransfer field descriptions
	TokenTransferType            = ffm("TokenTransfer.type", "The type of transfer such as mint/burn/transfer")
	TokenTransferLocalID         = ffm("TokenTransfer.localId", "The UUID of this token transfer, in the local FireFly node")
	TokenTransferPool            = ffm("TokenTransfer.pool", "The UUID the token pool this transfer applies to")
	TokenTransferTokenIndex      = ffm("TokenTransfer.tokenIndex", "The index of the token within the pool that this transfer applies to")
	TokenTransferURI             = ffm("TokenTransfer.uri", "The URI of the token this transfer applies to")
	TokenTransferConnector       = ffm("TokenTransfer.connector", "The token connector. Required on input when there are more than one token connectors configured")
	TokenTransferNamespace       = ffm("TokenTransfer.namespace", "The namespace for the transfer, which must match the namespace of the token pool")
	TokenTransferKey             = ffm("TokenTransfer.key", "The blockchain signing key for the transfer. On input defaults to the first signing key of the organization that operates the node")
	TokenTransferFrom            = ffm("TokenTransfer.from", "The source account for the transfer. On input defaults to the value of 'key'")
	TokenTransferTo              = ffm("TokenTransfer.to", "The target account for the transfer. Required on input")
	TokenTransferAmount          = ffm("TokenTransfer.amount", "The amount for the transfer. For non-fungible tokens will always be 1. For fungible tokens, the number of decimals for the token pool should be considered when inputting the amount. For example, with 18 decimals a fractional balance of 10.234 will be specified as 10,234,000,000,000,000,000")
	TokenTransferProtocolID      = ffm("TokenTransfer.protocolId", "The protocol identifier of the blockchain event for the token transfer. See the 'protocolId' field on blockchain events for more information")
	TokenTransferMessage         = ffm("TokenTransfer.message", "The UUID of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector")
	TokenTransferMessageHash     = ffm("TokenTransfer.messageHash", "The hash of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector")
	TokenTransferCreated         = ffm("TokenTransfer.created", "The creation time of the transfer")
	TokenTransferTX              = ffm("TokenTransfer.tx", "If submitted via FireFly through the local node, this will reference the UUID of the FireFly transaction")
	TokenTransferBlockchainEvent = ffm("TokenTransfer.blockchainEvent", "The UUID of the blockchain event")

	// TokenTransferInput field descriptions
	TokenTransferInputMessage = ffm("TokenTransferInput.message", "You can specify a message to correlate with the transfer, which can be of type broadcast or private. Your chosen token connector and on-chain smart contract must support on-chain/off-chain correlation by taking a `data` input on the transfer")
	TokenTransferInputPool    = ffm("TokenTransferInput.pool", "The name or UUID of a token pool")

	// TransactionStatus field descriptions
	TransactionStatusStatus  = ffm("TransactionStatus.status", "The overall computed status of the transaction, after analyzing the details during the API call")
	TransactionStatusDetails = ffm("TransactionStatus.details", "A set of records describing the activities within the transaction known by the local FireFly node")

	// TransactionStatusDetails field descriptions
	TransactionStatusDetailsType      = ffm("TransactionStatusDetails.type", "The type of the transaction status detail record")
	TransactionStatusDetailsSubType   = ffm("TransactionStatusDetails.subtype", "A sub-type, such as an operation type, or an event type")
	TransactionStatusDetailsStatus    = ffm("TransactionStatusDetails.status", "The status of the detail record. Cases where an event is required for completion, but has not arrived yet are marked with a 'pending' record")
	TransactionStatusDetailsTimestamp = ffm("TransactionStatusDetails.timestamp", "The time relevant to when the record was updated, such as the time an event was created, or the last update time of an operation")
	TransactionStatusDetailsID        = ffm("TransactionStatusDetails.id", "The UUID of the entry referenced by this detail. The type of this record can be inferred from the entry type")
	TransactionStatusDetailsError     = ffm("TransactionStatusDetails.error", "If an error occurred related to the detail entry, it is included here")
	TransactionStatusDetailsInfo      = ffm("TransactionStatusDetails.info", "Input data for operations, and output data for events")

	// ContractCallRequest field descriptions
	ContractCallRequestType      = ffm("ContractCallRequest.type", "Invocations cause transactions on the blockchain. Whereas queries simply execute logic in your local node to query data at a given current/historical block")
	ContractCallRequestInterface = ffm("ContractCallRequest.interface", "The UUID of a method within a pre-configured FireFly interface (FFI) definition for a smart contract. Required if the 'method' is omitted. Also see Contract APIs as a way to configure a dedicated API for your FFI, including all methods and an OpenAPI/Swagger interface")
	ContractCallRequestLocation  = ffm("ContractCallRequest.location", "A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel")
	ContractCallRequestKey       = ffm("ContractCallRequest.key", "The blockchain signing key that will sign the invocation. Defaults to the first signing key of the organization that operates the node")
	ContractCallRequestMethod    = ffm("ContractCallRequest.method", "An in-line FFI method definition for the method to invoke. Required when FFI is not specified")
)
