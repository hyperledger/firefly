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

package coreconfig

import (
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/spf13/viper"
)

var ffc = config.AddRootKey

const (
	// PluginConfigName is the user-supplied name for this plugin type
	PluginConfigName = "name"
	// PluginConfigType is the type of the plugin to be loaded
	PluginConfigType = "type"
	// PluginRemoteName is the plugin name to be sent in plugin calls
	PluginRemoteName = "remotename"
	// NamespaceName is the short name for a pre-defined namespace
	NamespaceName = "name"
	// NamespaceName is the long description for a pre-defined namespace
	NamespaceDescription = "description"
	// NamespaceRemoteName is the namespace name to be sent in plugin calls
	NamespaceRemoteName = "remotename"
	// NamespacePlugins is the list of namespace plugins
	NamespacePlugins = "plugins"
	// NamespaceDefaultKey is the default signing key for blockchain transactions within this namespace
	NamespaceDefaultKey = "defaultKey"
	// NamespaceAssetKeyNormalization mechanism to normalize keys before using them. Valid options: "blockchain_plugin" - use blockchain plugin (default), "none" - do not attempt normalization
	NamespaceAssetKeyNormalization = "asset.manager.keyNormalization"
	// NamespaceMultiparty contains the multiparty configuration for a namespace
	NamespaceMultiparty = "multiparty"
	// NamespaceMultipartyEnabled specifies if multi-party mode is enabled for a namespace
	NamespaceMultipartyEnabled = "enabled"
	// NamespaceMultipartyOrgName is a short name for the local root org within a namespace
	NamespaceMultipartyOrgName = "org.name"
	// NamespaceMultipartyOrgDescription is a description for the local root org within a namespace
	NamespaceMultipartyOrgDescription = "org.description"
	// NamespaceMultipartyOrgKey is the signing key allocated to the local root org within a namespace
	NamespaceMultipartyOrgKey = "org.key"
	// NamespaceMultipartyNodeName is the name for the local node within a namespace
	NamespaceMultipartyNodeName = "node.name"
	// NamespaceMultipartyNodeName is a description for the local node within a namespace
	NamespaceMultipartyNodeDescription = "node.description"
	// NamespaceMultipartyContract is a list of firefly contract configurations for this namespace
	NamespaceMultipartyContract = "contract"
	// NamespaceMultipartyContractFirstEvent is the first event to process for this contract
	NamespaceMultipartyContractFirstEvent = "firstEvent"
	// NamespaceMultipartyContractLocation is an object containing blockchain specific configuration
	NamespaceMultipartyContractLocation = "location"
)

