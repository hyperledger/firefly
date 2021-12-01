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

package database

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	// HashMismatch sentinel error
	HashMismatch = i18n.NewError(context.Background(), i18n.MsgHashMismatch)
	// IDMismatch sentinel error
	IDMismatch = i18n.NewError(context.Background(), i18n.MsgIDMismatch)
	// DeleteRecordNotFound sentinel error
	DeleteRecordNotFound = i18n.NewError(context.Background(), i18n.Msg404NotFound)
)

type UpsertOptimization int

const (
	UpsertOptimizationSkip UpsertOptimization = iota
	UpsertOptimizationNew
	UpsertOptimizationExisting
)

// Plugin is the interface implemented by each plugin
type Plugin interface {
	PeristenceInterface // Split out to aid pluggability the next level down (SQL provider etc.)

	// InitPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitPrefix(prefix config.Prefix)

	// Init initializes the plugin, with configuration
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.Prefix, callbacks Callbacks) error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities
}

type iNamespaceCollection interface {
	// UpsertNamespace - Upsert a namespace
	// Throws IDMismatch error if updating and ids don't match
	UpsertNamespace(ctx context.Context, data *fftypes.Namespace, allowExisting bool) (err error)

	// DeleteNamespace - Delete namespace
	DeleteNamespace(ctx context.Context, id *fftypes.UUID) (err error)

	// GetNamespace - Get an namespace by name
	GetNamespace(ctx context.Context, name string) (offset *fftypes.Namespace, err error)

	// GetNamespaces - Get namespaces
	GetNamespaces(ctx context.Context, filter Filter) (offset []*fftypes.Namespace, res *FilterResult, err error)
}

type iMessageCollection interface {
	// UpsertMessage - Upsert a message, with all the embedded data references.
	//                 The database layer must ensure that if a record already exists, the hash of that existing record
	//                 must match the hash of the record that is being inserted.
	UpsertMessage(ctx context.Context, message *fftypes.Message, optimization UpsertOptimization) (err error)

	// UpdateMessage - Update message
	UpdateMessage(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// UpdateMessages - Update messages
	UpdateMessages(ctx context.Context, filter Filter, update Update) (err error)

	// GetMessageByID - Get a message by ID
	GetMessageByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Message, err error)

	// GetMessages - List messages, reverse sorted (newest first) by Confirmed then Created, with pagination, and simple must filters
	GetMessages(ctx context.Context, filter Filter) (message []*fftypes.Message, res *FilterResult, err error)

	// GetMessageRefs - Lighter weight query to just get the reference info of messages
	GetMessageRefs(ctx context.Context, filter Filter) ([]*fftypes.MessageRef, *FilterResult, error)

	// GetMessagesForData - List messages where there is a data reference to the specified ID
	GetMessagesForData(ctx context.Context, dataID *fftypes.UUID, filter Filter) (message []*fftypes.Message, res *FilterResult, err error)
}

