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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	// HashMismatch sentinel error
	HashMismatch = i18n.NewError(context.Background(), i18n.MsgHashMismatch)
	// IDMismatch sentinel error
	IDMismatch = i18n.NewError(context.Background(), i18n.MsgIDMismatch)
	// DeleteRecordNotFound sentinel error
	DeleteRecordNotFound = i18n.NewError(context.Background(), i18n.Msg404NotFound)
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

// PeristenceInterface are the operations that must be implemented by a database interfavce plugin.
// The database mechanism of Firefly is designed to provide the balance between being able
// to query the data a member of the network has transferred/received via Firefly efficiently,
// while not trying to become the core database of the application (where full deeply nested
// rich query is needed).
//
// This means that we treat business data as opaque within the stroage, only verifying it against
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
	// Note, the caller is responsible for passing the context back to all database operations performed within the supplied function.
	RunAsGroup(ctx context.Context, fn func(ctx context.Context) error) error

	// UpsertNamespace - Upsert a namespace
	// Throws IDMismatch error if updating and ids don't match
	UpsertNamespace(ctx context.Context, data *fftypes.Namespace, allowExisting bool) (err error)

	// UpdateNamespace - Update namespace
	UpdateNamespace(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// DeleteNamespace - Delete namespace
	DeleteNamespace(ctx context.Context, id *fftypes.UUID) (err error)

	// GetNamespace - Get an namespace by name
	GetNamespace(ctx context.Context, name string) (offset *fftypes.Namespace, err error)

	// GetNamespaces - Get namespaces
	GetNamespaces(ctx context.Context, filter Filter) (offset []*fftypes.Namespace, err error)

	// UpsertMessage - Upsert a message, with all the embedded data references.
	// allowHashUpdate=false throws HashMismatch error if the updated message has a different hash
	UpsertMessage(ctx context.Context, message *fftypes.Message, allowExisting, allowHashUpdate bool) (err error)

	// InsertMessageLocal - sets a boolean flag on inserting a new message (cannot be an update) to state it is local.
	// Only time this flag is ever set. Subsequent updates can affect other fields, but not the local flag. Important to stop infinite message propagation.
	InsertMessageLocal(ctx context.Context, message *fftypes.Message) (err error)

	// UpdateMessage - Update message
	UpdateMessage(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// UpdateMessages - Update messages
	UpdateMessages(ctx context.Context, filter Filter, update Update) (err error)

	// GetMessageByID - Get a message by ID
	GetMessageByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Message, err error)

	// GetMessages - List messages, reverse sorted (newest first) by Confirmed then Created, with pagination, and simple must filters
	GetMessages(ctx context.Context, filter Filter) (message []*fftypes.Message, err error)

	// GetMessageRefs - Lighter weight query to just get the reference info of messages
	GetMessageRefs(ctx context.Context, filter Filter) ([]*fftypes.MessageRef, error)

	// GetMessagesForData - List messages where there is a data reference to the specified ID
	GetMessagesForData(ctx context.Context, dataID *fftypes.UUID, filter Filter) (message []*fftypes.Message, err error)

	// UpsertData - Upsert a data record
	// allowHashUpdate=false throws HashMismatch error if the updated message has a different hash
	UpsertData(ctx context.Context, data *fftypes.Data, allowExisting, allowHashUpdate bool) (err error)

	// UpdateData - Update data
	UpdateData(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetDataByID - Get a data record by ID
	GetDataByID(ctx context.Context, id *fftypes.UUID, withValue bool) (message *fftypes.Data, err error)

	// GetData - Get data
	GetData(ctx context.Context, filter Filter) (message []*fftypes.Data, err error)

	// GetDataRefs - Get data references only (no data)
	GetDataRefs(ctx context.Context, filter Filter) (message fftypes.DataRefs, err error)

	// UpsertBatch - Upsert a batch
	// allowHashUpdate=false throws HashMismatch error if the updated message has a different hash
	UpsertBatch(ctx context.Context, data *fftypes.Batch, allowExisting, allowHashUpdate bool) (err error)

	// UpdateBatch - Update data
	UpdateBatch(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetBatchByID - Get a batch by ID
	GetBatchByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Batch, err error)

	// GetBatches - Get batches
	GetBatches(ctx context.Context, filter Filter) (message []*fftypes.Batch, err error)

	// UpsertTransaction - Upsert a transaction
	// allowHashUpdate=false throws HashMismatch error if the updated message has a different hash
	UpsertTransaction(ctx context.Context, data *fftypes.Transaction, allowExisting, allowHashUpdate bool) (err error)

	// UpdateTransaction - Update transaction
	UpdateTransaction(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetTransactionByID - Get a transaction by ID
	GetTransactionByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Transaction, err error)

	// GetTransactions - Get transactions
	GetTransactions(ctx context.Context, filter Filter) (message []*fftypes.Transaction, err error)

	// UpsertDatatype - Upsert a data definitino
	UpsertDatatype(ctx context.Context, datadef *fftypes.Datatype, allowExisting bool) (err error)

	// UpdateDatatype - Update data definition
	UpdateDatatype(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetDatatypeByID - Get a data definition by ID
	GetDatatypeByID(ctx context.Context, id *fftypes.UUID) (datadef *fftypes.Datatype, err error)

	// GetDatatypeByName - Get a data definition by name
	GetDatatypeByName(ctx context.Context, ns, name, version string) (datadef *fftypes.Datatype, err error)

	// GetDatatypes - Get data definitions
	GetDatatypes(ctx context.Context, filter Filter) (datadef []*fftypes.Datatype, err error)

	// UpsertOffset - Upsert an offset
	UpsertOffset(ctx context.Context, data *fftypes.Offset, allowExisting bool) (err error)

	// UpdateOffset - Update offset
	UpdateOffset(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetOffset - Get an offset by name
	GetOffset(ctx context.Context, t fftypes.OffsetType, ns, name string) (offset *fftypes.Offset, err error)

	// GetOffsets - Get offsets
	GetOffsets(ctx context.Context, filter Filter) (offset []*fftypes.Offset, err error)

	// DeleteOffset - Delete an offset by name
	DeleteOffset(ctx context.Context, t fftypes.OffsetType, ns, name string) (err error)

	// UpsertPin - Will insert a pin at the end of the sequence, unless the batch+hash+index sequence already exists
	UpsertPin(ctx context.Context, parked *fftypes.Pin) (err error)

	// GetPins - Get pins
	GetPins(ctx context.Context, filter Filter) (offset []*fftypes.Pin, err error)

	// SetPinDispatched - Set the dispatched flag to true on the specified pins
	SetPinDispatched(ctx context.Context, sequence int64) (err error)

	// DeletePin - Delete a pin
	DeletePin(ctx context.Context, sequence int64) (err error)

	// UpsertOperation - Upsert an operation
	UpsertOperation(ctx context.Context, operation *fftypes.Operation, allowExisting bool) (err error)

	// UpdateOperation - Update matching operations
	UpdateOperation(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetOperationByID - Get an operation by ID
	GetOperationByID(ctx context.Context, id *fftypes.UUID) (operation *fftypes.Operation, err error)

	// GetOperations - Get operation
	GetOperations(ctx context.Context, filter Filter) (operation []*fftypes.Operation, err error)

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
	GetSubscriptions(ctx context.Context, filter Filter) (offset []*fftypes.Subscription, err error)

	// DeleteSubscriptionByID - Delete a subscription
	DeleteSubscriptionByID(ctx context.Context, id *fftypes.UUID) (err error)

	// UpsertEvent - Upsert an event
	UpsertEvent(ctx context.Context, data *fftypes.Event, allowExisting bool) (err error)

	// UpdateEvent - Update event
	UpdateEvent(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetEventByID - Get a event by ID
	GetEventByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Event, err error)

	// GetEvents - Get events
	GetEvents(ctx context.Context, filter Filter) (message []*fftypes.Event, err error)

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
	GetOrganizations(ctx context.Context, filter Filter) (org []*fftypes.Organization, err error)

	// UpsertNode - Upsert a node
	UpsertNode(ctx context.Context, data *fftypes.Node, allowExisting bool) (err error)

	// UpdateNode - Update node
	UpdateNode(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetNode - Get a node by ID
	GetNode(ctx context.Context, identity string) (node *fftypes.Node, err error)

	// GetNodeByID- Get a node by ID
	GetNodeByID(ctx context.Context, id *fftypes.UUID) (node *fftypes.Node, err error)

	// GetNodes - Get nodes
	GetNodes(ctx context.Context, filter Filter) (node []*fftypes.Node, err error)

	// UpserGroup - Upsert a group
	UpsertGroup(ctx context.Context, data *fftypes.Group, allowExisting bool) (err error)

	// UpdateGroup - Update group
	UpdateGroup(ctx context.Context, id *fftypes.UUID, update Update) (err error)

	// GetGroupByID - Get a group by ID
	GetGroupByID(ctx context.Context, id *fftypes.UUID) (node *fftypes.Group, err error)

	// GetGroups - Get groups
	GetGroups(ctx context.Context, filter Filter) (node []*fftypes.Group, err error)

	// UpsertNonceNext - Upsert a context, assigning zero if not found, or the next nonce if it is
	UpsertNonceNext(ctx context.Context, context *fftypes.Nonce) (err error)

	// GetNonce - Get a context by hash
	GetNonce(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.Nonce, err error)

	// GetNonces - Get contexts
	GetNonces(ctx context.Context, filter Filter) (node []*fftypes.Nonce, err error)

	// DeleteNonce - Delete context by hash
	DeleteNonce(ctx context.Context, hash *fftypes.Bytes32) (err error)

	// InsertNextPin - insert a nextpin
	InsertNextPin(ctx context.Context, nextpin *fftypes.NextPin) (err error)

	// GetNextPinByContextAndIdentity - lookup nextpin by context+identity
	GetNextPinByContextAndIdentity(ctx context.Context, context *fftypes.Bytes32, identity string) (message *fftypes.NextPin, err error)

	// GetNextPinByHash - lookup nextpin by its hash
	GetNextPinByHash(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.NextPin, err error)

	// GetNextPins - get nextpins
	GetNextPins(ctx context.Context, filter Filter) (message []*fftypes.NextPin, err error)

	// UpdateNextPin - update a next hash using its local database ID
	UpdateNextPin(ctx context.Context, sequence int64, update Update) (err error)

	// DeleteNextPin - delete a next hash, using its local database ID
	DeleteNextPin(ctx context.Context, sequence int64) (err error)
}

// Callbacks are the methods for passing data from plugin to core
//
// If Capabilities returns ClusterEvents=true then these should be brodcast to every instance within
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
// TODO: Clarify the relationship between Leader Election capabilities and Event capabilities
//
type Callbacks interface {
	MessageCreated(sequence int64)
	PinCreated(sequence int64)
	EventCreated(sequence int64)
	SubscriptionCreated(id *fftypes.UUID)
	SubscriptionDeleted(id *fftypes.UUID)
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
	"topics":    &FFNameArrayField{},
	"tag":       &StringField{},
	"group":     &UUIDField{},
	"created":   &TimeField{},
	"hash":      &StringField{},
	"pins":      &StringField{},
	"confirmed": &TimeField{},
	"sequence":  &Int64Field{},
	"tx.type":   &StringField{},
	"batch":     &UUIDField{},
	"local":     &BoolField{},
}

// BatchQueryFactory filter fields for batches
var BatchQueryFactory = &queryFields{
	"id":         &UUIDField{},
	"namespace":  &StringField{},
	"type":       &StringField{},
	"author":     &StringField{},
	"group":      &UUIDField{},
	"hash":       &StringField{},
	"payloadref": &StringField{},
	"created":    &TimeField{},
	"confirmed":  &TimeField{},
	"tx.type":    &StringField{},
	"tx.id":      &UUIDField{},
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
	"hash":             &StringField{},
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
	"namespace": &StringField{},
	"name":      &StringField{},
	"type":      &StringField{},
	"current":   &Int64Field{},
}

// OperationQueryFactory filter fields for data operations
var OperationQueryFactory = &queryFields{
	"id":        &UUIDField{},
	"tx":        &UUIDField{},
	"type":      &StringField{},
	"member":    &StringField{},
	"status":    &StringField{},
	"error":     &StringField{},
	"plugin":    &StringField{},
	"info":      &JSONField{},
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
	"sequence":  &Int64Field{},
	"created":   &TimeField{},
}

// PinQueryFactory filter fields for parked contexts
var PinQueryFactory = &queryFields{
	"sequence":   &Int64Field{},
	"masked":     &BoolField{},
	"hash":       &StringField{},
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
	"identity":    &StringField{},
	"description": &StringField{},
	"dx.peer":     &StringField{},
	"dx.endpoint": &JSONField{},
	"created":     &TimeField{},
}

// GroupQueryFactory filter fields for nodes
var GroupQueryFactory = &queryFields{
	"id":          &UUIDField{},
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
	"group":   &UUIDField{},
	"topic":   &StringField{},
}

// NextPinQueryFactory filter fields for nodes
var NextPinQueryFactory = &queryFields{
	"context":  &StringField{},
	"identity": &StringField{},
	"hash":     &StringField{},
	"nonce":    &Int64Field{},
}