// The following keys can be access from the root configuration.
// Plugins are responsible for defining their own keys using the Config interface
var (
	// APIDefaultFilterLimit is the default limit that will be applied to filtered queries on the API
	APIDefaultFilterLimit = ffc("api.defaultFilterLimit")
	// APIMaxFilterLimit is the maximum limit that can be specified by an API call
	APIMaxFilterLimit = ffc("api.maxFilterLimit")
	// APIMaxFilterSkip is the maximum skip value that can be specified on the API
	APIMaxFilterSkip = ffc("api.maxFilterLimit")
	// APIRequestTimeout is the server side timeout for API calls (context timeout), to avoid the server continuing processing when the client gives up
	APIRequestTimeout = ffc("api.requestTimeout")
	// APIRequestMaxTimeout is the maximum timeout an application can set using a Request-Timeout header
	APIRequestMaxTimeout = ffc("api.requestMaxTimeout")
	// APIOASPanicOnMissingDescription controls whether the OpenAPI Spec generator will strongly enforce descriptions on every field or not
	APIOASPanicOnMissingDescription = ffc("api.oas.panicOnMissingDescription")
	// BatchCacheLimit max number of cache items for batches
	BatchCacheLimit = ffc("batch.cache.limit")
	// BatchCacheTTL time to live for cache of batches
	BatchCacheTTL = ffc("batch.cache.ttl")
	// BatchManagerReadPageSize is the size of each page of messages read from the database into memory when assembling batches
	BatchManagerReadPageSize = ffc("batch.manager.readPageSize")
	// BatchManagerReadPollTimeout is how long without any notifications of new messages to wait, before doing a page query
	BatchManagerReadPollTimeout = ffc("batch.manager.pollTimeout")
	// BatchManagerMinimumPollDelay is the minimum time the batch manager waits between polls on the DB - to prevent thrashing
	BatchManagerMinimumPollDelay = ffc("batch.manager.minimumPollDelay")
	// BatchRetryFactor is the retry backoff factor for database operations performed by the batch manager
	BatchRetryFactor = ffc("batch.retry.factor")
	// BatchRetryInitDelay is the retry initial delay for database operations
	BatchRetryInitDelay = ffc("batch.retry.initDelay")
	// BatchRetryMaxDelay is the maximum delay between retry attempts
	BatchRetryMaxDelay = ffc("batch.retry.maxDelay")
	// BlobReceiverRetryInitDelay is the initial retry delay
	BlobReceiverRetryInitDelay = ffc("blobreceiver.retry.initialDelay")
	// BlobReceiverRetryMaxDelay is the maximum retry delay
	BlobReceiverRetryMaxDelay = ffc("blobreceiver.retry.maxDelay")
	// BlobReceiverRetryFactor is the backoff factor to use for retries
	BlobReceiverRetryFactor = ffc("blobreceiver.retry.factor")
	// BlobReceiverWorkerCount
	BlobReceiverWorkerCount = ffc("blobreceiver.worker.count")
	// BlobReceiverWorkerBatchTimeout
	BlobReceiverWorkerBatchTimeout = ffc("blobreceiver.worker.batchTimeout")
	// BlobReceiverWorkerBatchMaxInserts
	BlobReceiverWorkerBatchMaxInserts = ffc("blobreceiver.worker.batchMaxInserts")
	// BlockchainEventCacheLimit max number of cache items for blockchain events
	BlockchainEventCacheLimit = ffc("blockchainevent.cache.limit")
	// BlockchainEventCacheTTL time to live for cache of blockchain events
	BlockchainEventCacheTTL = ffc("blockchainevent.cache.ttl")
	// BroadcastBatchAgentTimeout how long to keep around a batching agent for a sending identity before disposal
	BroadcastBatchAgentTimeout = ffc("broadcast.batch.agentTimeout")
	// BroadcastBatchSize is the maximum number of messages that can be packed into a batch
	BroadcastBatchSize = ffc("broadcast.batch.size")
	// BroadcastBatchPayloadLimit is the maximum payload size of a batch for broadcast messages
	BroadcastBatchPayloadLimit = ffc("broadcast.batch.payloadLimit")
	// BroadcastBatchTimeout is the timeout to wait for a batch to fill, before sending
	BroadcastBatchTimeout = ffc("broadcast.batch.timeout")
	// CacheBlockchainTTL time to live of blockchain plugin cache
	CacheBlockchainTTL = ffc("cache.blockchain.ttl")
	// CacheBlockchainLimit max number of cache items for blockchain plugin cache
	CacheBlockchainLimit = ffc("cache.blockchain.limit")
	// CacheOperationsTTL time to live for cache of operations
	CacheOperationsTTL = ffc("cache.operations.ttl")
	// CacheOperationsLimit the max number of cache items for operations
	CacheOperationsLimit = ffc("cache.operations.limit")
	// DownloadWorkerCount is the number of download workers created to pull data from shared storage to the local DX
	DownloadWorkerCount = ffc("download.worker.count")
	// DownloadWorkerQueueLength is the length of the work queue in the channel to the workers - defaults to 2x the worker count
	DownloadWorkerQueueLength = ffc("download.worker.queueLength")
	// DownloadRetryMaxAttempts is the maximum number of automatic attempts to make for each shared storage download before failing the operation
	DownloadRetryMaxAttempts = ffc("download.retry.maxAttempts")
	// DownloadRetryInitDelay is the initial retry delay
	DownloadRetryInitDelay = ffc("download.retry.initialDelay")
	// DownloadRetryMaxDelay is the maximum retry delay
	DownloadRetryMaxDelay = ffc("download.retry.maxDelay")
	// DownloadRetryFactor is the backoff factor to use for retries
	DownloadRetryFactor = ffc("download.retry.factor")
	// PrivateMessagingBatchAgentTimeout how long to keep around a batching agent for a sending identity before disposal
	PrivateMessagingBatchAgentTimeout = ffc("privatemessaging.batch.agentTimeout")
	// PrivateMessagingBatchSize is the maximum size of a batch for broadcast messages
	PrivateMessagingBatchSize = ffc("privatemessaging.batch.size")
	// PrivateMessagingBatchPayloadLimit is the maximum payload size of a private message data exchange payload
	PrivateMessagingBatchPayloadLimit = ffc("privatemessaging.batch.payloadLimit")
	// PrivateMessagingBatchTimeout is the timeout to wait for a batch to fill, before sending
	PrivateMessagingBatchTimeout = ffc("privatemessaging.batch.timeout")
	// PrivateMessagingRetryFactor the backoff factor to use for retry of database operations
	PrivateMessagingRetryFactor = ffc("privatemessaging.retry.factor")
	// PrivateMessagingRetryInitDelay the initial delay to use for retry of data base operations
	PrivateMessagingRetryInitDelay = ffc("privatemessaging.retry.initDelay")
	// PrivateMessagingRetryMaxDelay the maximum delay to use for retry of data base operations
	PrivateMessagingRetryMaxDelay = ffc("privatemessaging.retry.maxDelay")
	// DatabaseType the type of the database interface plugin to use
	HistogramsMaxChartRows = ffc("histograms.maxChartRows")
	// TokensList is the root key containing a list of supported token connectors
	TokensList = ffc("tokens")
	// PluginsTokensList is the key containing a list of supported tokens plugins
	PluginsTokensList = ffc("plugins.tokens")
	// PluginsAuthList is the key containing a list of supported auth plugins
	PluginsAuthList = ffc("plugins.auth")
	// PluginsBlockchainList is the key containing a list of configured blockchain plugins
	PluginsBlockchainList = ffc("plugins.blockchain")
	// PluginsSharedStorageList is the key containing a list of configured shared storage plugins
	PluginsSharedStorageList = ffc("plugins.sharedstorage")
	// PluginsDatabaseList is the key containing a list of configured database plugins
	PluginsDatabaseList = ffc("plugins.database")
	// PluginsDataExchangeList is the key containing a list of configured database plugins
	PluginsDataExchangeList = ffc("plugins.dataexchange")
	// PluginsIdentityList is the key containing a list of configured identity plugins
	PluginsIdentityList = ffc("plugins.identity")
	// DebugPort a HTTP port on which to enable the go debugger
	DebugPort = ffc("debug.port")
	// EventTransportsDefault the default event transport for new subscriptions
	EventTransportsDefault = ffc("event.transports.default")
	// EventTransportsEnabled which event interface plugins are enabled
	EventTransportsEnabled = ffc("event.transports.enabled")
	// EventAggregatorFirstEvent the first event the aggregator should process, if no previous offest is stored in the DB
	EventAggregatorFirstEvent = ffc("event.aggregator.firstEvent")
	// EventAggregatorBatchSize the maximum number of records to read from the DB before performing an aggregation run
	EventAggregatorBatchSize = ffc("event.aggregator.batchSize")
	// EventAggregatorBatchTimeout how long to wait for new events to arrive before performing aggregation on a page of events
	EventAggregatorBatchTimeout = ffc("event.aggregator.batchTimeout")
	// EventAggregatorPollTimeout the time to wait without a notification of new events, before trying a select on the table
	EventAggregatorPollTimeout = ffc("event.aggregator.pollTimeout")
	// EventAggregatorRewindTimeout the minimum time to wait for rewinds to accumulate before resolving them
	EventAggregatorRewindTimeout = ffc("event.aggregator.rewindTimeout")
	// EventAggregatorRewindQueueLength the size of the queue into the rewind dispatcher
	EventAggregatorRewindQueueLength = ffc("event.aggregator.rewindQueueLength")
	// EventAggregatorRewindQueryLimit safety limit on the maximum number of records to search when performing queries to search for rewinds
	EventAggregatorRewindQueryLimit = ffc("event.aggregator.rewindQueryLimit")
	// EventAggregatorRetryFactor the backoff factor to use for retry of database operations
	EventAggregatorRetryFactor = ffc("event.aggregator.retry.factor")
	// EventAggregatorRetryInitDelay the initial delay to use for retry of data base operations
	EventAggregatorRetryInitDelay = ffc("event.aggregator.retry.initDelay")
	// EventAggregatorRetryMaxDelay the maximum delay to use for retry of data base operations
	EventAggregatorRetryMaxDelay = ffc("event.aggregator.retry.maxDelay")
	// EventDispatcherPollTimeout the time to wait without a notification of new events, before trying a select on the table
	EventDispatcherPollTimeout = ffc("event.dispatcher.pollTimeout")
	// EventDispatcherBufferLength the number of events + attachments an individual dispatcher should hold in memory ready for delivery to the subscription
	EventDispatcherBufferLength = ffc("event.dispatcher.bufferLength")
	// EventDispatcherBatchTimeout a short time to wait for new events to arrive before re-polling for new events
	EventDispatcherBatchTimeout = ffc("event.dispatcher.batchTimeout")
	// EventDispatcherRetryFactor the backoff factor to use for retry of database operations
	EventDispatcherRetryFactor = ffc("event.dispatcher.retry.factor")
	// EventDispatcherRetryInitDelay he initial delay to use for retry of data base operations
	EventDispatcherRetryInitDelay = ffc("event.dispatcher.retry.initDelay")
	// EventDispatcherRetryMaxDelay he maximum delay to use for retry of data base operations
	EventDispatcherRetryMaxDelay = ffc("event.dispatcher.retry.maxDelay")
	// EventDBEventsBufferSize the size of the buffer of change events
	EventDBEventsBufferSize = ffc("event.dbevents.bufferSize")
	// EventListenerTopicCacheLimit max number of cache items for blockchain listener topics
	EventListenerTopicCacheLimit = ffc("event.listenerTopic.cache.limit")
	// EventListenerTopicCacheTTL time-to-live for for cache of blockchain listener topics
	EventListenerTopicCacheTTL = ffc("event.listenerTopic.cache.ttl")
	// GroupCacheLimit cache size for private group addresses
	GroupCacheLimit = ffc("group.cache.limit")
	// GroupCacheTTL cache time-to-live for private group addresses
	GroupCacheTTL = ffc("group.cache.ttl")
	// SPIEnabled determines whether the admin interface will be enabled or not
	SPIEnabled = ffc("spi.enabled")
	// SPIWebSocketEventQueueLength is the maximum number of events that will queue up on the server side of each WebSocket connection before events start being dropped
	SPIWebSocketEventQueueLength = ffc("spi.ws.eventQueueLength")
	// SPIWebSocketBlockedWarnInterval how often to emit a warning if an admin.ws is blocked and not receiving events
	SPIWebSocketBlockedWarnInterval = ffc("spi.ws.blockedWarnInterval")
	// SPIWebSocketReadBufferSize is the WebSocket read buffer size for the admin change-event WebSocket
	SPIWebSocketReadBufferSize = ffc("spi.ws.readBufferSize")
	// SPIWebSocketWriteBufferSize is the WebSocket write buffer size for the admin change-event WebSocket
	SPIWebSocketWriteBufferSize = ffc("spi.ws.writeBufferSize")
	// IdentityManagerCacheTTL the identity manager cache time to live
	IdentityManagerCacheTTL = ffc("identity.manager.cache.ttl")
	// IdentityManagerCacheLimit the identity manager cache limit in count of items
	IdentityManagerCacheLimit = ffc("identity.manager.cache.limit")
	// MessageCacheSize max size for cache of messages
	MessageCacheSize = ffc("message.cache.size")
	// MessageCacheTTL time-to-live for cache of messages
	MessageCacheTTL = ffc("message.cache.ttl")
	// MessageWriterCount
	MessageWriterCount = ffc("message.writer.count")
	// MessageWriterBatchTimeout
	MessageWriterBatchTimeout = ffc("message.writer.batchTimeout")
	// MessageWriterBatchMaxInserts
	MessageWriterBatchMaxInserts = ffc("message.writer.batchMaxInserts")
	// MetricsEnabled determines whether metrics will be instrumented and if the metrics server will be enabled or not
	MetricsEnabled = ffc("metrics.enabled")
	// MetricsPath determines what path to serve the Prometheus metrics from
	MetricsPath = ffc("metrics.path")
	// NamespacesDefault is the default namespace - must be in the predefines list
	NamespacesDefault = ffc("namespaces.default")
	// NamespacesPredefined is a list of namespaces to ensure exists, without requiring a broadcast from the network
	NamespacesPredefined = ffc("namespaces.predefined")
	// NodeName is the short name for the node
	NodeName = ffc("node.name")
	// NodeDescription is a description for the node
	NodeDescription = ffc("node.description")
	// OpUpdateRetryInitDelay is the initial retry delay
	OpUpdateRetryInitDelay = ffc("opupdate.retry.initialDelay")
	// OpUpdatedRetryMaxDelay is the maximum retry delay
	OpUpdateRetryMaxDelay = ffc("opupdate.retry.maxDelay")
	// OpUpdateRetryFactor is the backoff factor to use for retries
	OpUpdateRetryFactor = ffc("opupdate.retry.factor")
	// OpUpdateWorkerCount
	OpUpdateWorkerCount = ffc("opupdate.worker.count")
	// OpUpdateWorkerBatchTimeout
	OpUpdateWorkerBatchTimeout = ffc("opupdate.worker.batchTimeout")
	// OpUpdateWorkerBatchMaxInserts
	OpUpdateWorkerBatchMaxInserts = ffc("opupdate.worker.batchMaxInserts")
	// OpUpdateWorkerQueueLength
	OpUpdateWorkerQueueLength = ffc("opupdate.worker.queueLength")
	// OrgName is the short name for the org
	OrgName = ffc("org.name")
	// OrgKey is the signing identity allocated to the organization (can be the same as the nodes)
	OrgKey = ffc("org.key")
	// OrgDescription is a description for the org
	OrgDescription = ffc("org.description")
	// OrchestratorStartupAttempts is how many time to attempt to connect to core infrastructure on startup
	OrchestratorStartupAttempts = ffc("orchestrator.startupAttempts")
	// SubscriptionDefaultsReadAhead default read ahead to enable for subscriptions that do not explicitly configure readahead
	SubscriptionDefaultsReadAhead = ffc("subscription.defaults.batchSize")
	// SubscriptionMax maximum number of pre-defined subscriptions that can exist (note for high fan-out consider connecting a dedicated pub/sub broker to the dispatcher)
	SubscriptionMax = ffc("subscription.max")
	// SubscriptionsRetryInitialDelay is the initial retry delay
	SubscriptionsRetryInitialDelay = ffc("subscription.retry.initDelay")
	// SubscriptionsRetryMaxDelay is the initial retry delay
	SubscriptionsRetryMaxDelay = ffc("subscription.retry.maxDelay")
	// SubscriptionsRetryFactor the backoff factor to use for retry of database operations
	SubscriptionsRetryFactor = ffc("subscription.retry.factor")
	// TransactionCacheSize max size for cache of transactions
	TransactionCacheSize = ffc("transaction.cache.size")
	// TransactionCacheTTL time-to-live for cache of transactions
	TransactionCacheTTL = ffc("transaction.cache.ttl")
	// AssetManagerKeyNormalization mechanism to normalize keys before using them. Valid options: "blockchain_plugin" - use blockchain plugin (default), "none" - do not attempt normalization
	AssetManagerKeyNormalization = ffc("asset.manager.keyNormalization")
	// UIEnabled set to false to disable the UI (default is true, so UI will be enabled if ui.path is valid)
	UIEnabled = ffc("ui.enabled")
	// UIPath the path on which to serve the UI
	UIPath = ffc("ui.path")
	// ValidatorCacheSize max size for cache of validators
	ValidatorCacheSize = ffc("validator.cache.size")
	// ValidatorCacheTTL time-to-live for cache of validators
	ValidatorCacheTTL = ffc("validator.cache.ttl")
)