type iDataCollection interface {
	// UpsertData - Upsert a data record. A hint can be supplied to whether the data already exists.
	//              The database layer must ensure that if a record already exists, the hash of that existing record
	//              must match the hash of the record that is being inserted.
	UpsertData(ctx context.Context, data *fftypes.Data, optimization UpsertOptimization) (err error)

	// UpdateData - Update data
	UpdateData(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetDataByID - Get a data record by ID
	GetDataByID(ctx context.Context, id *fftypes.UUID, withValue bool) (message *fftypes.Data, err error)

	// GetData - Get data
	GetData(ctx context.Context, filter Filter) (message []*fftypes.Data, res *FilterResult, err error)

	// GetDataRefs - Get data references only (no data)
	GetDataRefs(ctx context.Context, filter Filter) (message fftypes.DataRefs, res *FilterResult, err error)
}

type iBatchCollection interface {
	// UpsertBatch - Upsert a batch
	// allowHashUpdate=false throws HashMismatch error if the updated message has a different hash
	UpsertBatch(ctx context.Context, data *fftypes.Batch, allowHashUpdate bool) (err error)

	// UpdateBatch - Update data
	UpdateBatch(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetBatchByID - Get a batch by ID
	GetBatchByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Batch, err error)

	// GetBatches - Get batches
	GetBatches(ctx context.Context, filter Filter) (message []*fftypes.Batch, res *FilterResult, err error)
}

type iTransactionCollection interface {
	// UpsertTransaction - Upsert a transaction
	// allowHashUpdate=false throws HashMismatch error if the updated message has a different hash
	UpsertTransaction(ctx context.Context, data *fftypes.Transaction, allowHashUpdate bool) (err error)

	// UpdateTransaction - Update transaction
	UpdateTransaction(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetTransactionByID - Get a transaction by ID
	GetTransactionByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Transaction, err error)

	// GetTransactions - Get transactions
	GetTransactions(ctx context.Context, filter Filter) (message []*fftypes.Transaction, res *FilterResult, err error)
}

type iDatatypeCollection interface {
	// UpsertDatatype - Upsert a data definitino
	UpsertDatatype(ctx context.Context, datadef *fftypes.Datatype, allowExisting bool) (err error)

	// UpdateDatatype - Update data definition
	UpdateDatatype(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetDatatypeByID - Get a data definition by ID
	GetDatatypeByID(ctx context.Context, id *fftypes.UUID) (datadef *fftypes.Datatype, err error)

	// GetDatatypeByName - Get a data definition by name
	GetDatatypeByName(ctx context.Context, ns, name, version string) (datadef *fftypes.Datatype, err error)

	// GetDatatypes - Get data definitions
	GetDatatypes(ctx context.Context, filter Filter) (datadef []*fftypes.Datatype, res *FilterResult, err error)
}

type iOffsetCollection interface {
	// UpsertOffset - Upsert an offset
	UpsertOffset(ctx context.Context, data *fftypes.Offset, allowExisting bool) (err error)

	// UpdateOffset - Update offset
	UpdateOffset(ctx context.Context, rowID int64, update Update) (err error)

	// GetOffset - Get an offset by name
	GetOffset(ctx context.Context, t fftypes.OffsetType, name string) (offset *fftypes.Offset, err error)

	// GetOffsets - Get offsets
	GetOffsets(ctx context.Context, filter Filter) (offset []*fftypes.Offset, res *FilterResult, err error)

	// DeleteOffset - Delete an offset by name
	DeleteOffset(ctx context.Context, t fftypes.OffsetType, name string) (err error)
}

type iPinCollection interface {
	// UpsertPin - Will insert a pin at the end of the sequence, unless the batch+hash+index sequence already exists
	UpsertPin(ctx context.Context, parked *fftypes.Pin) (err error)

	// GetPins - Get pins
	GetPins(ctx context.Context, filter Filter) (offset []*fftypes.Pin, res *FilterResult, err error)

	// SetPinDispatched - Set the dispatched flag to true on the specified pins
	SetPinDispatched(ctx context.Context, sequence int64) (err error)

	// DeletePin - Delete a pin
	DeletePin(ctx context.Context, sequence int64) (err error)
}

type iOperationCollection interface {
	// UpsertOperation - Upsert an operation
	UpsertOperation(ctx context.Context, operation *fftypes.Operation, allowExisting bool) (err error)

	// UpdateOperation - Update operation by ID
	UpdateOperation(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetOperationByID - Get an operation by ID
	GetOperationByID(ctx context.Context, id *fftypes.UUID) (operation *fftypes.Operation, err error)

	// GetOperations - Get operation
	GetOperations(ctx context.Context, filter Filter) (operation []*fftypes.Operation, res *FilterResult, err error)
}

type iSubscriptionCollection interface {
	// UpsertSubscription - Upsert a subscription
	UpsertSubscription(ctx context.Context, data *fftypes.Subscription, allowExisting bool) (err error)

	// UpdateSubscription - Update subscription
	// Throws IDMismatch error if updating and ids don't match
	UpdateSubscription(ctx context.Context, ns, name string, update Update) (err error)

	// GetSubscriptionByName - Get an subscription by name
	GetSubscriptionByName(ctx context.Context, ns, name string) (offset *fftypes.Subscription, err error)

	// GetSubscriptionByID - Get an subscription by id
	GetSubscriptionByID(ctx context.Context, id *fftypes.UUID) (offset *fftypes.Subscription, err error)

	// GetSubscriptions - Get subscriptions
	GetSubscriptions(ctx context.Context, filter Filter) (offset []*fftypes.Subscription, res *FilterResult, err error)

	// DeleteSubscriptionByID - Delete a subscription
	DeleteSubscriptionByID(ctx context.Context, id *fftypes.UUID) (err error)
}

type iEventCollection interface {
	// InsertEvent - Insert an event
	InsertEvent(ctx context.Context, data *fftypes.Event) (err error)

	// UpdateEvent - Update event
	UpdateEvent(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetEventByID - Get a event by ID
	GetEventByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Event, err error)

	// GetEvents - Get events
	GetEvents(ctx context.Context, filter Filter) (message []*fftypes.Event, res *FilterResult, err error)
}

type iOrganizationsCollection interface {
	// UpsertOrganization - Upsert an organization
	UpsertOrganization(ctx context.Context, data *fftypes.Organization, allowExisting bool) (err error)

	// UpdateOrganization - Update organization
	UpdateOrganization(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetOrganizationByIdentity - Get a organization by identity
	GetOrganizationByIdentity(ctx context.Context, identity string) (org *fftypes.Organization, err error)

	// GetOrganizationByName - Get a organization by name
	GetOrganizationByName(ctx context.Context, name string) (org *fftypes.Organization, err error)

	// GetOrganizationByID - Get a organization by ID
	GetOrganizationByID(ctx context.Context, id *fftypes.UUID) (org *fftypes.Organization, err error)

	// GetOrganizations - Get organizations
	GetOrganizations(ctx context.Context, filter Filter) (org []*fftypes.Organization, res *FilterResult, err error)
}

type iNodeCollection interface {
	// UpsertNode - Upsert a node
	UpsertNode(ctx context.Context, data *fftypes.Node, allowExisting bool) (err error)

	// UpdateNode - Update node
	UpdateNode(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetNode - Get a node by ID
	GetNode(ctx context.Context, owner, name string) (node *fftypes.Node, err error)

	// GetNodeByID- Get a node by ID
	GetNodeByID(ctx context.Context, id *fftypes.UUID) (node *fftypes.Node, err error)

	// GetNodes - Get nodes
	GetNodes(ctx context.Context, filter Filter) (node []*fftypes.Node, res *FilterResult, err error)
}

type iGroupCollection interface {
	// UpserGroup - Upsert a group
	UpsertGroup(ctx context.Context, data *fftypes.Group, allowExisting bool) (err error)

	// UpdateGroup - Update group
	UpdateGroup(ctx context.Context, hash *fftypes.Bytes32, update Update) (err error)

	// GetGroupByHash - Get a group by ID
	GetGroupByHash(ctx context.Context, hash *fftypes.Bytes32) (node *fftypes.Group, err error)

	// GetGroups - Get groups
	GetGroups(ctx context.Context, filter Filter) (node []*fftypes.Group, res *FilterResult, err error)
}

type iNonceCollection interface {
	// UpsertNonceNext - Upsert a context, assigning zero if not found, or the next nonce if it is
	UpsertNonceNext(ctx context.Context, context *fftypes.Nonce) (err error)

	// GetNonce - Get a context by hash
	GetNonce(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.Nonce, err error)

	// GetNonces - Get contexts
	GetNonces(ctx context.Context, filter Filter) (node []*fftypes.Nonce, res *FilterResult, err error)

	// DeleteNonce - Delete context by hash
	DeleteNonce(ctx context.Context, hash *fftypes.Bytes32) (err error)
}

type iNextPinCollection interface {
	// InsertNextPin - insert a nextpin
	InsertNextPin(ctx context.Context, nextpin *fftypes.NextPin) (err error)

	// GetNextPinByContextAndIdentity - lookup nextpin by context+identity
	GetNextPinByContextAndIdentity(ctx context.Context, context *fftypes.Bytes32, identity string) (message *fftypes.NextPin, err error)

	// GetNextPinByHash - lookup nextpin by its hash
	GetNextPinByHash(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.NextPin, err error)

	// GetNextPins - get nextpins
	GetNextPins(ctx context.Context, filter Filter) (message []*fftypes.NextPin, res *FilterResult, err error)

	// UpdateNextPin - update a next hash using its local database ID
	UpdateNextPin(ctx context.Context, sequence int64, update Update) (err error)

	// DeleteNextPin - delete a next hash, using its local database ID
	DeleteNextPin(ctx context.Context, sequence int64) (err error)
}

type iBlobCollection interface {
	// InsertBlob - insert a blob
	InsertBlob(ctx context.Context, blob *fftypes.Blob) (err error)

	// GetBlobMatchingHash - lookup first blob batching a hash
	GetBlobMatchingHash(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.Blob, err error)

	// GetBlobs - get blobs
	GetBlobs(ctx context.Context, filter Filter) (message []*fftypes.Blob, res *FilterResult, err error)

	// DeleteBlob - delete a blob, using its local database ID
	DeleteBlob(ctx context.Context, sequence int64) (err error)
}

type iConfigRecordCollection interface {
	// UpsertConfigRecord - Upsert a config record
	// Throws IDMismatch error if updating and ids don't match
	UpsertConfigRecord(ctx context.Context, data *fftypes.ConfigRecord, allowExisting bool) (err error)

	// GetConfigRecord - Get an config record by key
	GetConfigRecord(ctx context.Context, key string) (offset *fftypes.ConfigRecord, err error)

	// GetConfigRecords - Get config records
	GetConfigRecords(ctx context.Context, filter Filter) (offset []*fftypes.ConfigRecord, res *FilterResult, err error)

	// DeleteConfigRecord - Delete config record
	DeleteConfigRecord(ctx context.Context, key string) (err error)
}

type iTokenPoolCollection interface {
	// UpsertTokenPool - Upsert a token pool
	UpsertTokenPool(ctx context.Context, pool *fftypes.TokenPool) error

	// GetTokenPool - Get a token pool by name
	GetTokenPool(ctx context.Context, ns, name string) (*fftypes.TokenPool, error)

	// GetTokenPoolByID - Get a token pool by pool ID
	GetTokenPoolByID(ctx context.Context, id *fftypes.UUID) (*fftypes.TokenPool, error)

	// GetTokenPoolByID - Get a token pool by protocol ID
	GetTokenPoolByProtocolID(ctx context.Context, connector, protocolID string) (*fftypes.TokenPool, error)

	// GetTokenPools - Get token pools
	GetTokenPools(ctx context.Context, filter Filter) ([]*fftypes.TokenPool, *FilterResult, error)
}

type iTokenBalanceCollection interface {
	// UpdateTokenBalances - Move some token balance from one account to another
	UpdateTokenBalances(ctx context.Context, transfer *fftypes.TokenTransfer) error

	// GetTokenBalance - Get a token balance by pool and account identity
	GetTokenBalance(ctx context.Context, poolID *fftypes.UUID, tokenIndex, identity string) (*fftypes.TokenBalance, error)

	// GetTokenBalances - Get token balances
	GetTokenBalances(ctx context.Context, filter Filter) ([]*fftypes.TokenBalance, *FilterResult, error)

	// GetTokenAccounts - Get token accounts (all distinct addresses that have a balance)
	GetTokenAccounts(ctx context.Context, filter Filter) ([]*fftypes.TokenAccount, *FilterResult, error)

	// GetTokenAccountPools - Get the list of pools referenced by a given account
	GetTokenAccountPools(ctx context.Context, key string, filter Filter) ([]*fftypes.TokenAccountPool, *FilterResult, error)
}

type iTokenTransferCollection interface {
	// UpsertTokenTransfer - Upsert a token transfer
	UpsertTokenTransfer(ctx context.Context, transfer *fftypes.TokenTransfer) error

	// GetTokenTransfer - Get a token transfer by ID
	GetTokenTransfer(ctx context.Context, localID *fftypes.UUID) (*fftypes.TokenTransfer, error)

	// GetTokenTransferByProtocolID - Get a token transfer by protocol ID
	GetTokenTransferByProtocolID(ctx context.Context, connector, protocolID string) (*fftypes.TokenTransfer, error)

	// GetTokenTransfers - Get token transfers
	GetTokenTransfers(ctx context.Context, filter Filter) ([]*fftypes.TokenTransfer, *FilterResult, error)
}

type iContractDefinitionCollection interface {
	// InsertContractInterface - inserts a new contract definition. Must be unique for namespace, name, version
	InsertContractInterface(ctx context.Context, cd *fftypes.FFI) error
	GetContractInterfaces(ctx context.Context, ns string, filter Filter) ([]*fftypes.FFI, *FilterResult, error)
	GetContractInterfaceByID(ctx context.Context, id string) (*fftypes.FFI, error)
	GetContractInterfaceByNameAndVersion(ctx context.Context, ns, name, version string) (*fftypes.FFI, error)
}

type iContractMethodCollection interface {
	UpsertContractMethod(ctx context.Context, ns string, contractID *fftypes.UUID, method *fftypes.FFIMethod) error
	GetContractMethodByName(ctx context.Context, ns string, contractID *fftypes.UUID, name string) (*fftypes.FFIMethod, error)
	GetContractMethods(ctx context.Context, filter Filter) (methods []*fftypes.FFIMethod, res *FilterResult, err error)
}

type iContractEventCollection interface {
	UpsertContractEvent(ctx context.Context, ns string, contractID *fftypes.UUID, method *fftypes.FFIEvent) error
	GetContractEventByName(ctx context.Context, ns string, contractID *fftypes.UUID, name string) (*fftypes.FFIEvent, error)
	GetContractEvents(ctx context.Context, filter Filter) (events []*fftypes.FFIEvent, res *FilterResult, err error)
}

type iContractAPICollection interface {
	InsertContractAPI(ctx context.Context, cd *fftypes.ContractAPI) error
	GetContractAPIs(ctx context.Context, ns string, filter Filter) ([]*fftypes.ContractAPI, *FilterResult, error)
	GetContractAPIByID(ctx context.Context, id string) (*fftypes.ContractAPI, error)
	GetContractAPIByName(ctx context.Context, ns, name string) (*fftypes.ContractAPI, error)
}

// PersistenceInterface are the operations that must be implemented by a database interface plugin.
// The database mechanism of Firefly is designed to provide the balance between being able
// to query the data a member of the network has transferred/received via Firefly efficiently,
// while not trying to become the core database of the application (where full deeply nested
// rich query is needed).
//
// This means that we treat business data as opaque within the storage, only verifying it against
// a data definition within the Firefly core runtime itself.
// The data types, indexes and relationships are designed to be simple, and map closely to the
// REST semantics of the Firefly API itself.
//
// As a result, the database interface could be implemented efficiently by most database technologies.
// Including both Relational/SQL and Document/NoSQL database technologies.
//
// As such we suggest the factors in choosing your database should be non-functional, such as:
// - Which provides you with the HA/DR capabilities you require
// - Which is most familiar within your existing devops pipeline for the application
// - Whether you can consolidate the HA/DR and server infrastructure for your app DB with the Firefly DB
//
// Each database does need an update to the core codebase, to provide a plugin that implements this
// interface.
// For SQL databases the process of adding a new database is simplified via the common SQL layer.
// For NoSQL databases, the code should be straight forward to map the collections, indexes, and operations.
//
type PeristenceInterface interface {
	fftypes.Named

	// RunAsGroup instructs the database plugin that all database operations performed within the context
	// function can be grouped into a single transaction (if supported).
	// Requirements:
	// - Firefly must not depend on this to guarantee ACID properties (it is only a suggestion/optimization)
	// - The database implementation must support nested RunAsGroup calls (ie by reusing a transaction if one exists)
	// - The caller is responsible for passing the supplied context to all database operations within the callback function
	RunAsGroup(ctx context.Context, fn func(ctx context.Context) error) error

	iNamespaceCollection
	iMessageCollection
	iDataCollection
	iBatchCollection
	iTransactionCollection
	iDatatypeCollection
	iOffsetCollection
	iPinCollection
	iOperationCollection
	iSubscriptionCollection
	iEventCollection
	iOrganizationsCollection
	iNodeCollection
	iGroupCollection
	iNonceCollection
	iNextPinCollection
	iBlobCollection
	iConfigRecordCollection
	iTokenPoolCollection
	iTokenBalanceCollection
	iTokenTransferCollection
	iContractDefinitionCollection
	iContractMethodCollection
	iContractEventCollection
	iContractAPICollection
}

// CollectionName represents all collections
type CollectionName string

// OrderedUUIDCollectionNS collections have a strong order that includes a sequence integer
// that uniquely identifies the entry in a sequence. The sequence is LOCAL to this
// FireFly node. We try to minimize adding new collections of this type, as they have
// implementation complexity in some databases (such as NoSQL databases)
type OrderedUUIDCollectionNS CollectionName

const (
	CollectionMessages OrderedUUIDCollectionNS = "message"
	CollectionEvents   OrderedUUIDCollectionNS = "events"
)

// OrderedCollection is a collection that is ordered, and that sequence is the only key
type OrderedCollection CollectionName

const (
	CollectionPins OrderedCollection = "pins"
)

// UUIDCollectionNS is the most common type of collection - each entry has a UUID that
// is globally unique, and used externally by apps to address entries in the collection.
// Objects in these collections are all namespaced,.
type UUIDCollectionNS CollectionName

const (
	CollectionBatches            UUIDCollectionNS = "batches"
	CollectionData               UUIDCollectionNS = "data"
	CollectionDataTypes          UUIDCollectionNS = "datatypes"
	CollectionOperations         UUIDCollectionNS = "operations"
	CollectionSubscriptions      UUIDCollectionNS = "subscriptions"
	CollectionTransactions       UUIDCollectionNS = "transactions"
	CollectionTokenPools         UUIDCollectionNS = "tokenpools"
	CollectionContractInterfaces UUIDCollectionNS = "contractinterfaces"
	CollectionContractMethods    UUIDCollectionNS = "contractmethods"
	CollectionContractEvents     UUIDCollectionNS = "contractevents"
	CollectionContractParams     UUIDCollectionNS = "contractparams"
	CollectionContractAPIs       UUIDCollectionNS = "contractapis"
)

// HashCollectionNS is a collection where the primary key is a hash, such that it can
// by identified by any member of the network at any time, without it first having
// been broadcast.
type HashCollectionNS CollectionName

const (
	CollectionGroups HashCollectionNS = "groups"
)

// UUIDCollection is like UUIDCollectionNS, but for objects that do not reside within a namespace
type UUIDCollection CollectionName

const (
	CollectionNamespaces     UUIDCollection = "namespaces"
	CollectionNodes          UUIDCollection = "nodes"
	CollectionOrganizations  UUIDCollection = "organizations"
	CollectionTokenTransfers UUIDCollection = "tokentransfers"
)

// OtherCollection are odd balls, that don't fit any of the categories above.
// These collections do not support change events, and generally their
// creation is coordinated with creation of another object that does support change events.
// Mainly they are entries that require lookup by compound IDs.
type OtherCollection CollectionName

const (
	CollectionConfigrecords OtherCollection = "configrecords"
	CollectionBlobs         OtherCollection = "blobs"
	CollectionNextpins      OtherCollection = "nextpins"
	CollectionNonces        OtherCollection = "nonces"
	CollectionOffsets       OtherCollection = "offsets"
	CollectionTokenBalances OtherCollection = "tokenbalances"
)

// Callbacks are the methods for passing data from plugin to core
//
// If Capabilities returns ClusterEvents=true then these should be broadcast to every instance within
// a cluster that is connected to the database.
//
// If Capabilities returns ClusterEvents=false then these events can be simply coupled in-process to
// update activities.
//
// The system does not rely on these events exclusively for data/transaction integrity, but if an event is
// missed/delayed it might result in slower processing.
// For example, the batch interface will initiate a batch as soon as an event is triggered, but it will use
// a subsequent database query as the source of truth of the latest set/order of data, and it will periodically
// check for new messages even if it does not receive any events.
//
// Events are emitted locally to the individual FireFly core process. However, a WebSocket interface is
// available for remote listening to these events. That allows the UI to listen to the events, as well as
// providing a building block for a cluster of FireFly servers to directly propgate events to each other.
//
type Callbacks interface {
	// OrderedUUIDCollectionNSEvent emits the sequence on insert, but it will be -1 on update
	OrderedUUIDCollectionNSEvent(resType OrderedUUIDCollectionNS, eventType fftypes.ChangeEventType, ns string, id *fftypes.UUID, sequence int64)
	OrderedCollectionEvent(resType OrderedCollection, eventType fftypes.ChangeEventType, sequence int64)
	UUIDCollectionNSEvent(resType UUIDCollectionNS, eventType fftypes.ChangeEventType, ns string, id *fftypes.UUID)
	UUIDCollectionEvent(resType UUIDCollection, eventType fftypes.ChangeEventType, id *fftypes.UUID)
	HashCollectionNSEvent(resType HashCollectionNS, eventType fftypes.ChangeEventType, ns string, hash *fftypes.Bytes32)
}

// Capabilities defines the capabilities a plugin can report as implementing or not
type Capabilities struct {
	ClusterEvents bool
}

// NamespaceQueryFactory filter fields for namespaces
var NamespaceQueryFactory = &queryFields{
	"id":          &UUIDField{},
	"message":     &UUIDField{},
	"type":        &StringField{},
	"name":        &StringField{},
	"description": &StringField{},
	"created":     &TimeField{},
	"confirmed":   &TimeField{},
}

// MessageQueryFactory filter fields for messages
var MessageQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"cid":       &UUIDField{},
	"namespace": &StringField{},
	"type":      &StringField{},
	"author":    &StringField{},
	"key":       &StringField{},
	"topics":    &FFNameArrayField{},
	"tag":       &StringField{},
	"group":     &Bytes32Field{},
	"created":   &TimeField{},
	"hash":      &Bytes32Field{},
	"pins":      &FFNameArrayField{},
	"state":     &StringField{},
	"confirmed": &TimeField{},
	"sequence":  &Int64Field{},
	"txtype":    &StringField{},
	"batch":     &UUIDField{},
}

// BatchQueryFactory filter fields for batches
var BatchQueryFactory = &queryFields{
	"id":         &UUIDField{},
	"namespace":  &StringField{},
	"type":       &StringField{},
	"author":     &StringField{},
	"key":        &StringField{},
	"group":      &Bytes32Field{},
	"hash":       &Bytes32Field{},
	"payloadref": &StringField{},
	"created":    &TimeField{},
	"confirmed":  &TimeField{},
	"tx.type":    &StringField{},
	"tx.id":      &UUIDField{},
	"node":       &UUIDField{},
}

// TransactionQueryFactory filter fields for transactions
var TransactionQueryFactory = &queryFields{
	"id":         &UUIDField{},
	"type":       &StringField{},
	"signer":     &StringField{},
	"status":     &StringField{},
	"reference":  &UUIDField{},
	"protocolid": &StringField{},
	"created":    &TimeField{},
	"sequence":   &Int64Field{},
	"info":       &JSONField{},
	"namespace":  &StringField{},
}

// DataQueryFactory filter fields for data
var DataQueryFactory = &queryFields{
	"id":               &UUIDField{},
	"namespace":        &StringField{},
	"validator":        &StringField{},
	"datatype.name":    &StringField{},
	"datatype.version": &StringField{},
	"hash":             &Bytes32Field{},
	"blob.hash":        &Bytes32Field{},
	"blob.public":      &StringField{},
	"created":          &TimeField{},
}

// DatatypeQueryFactory filter fields for data definitions
var DatatypeQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"message":   &UUIDField{},
	"namespace": &StringField{},
	"validator": &StringField{},
	"name":      &StringField{},
	"version":   &StringField{},
	"created":   &TimeField{},
}

// OffsetQueryFactory filter fields for data offsets
var OffsetQueryFactory = &queryFields{
	"name":    &StringField{},
	"type":    &StringField{},
	"current": &Int64Field{},
}

// OperationQueryFactory filter fields for data operations
var OperationQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"tx":        &UUIDField{},
	"type":      &StringField{},
	"namespace": &StringField{},
	"status":    &StringField{},
	"error":     &StringField{},
	"plugin":    &StringField{},
	"input":     &JSONField{},
	"output":    &JSONField{},
	"backendid": &StringField{},
	"created":   &TimeField{},
	"updated":   &TimeField{},
}

// SubscriptionQueryFactory filter fields for data subscriptions
var SubscriptionQueryFactory = &queryFields{
	"id":            &UUIDField{},
	"namespace":     &StringField{},
	"name":          &StringField{},
	"transport":     &StringField{},
	"events":        &StringField{},
	"filter.topics": &StringField{},
	"filter.tag":    &StringField{},
	"filter.group":  &StringField{},
	"options":       &StringField{},
	"created":       &TimeField{},
}

// EventQueryFactory filter fields for data events
var EventQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"type":      &StringField{},
	"namespace": &StringField{},
	"reference": &UUIDField{},
	"group":     &Bytes32Field{},
	"sequence":  &Int64Field{},
	"created":   &TimeField{},
}

// PinQueryFactory filter fields for parked contexts
var PinQueryFactory = &queryFields{
	"sequence":   &Int64Field{},
	"masked":     &BoolField{},
	"hash":       &Bytes32Field{},
	"batch":      &UUIDField{},
	"index":      &Int64Field{},
	"dispatched": &BoolField{},
	"created":    &TimeField{},
}

// OrganizationQueryFactory filter fields for organizations
var OrganizationQueryFactory = &queryFields{
	"id":          &UUIDField{},
	"message":     &UUIDField{},
	"parent":      &StringField{},
	"identity":    &StringField{},
	"description": &StringField{},
	"profile":     &JSONField{},
	"created":     &TimeField{},
}

// NodeQueryFactory filter fields for nodes
var NodeQueryFactory = &queryFields{
	"id":          &UUIDField{},
	"message":     &UUIDField{},
	"owner":       &StringField{},
	"name":        &StringField{},
	"description": &StringField{},
	"dx.peer":     &StringField{},
	"dx.endpoint": &JSONField{},
	"created":     &TimeField{},
}

// GroupQueryFactory filter fields for nodes
var GroupQueryFactory = &queryFields{
	"hash":        &Bytes32Field{},
	"message":     &UUIDField{},
	"namespace":   &StringField{},
	"description": &StringField{},
	"ledger":      &UUIDField{},
	"created":     &TimeField{},
}

// NonceQueryFactory filter fields for nodes
var NonceQueryFactory = &queryFields{
	"context": &StringField{},
	"nonce":   &Int64Field{},
	"group":   &Bytes32Field{},
	"topic":   &StringField{},
}

// NextPinQueryFactory filter fields for nodes
var NextPinQueryFactory = &queryFields{
	"context":  &Bytes32Field{},
	"identity": &StringField{},
	"hash":     &Bytes32Field{},
	"nonce":    &Int64Field{},
}

// ConfigRecordQueryFactory filter fields for config records
var ConfigRecordQueryFactory = &queryFields{
	"key":   &StringField{},
	"value": &StringField{},
}

// BlobQueryFactory filter fields for config records
var BlobQueryFactory = &queryFields{
	"hash":       &Bytes32Field{},
	"payloadref": &StringField{},
	"created":    &TimeField{},
}

// TokenPoolQueryFactory filter fields for token pools
var TokenPoolQueryFactory = &queryFields{
	"id":         &UUIDField{},
	"type":       &StringField{},
	"namespace":  &StringField{},
	"name":       &StringField{},
	"standard":   &StringField{},
	"protocolid": &StringField{},
	"key":        &StringField{},
	"symbol":     &StringField{},
	"message":    &UUIDField{},
	"state":      &StringField{},
	"created":    &TimeField{},
	"connector":  &StringField{},
}

// TokenBalanceQueryFactory filter fields for token accounts
var TokenBalanceQueryFactory = &queryFields{
	"pool":       &UUIDField{},
	"tokenindex": &StringField{},
	"uri":        &StringField{},
	"connector":  &StringField{},
	"namespace":  &StringField{},
	"key":        &StringField{},
	"balance":    &Int64Field{},
	"updated":    &TimeField{},
}

// TokenTransferQueryFactory filter fields for token transfers
var TokenTransferQueryFactory = &queryFields{
	"localid":     &StringField{},
	"pool":        &UUIDField{},
	"tokenindex":  &StringField{},
	"uri":         &StringField{},
	"connector":   &StringField{},
	"namespace":   &StringField{},
	"key":         &StringField{},
	"from":        &StringField{},
	"to":          &StringField{},
	"amount":      &Int64Field{},
	"protocolid":  &StringField{},
	"message":     &UUIDField{},
	"messagehash": &Bytes32Field{},
	"created":     &TimeField{},
}

// ContractInterfaceQueryFactory filter fields for contract definitions
var ContractInterfaceQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"namespace": &StringField{},
	"name":      &StringField{},
	"version":   &StringField{},
}

// ContractEventQueryFactory filter fields for contract events
var ContractEventQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"namespace": &StringField{},
	"name":      &StringField{},
}
