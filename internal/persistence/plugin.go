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

package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
)

// Plugin is the interface implemented by each plugin
type Plugin interface {
	PeristenceInterface // Split out to aid pluggability the next level down (SQL provider etc.)

	// ConfigInterface returns the structure into which to marshal the plugin config
	ConfigInterface() interface{}

	// Init initializes the plugin, with the config marshaled into the return of ConfigInterface
	// Returns the supported featureset of the interface
	Init(ctx context.Context, config interface{}, events Events) (*Capabilities, error)
}

// The persistence mechanism of Firefly is designed to provide the balance between being able
// to query the data a member of the network has transferred/received via Firefly efficiently,
// while not trying to become the core database of the application (where full deeply nested
// rich query is needed).
//
// This means that we treat business data as opaque within the stroage, only verifying it against
// a data schema within the Firefly core runtime itself.
// The data types, indexes and relationships are designed to be simple, and map closely to the
// REST semantics of the Firefly API itself.
//
// As a result, the persistence interface could be implemented efficiently by most database technologies.
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
	// Upsert a message
	UpsertMessage(ctx context.Context, message *fftypes.MessageRefsOnly) (err error)

	// Get a message by Id
	GetMessageById(ctx context.Context, id *uuid.UUID) (message *fftypes.MessageRefsOnly, err error)

	// List messages, reverse sorted (newest first) by Confirmed then Created, with pagination, and simple must filters
	GetMessages(ctx context.Context, skip, limit uint64, filter *MessageFilter) (message []*fftypes.MessageRefsOnly, err error)

	// Upsert a data record
	UpsertData(ctx context.Context, data *fftypes.Data) (err error)

	// Get a data record by Id
	GetDataById(ctx context.Context, id *uuid.UUID) (message *fftypes.Data, err error)

	// Upsert a batch
	UpsertBatch(ctx context.Context, data *fftypes.Batch) (err error)

	// Get a batch by Id
	GetBatchById(ctx context.Context, id *uuid.UUID) (message *fftypes.Batch, err error)

	// Get batches
	GetBatches(ctx context.Context, skip, limit uint64, filter *BatchFilter) (message []*fftypes.Batch, err error)

	// Upsert a transaction
	UpsertTransaction(ctx context.Context, data *fftypes.Transaction) (err error)

	// Get a transaction by Id
	GetTransactionById(ctx context.Context, id *uuid.UUID) (message *fftypes.Transaction, err error)

	// Get transaction
	GetTransactions(ctx context.Context, skip, limit uint64, filter *TransactionFilter) (message []*fftypes.Transaction, err error)
}

// No events currently defined for the persistence interface
type Events interface {
}

// No capabilities currently defined for the persistence interface - all features are mandatory
type Capabilities struct {
}

type MessageFilter struct {
	ConfrimedOnly   bool
	UnconfrimedOnly bool
	IDEquals        *uuid.UUID
	NamespaceEquals string
	TypeEquals      string
	AuthorEquals    string
	TopicEquals     string
	ContextEquals   string
	GroupEquals     *uuid.UUID
	CIDEquals       *uuid.UUID
	CreatedAfter    uint64
	ConfirmedAfter  uint64
}

type BatchFilter struct {
	ConfrimedOnly   bool
	UnconfrimedOnly bool
	IDEquals        *uuid.UUID
	AuthorEquals    string
	NamespaceEquals string
	CreatedAfter    uint64
	ConfirmedAfter  uint64
}

type TransactionFilter struct {
	ConfrimedOnly    bool
	UnconfrimedOnly  bool
	IDEquals         *uuid.UUID
	AuthorEquals     string
	TrackingIDEquals string
	ProtocolIDEquals string
	CreatedAfter     uint64
	ConfirmedAfter   uint64
}