func setDefaults() {
	// Set defaults
	viper.SetDefault(string(APIDefaultFilterLimit), 25)
	viper.SetDefault(string(APIRequestTimeout), "120s")
	viper.SetDefault(string(APIRequestMaxTimeout), "10m")
	viper.SetDefault(string(APIMaxFilterLimit), 250)
	viper.SetDefault(string(APIMaxFilterSkip), 1000) // protects database (skip+limit pagination is not for bulk operations)
	viper.SetDefault(string(APIRequestTimeout), "120s")
	viper.SetDefault(string(AssetManagerKeyNormalization), "blockchain_plugin")
	viper.SetDefault(string(BatchCacheLimit), 100)
	viper.SetDefault(string(BatchCacheTTL), "5m")
	viper.SetDefault(string(BatchManagerReadPageSize), 100)
	viper.SetDefault(string(BatchManagerReadPollTimeout), "30s")
	viper.SetDefault(string(BatchManagerMinimumPollDelay), "100ms")
	viper.SetDefault(string(BatchRetryFactor), 2.0)
	viper.SetDefault(string(BatchRetryFactor), 2.0)
	viper.SetDefault(string(BatchRetryInitDelay), "250ms")
	viper.SetDefault(string(BatchRetryInitDelay), "250ms")
	viper.SetDefault(string(BatchRetryMaxDelay), "30s")
	viper.SetDefault(string(BatchRetryMaxDelay), "30s")
	viper.SetDefault(string(BlobReceiverRetryInitDelay), "250ms")
	viper.SetDefault(string(BlobReceiverRetryMaxDelay), "1m")
	viper.SetDefault(string(BlobReceiverRetryFactor), 2.0)
	viper.SetDefault(string(BlobReceiverWorkerBatchTimeout), "50ms")
	viper.SetDefault(string(BlobReceiverWorkerCount), 5)
	viper.SetDefault(string(BlobReceiverWorkerBatchMaxInserts), 200)
	viper.SetDefault(string(BlockchainEventCacheLimit), 100)
	viper.SetDefault(string(BlockchainEventCacheTTL), "5m")
	viper.SetDefault(string(BroadcastBatchAgentTimeout), "2m")
	viper.SetDefault(string(BroadcastBatchSize), 200)
	viper.SetDefault(string(BroadcastBatchPayloadLimit), "800Kb")
	viper.SetDefault(string(BroadcastBatchTimeout), "1s")
	viper.SetDefault(string(CacheBlockchainLimit), 100)
	viper.SetDefault(string(CacheBlockchainTTL), "5m")
	viper.SetDefault(string(CacheOperationsLimit), 200)
	viper.SetDefault(string(CacheOperationsTTL), "5m")
	viper.SetDefault(string(HistogramsMaxChartRows), 100)
	viper.SetDefault(string(DebugPort), -1)
	viper.SetDefault(string(DownloadWorkerCount), 10)
	viper.SetDefault(string(DownloadRetryMaxAttempts), 100)
	viper.SetDefault(string(DownloadRetryInitDelay), "100ms")
	viper.SetDefault(string(DownloadRetryMaxDelay), "1m")
	viper.SetDefault(string(DownloadRetryFactor), 2.0)
	viper.SetDefault(string(EventAggregatorFirstEvent), core.SubOptsFirstEventOldest)
	viper.SetDefault(string(EventAggregatorBatchSize), 200)
	viper.SetDefault(string(EventAggregatorBatchTimeout), "250ms")
	viper.SetDefault(string(EventAggregatorPollTimeout), "30s")
	viper.SetDefault(string(EventAggregatorRewindTimeout), "50ms")
	viper.SetDefault(string(EventAggregatorRewindQueueLength), 10)
	viper.SetDefault(string(EventAggregatorRewindQueryLimit), 1000)
	viper.SetDefault(string(EventAggregatorRetryFactor), 2.0)
	viper.SetDefault(string(EventAggregatorRetryInitDelay), "100ms")
	viper.SetDefault(string(EventAggregatorRetryMaxDelay), "30s")
	viper.SetDefault(string(EventDBEventsBufferSize), 100)
	viper.SetDefault(string(EventDispatcherBufferLength), 5)
	viper.SetDefault(string(EventDispatcherBatchTimeout), "250ms")
	viper.SetDefault(string(EventDispatcherPollTimeout), "30s")
	viper.SetDefault(string(EventTransportsEnabled), []string{"websockets", "webhooks"})
	viper.SetDefault(string(EventTransportsDefault), "websockets")
	viper.SetDefault(string(EventListenerTopicCacheLimit), 100)
	viper.SetDefault(string(EventListenerTopicCacheTTL), "5m")
	viper.SetDefault(string(GroupCacheLimit), 50)
	viper.SetDefault(string(GroupCacheTTL), "1h")
	viper.SetDefault(string(SPIEnabled), false)
	viper.SetDefault(string(SPIWebSocketReadBufferSize), "16Kb")
	viper.SetDefault(string(SPIWebSocketWriteBufferSize), "16Kb")
	viper.SetDefault(string(SPIWebSocketBlockedWarnInterval), "1m")
	viper.SetDefault(string(SPIWebSocketEventQueueLength), 250)
	viper.SetDefault(string(MessageCacheSize), "50Mb")
	viper.SetDefault(string(MessageCacheTTL), "5m")
	viper.SetDefault(string(MessageWriterBatchMaxInserts), 200)
	viper.SetDefault(string(MessageWriterBatchTimeout), "10ms")
	viper.SetDefault(string(MessageWriterCount), 5)
	viper.SetDefault(string(NamespacesDefault), "default")
	viper.SetDefault(string(OrchestratorStartupAttempts), 5)
	viper.SetDefault(string(OpUpdateRetryInitDelay), "250ms")
	viper.SetDefault(string(OpUpdateRetryMaxDelay), "1m")
	viper.SetDefault(string(OpUpdateRetryFactor), 2.0)
	viper.SetDefault(string(OpUpdateWorkerBatchTimeout), "50ms")
	viper.SetDefault(string(OpUpdateWorkerCount), 5)
	viper.SetDefault(string(OpUpdateWorkerBatchMaxInserts), 200)
	viper.SetDefault(string(OpUpdateWorkerQueueLength), 50)
	viper.SetDefault(string(PrivateMessagingRetryFactor), 2.0)
	viper.SetDefault(string(PrivateMessagingRetryInitDelay), "100ms")
	viper.SetDefault(string(PrivateMessagingRetryMaxDelay), "30s")
	viper.SetDefault(string(PrivateMessagingBatchAgentTimeout), "2m")
	viper.SetDefault(string(PrivateMessagingBatchSize), 200)
	viper.SetDefault(string(PrivateMessagingBatchTimeout), "1s")
	viper.SetDefault(string(PrivateMessagingBatchPayloadLimit), "800Kb")
	viper.SetDefault(string(SubscriptionDefaultsReadAhead), 0)
	viper.SetDefault(string(SubscriptionMax), 500)
	viper.SetDefault(string(SubscriptionsRetryInitialDelay), "250ms")
	viper.SetDefault(string(SubscriptionsRetryMaxDelay), "30s")
	viper.SetDefault(string(SubscriptionsRetryFactor), 2.0)
	viper.SetDefault(string(TransactionCacheSize), "1Mb")
	viper.SetDefault(string(TransactionCacheTTL), "5m")
	viper.SetDefault(string(UIEnabled), true)
	viper.SetDefault(string(ValidatorCacheSize), "1Mb")
	viper.SetDefault(string(ValidatorCacheTTL), "1h")
	viper.SetDefault(string(IdentityManagerCacheLimit), 100 /* items */)
	viper.SetDefault(string(IdentityManagerCacheTTL), "1h")
}

func Reset() {
	config.RootConfigReset(setDefaults)
}
