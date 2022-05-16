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

import (
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"golang.org/x/text/language"
)

var ffc = i18n.FFC

//revive:disable
var (
	ConfigGlobalMigrationsAuto      = ffc(language.AmericanEnglish, "config.global.migrations.auto", "Enables automatic database migrations", i18n.BooleanType)
	ConfigGlobalMigrationsDirectory = ffc(language.AmericanEnglish, "config.global.migrations.directory", "The directory containing the numerically ordered migration DDL files to apply to the database", i18n.StringType)
	ConfigGlobalShutdownTimeout     = ffc(language.AmericanEnglish, "config.global.shutdownTimeout", "The maximum amount of time to wait for any open HTTP requests to finish before shutting down the HTTP server", i18n.TimeDurationType)

	ConfigAdminAddress      = ffc(language.AmericanEnglish, "config.admin.address", "The IP address on which the admin HTTP API should listen", "IP Address "+i18n.StringType)
	ConfigAdminEnabled      = ffc(language.AmericanEnglish, "config.admin.enabled", "Enables the admin HTTP API", i18n.BooleanType)
	ConfigAdminPort         = ffc(language.AmericanEnglish, "config.admin.port", "The port on which the admin HTTP API should listen", i18n.IntType)
	ConfigAdminPreInit      = ffc(language.AmericanEnglish, "config.admin.preinit", "Enables the pre-init mode. This mode will let the FireFly Core process start, but not initialize any plugins, besides the database to read any configuration overrides. This allows the admin HTTP API to be used to define custom configuration before starting the rest of FireFly Core.", i18n.BooleanType)
	ConfigAdminPublicURL    = ffc(language.AmericanEnglish, "config.admin.publicURL", "The fully qualified public URL for the admin API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.", "URL "+i18n.StringType)
	ConfigAdminReadTimeout  = ffc(language.AmericanEnglish, "config.admin.readTimeout", "The maximum time to wait when reading from an HTTP connection", i18n.TimeDurationType)
	ConfigAdminWriteTimeout = ffc(language.AmericanEnglish, "config.admin.writeTimeout", "The maximum time to wait when writing to an HTTP connection", i18n.TimeDurationType)

	ConfigAPIDefaultFilterLimit        = ffc(language.AmericanEnglish, "config.api.defaultFilterLimit", "The maximum number of rows to return if no limit is specified on an API request", i18n.IntType)
	ConfigAPIMaxFilterLimit            = ffc(language.AmericanEnglish, "config.api.maxFilterLimit", "The largest value of `limit` that an HTTP client can specify in a request", i18n.IntType)
	ConfigAPIRequestMaxTimeout         = ffc(language.AmericanEnglish, "config.api.requestMaxTimeout", "The maximum amount of time that an HTTP client can specify in a `Request-Timeout` header to keep a specific request open", i18n.TimeDurationType)
	ConfigAssetManagerKeyNormalization = ffc(language.AmericanEnglish, "config.asset.manager.keyNormalization", "Mechanism to normalize keys before using them. Valid options are `blockchain_plugin` - use blockchain plugin (default) or `none` - do not attempt normalization", i18n.StringType)

	ConfigBatchManagerMinimumPollDelay = ffc(language.AmericanEnglish, "config.batch.manager.minimumPollDelay", "The minimum time the batch manager waits between polls on the DB - to prevent thrashing", i18n.TimeDurationType)
	ConfigBatchManagerPollTimeout      = ffc(language.AmericanEnglish, "config.batch.manager.pollTimeout", "How long to wait without any notifications of new messages before doing a page query", i18n.TimeDurationType)
	ConfigBatchManagerReadPageSize     = ffc(language.AmericanEnglish, "config.batch.manager.readPageSize", "The size of each page of messages read from the database into memory when assembling batches", i18n.IntType)

	ConfigBlobreceiverWorkerBatchMaxInserts = ffc(language.AmericanEnglish, "config.blobreceiver.worker.batchMaxInserts", "The maximum number of items the blob receiver worker will insert in a batch", i18n.IntType)
	ConfigBlobreceiverWorkerBatchTimeout    = ffc(language.AmericanEnglish, "config.blobreceiver.worker.batchTimeout", "The maximum amount of the the blob receiver worker will wait", i18n.TimeDurationType)
	ConfigBlobreceiverWorkerCount           = ffc(language.AmericanEnglish, "config.blobreceiver.worker.count", "The number of blob receiver workers", i18n.IntType)

	ConfigBlockchainType = ffc(language.AmericanEnglish, "config.blockchain.type", "A string defining which type of blockchain plugin to use. This tells FireFly which type of configuration to load for the rest of the `blockchain` section.", i18n.StringType)

	ConfigBlockchainEthereumAddressResolverBodyTemplate          = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.bodyTemplate", "The body go template string to use when making HTTP requests", i18n.GoTemplateType)
	ConfigBlockchainEthereumAddressResolverCustomClient          = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.customClient", "Used for testing purposes only", i18n.IgnoredType)
	ConfigBlockchainEthereumAddressResolverExpectContinueTimeout = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.expectContinueTimeout", "See [ExpectContinueTimeout in the Go docs](https://pkg.go.dev/net/http#Transport)", i18n.TimeDurationType)
	ConfigBlockchainEthereumAddressResolverHeaders               = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.headers", "Adds custom headers to HTTP requests", i18n.StringType)
	ConfigBlockchainEthereumAddressResolverIdleTimeout           = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.idleTimeout", "The max duration to hold a HTTP keepalive connection between calls", i18n.TimeDurationType)
	ConfigBlockchainEthereumAddressResolverMaxIdleConns          = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.maxIdleConns", "The max number of idle connections to hold pooled", i18n.IntType)
	ConfigBlockchainEthereumAddressResolverMethod                = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.method", "The HTTP method to use when making requests to the Address Resolver", i18n.StringType)

	ConfigBlockchainEthereumAddressResolverResponseField  = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.responseField", "The name of a JSON field that is provided in the response, that contains the ethereum address (default `address`)", i18n.StringType)
	ConfigBlockchainEthereumAddressResolverRetainOriginal = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.retainOriginal", "When true the original pre-resolved string is retained after the lookup, and passed down to Ethconnect as the from address", i18n.BooleanType)
	ConfigBlockchainEthereumAddressResolverURL            = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.url", "The URL of the Address Resolver", i18n.StringType)
	ConfigBlockchainEthereumAddressResolverURLTemplate    = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.urlTemplate", "The URL Go template string to use when calling the Address Resolver", i18n.GoTemplateType)

	ConfigBlockchainEthereumAddressResolverProxyURL = ffc(language.AmericanEnglish, "config.blockchain.ethereum.addressResolver.proxy.url", "Optional HTTP proxy server to use when connecting to the Address Resolver", "URL "+i18n.StringType)

	ConfigBlockchainEthereumEthconnectBatchSize    = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.batchSize", "The number of events Ethconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream.", i18n.IntType)
	ConfigBlockchainEthereumEthconnectBatchTimeout = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.batchTimeout", "How long Ethconnect should wait for new events to arrive and fill a batch, before sending the batch to FireFly core. Only applies when automatically creating a new event stream.", i18n.TimeDurationType)
	ConfigBlockchainEthereumEthconnectInstance     = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.instance", "The Ethereum address of the FireFly BatchPin smart contract that has been deployed to the blockchain", "Address "+i18n.StringType)
	ConfigBlockchainEthereumEthconnectFromBlock    = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.fromBlock", "The first event this FireFly instance should listen to from the BatchPin smart contract. Default=0. Only affects initial creation of the event stream", "Address "+i18n.StringType)
	ConfigBlockchainEthereumEthconnectPrefixLong   = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.prefixLong", "The prefix that will be used for Ethconnect specific HTTP headers when FireFly makes requests to Ethconnect", i18n.StringType)
	ConfigBlockchainEthereumEthconnectPrefixShort  = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.prefixShort", "The prefix that will be used for Ethconnect specific query parameters when FireFly makes requests to Ethconnect", i18n.StringType)
	ConfigBlockchainEthereumEthconnectTopic        = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single ethconnect", i18n.StringType)
	ConfigBlockchainEthereumEthconnectURL          = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.url", "The URL of the Ethconnect instance", "URL "+i18n.StringType)

	ConfigBlockchainEthereumFFTMURL      = ffc(language.AmericanEnglish, "config.blockchain.ethereum.fftm.url", "The URL of the FireFly Transaction Manager runtime, if enabled", i18n.StringType)
	ConfigBlockchainEthereumFFTMProxyURL = ffc(language.AmericanEnglish, "config.blockchain.ethereum.fftm.proxy.url", "Optional HTTP proxy server to use when connecting to the Transaction Manager", i18n.StringType)

	ConfigBlockchainEthereumEthconnectProxyURL = ffc(language.AmericanEnglish, "config.blockchain.ethereum.ethconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Ethconnect", "URL "+i18n.StringType)

	ConfigBlockchainFabricFabconnectBatchSize    = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.batchSize", "The number of events Fabconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream.", i18n.IntType)
	ConfigBlockchainFabricFabconnectBatchTimeout = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.batchTimeout", "The maximum amount of time to wait for a batch to complete", i18n.TimeDurationType)
	ConfigBlockchainFabricFabconnectChaincode    = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.chaincode", "The name of the Fabric chaincode that FireFly will use for BatchPin transactions", i18n.StringType)
	ConfigBlockchainFabricFabconnectChannel      = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.channel", "The Fabric channel that FireFly will use for BatchPin transactions", i18n.StringType)
	ConfigBlockchainFabricFabconnectPrefixLong   = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.prefixLong", "The prefix that will be used for Fabconnect specific HTTP headers when FireFly makes requests to Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectPrefixShort  = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.prefixShort", "The prefix that will be used for Fabconnect specific query parameters when FireFly makes requests to Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectSigner       = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.signer", "The Fabric signing key to use when submitting transactions to Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectTopic        = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectURL          = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.url", "The URL of the Fabconnect instance", "URL "+i18n.StringType)

	ConfigBlockchainFabricFabconnectProxyURL = ffc(language.AmericanEnglish, "config.blockchain.fabric.fabconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Fabconnect", "URL "+i18n.StringType)

	ConfigBroadcastBatchAgentTimeout = ffc(language.AmericanEnglish, "config.broadcast.batch.agentTimeout", "How long to keep around a batching agent for a sending identity before disposal", i18n.StringType)
	ConfigBroadcastBatchPayloadLimit = ffc(language.AmericanEnglish, "config.broadcast.batch.payloadLimit", "The maximum payload size of a batch for broadcast messages", i18n.ByteSizeType)
	ConfigBroadcastBatchSize         = ffc(language.AmericanEnglish, "config.broadcast.batch.size", "The maximum number of messages that can be packed into a batch", i18n.IntType)
	ConfigBroadcastBatchTimeout      = ffc(language.AmericanEnglish, "config.broadcast.batch.timeout", "The timeout to wait for a batch to fill, before sending", i18n.TimeDurationType)

	ConfigDatabaseType = ffc(language.AmericanEnglish, "config.database.type", "The type of the database interface plugin to use", i18n.IntType)

	ConfigDatabasePostgresMaxConnIdleTime = ffc(language.AmericanEnglish, "config.database.postgres.maxConnIdleTime", "The maximum amount of time a database connection can be idle", i18n.TimeDurationType)
	ConfigDatabasePostgresMaxConnLifetime = ffc(language.AmericanEnglish, "config.database.postgres.maxConnLifetime", "The maximum amount of time to keep a database connection open", i18n.TimeDurationType)
	ConfigDatabasePostgresMaxConns        = ffc(language.AmericanEnglish, "config.database.postgres.maxConns", "Maximum connections to the database", i18n.IntType)
	ConfigDatabasePostgresMaxIdleConns    = ffc(language.AmericanEnglish, "config.database.postgres.maxIdleConns", "The maximum number of idle connections to the database", i18n.IntType)
	ConfigDatabasePostgresURL             = ffc(language.AmericanEnglish, "config.database.postgres.url", "The PostgreSQL connection string for the database", i18n.StringType)

	ConfigDatabaseSqlite3MaxConnIdleTime = ffc(language.AmericanEnglish, "config.database.sqlite3.maxConnIdleTime", "The maximum amount of time a database connection can be idle", i18n.TimeDurationType)
	ConfigDatabaseSqlite3MaxConnLifetime = ffc(language.AmericanEnglish, "config.database.sqlite3.maxConnLifetime", "The maximum amount of time to keep a database connection open", i18n.TimeDurationType)
	ConfigDatabaseSqlite3MaxConns        = ffc(language.AmericanEnglish, "config.database.sqlite3.maxConns", "Maximum connections to the database", i18n.IntType)
	ConfigDatabaseSqlite3MaxIdleConns    = ffc(language.AmericanEnglish, "config.database.sqlite3.maxIdleConns", "The maximum number of idle connections to the database", i18n.IntType)
	ConfigDatabaseSqlite3URL             = ffc(language.AmericanEnglish, "config.database.sqlite3.url", "The SQLite connection string for the database", i18n.StringType)

	ConfigDataexchangeType = ffc(language.AmericanEnglish, "config.dataexchange.type", "The Data Exchange plugin to use", i18n.StringType)

	ConfigDataexchangeFfdxInitEnabled     = ffc(language.AmericanEnglish, "config.dataexchange.ffdx.initEnabled", "Instructs FireFly to always post all current nodes to the `/init` API before connecting or reconnecting to the connector", i18n.BooleanType)
	ConfigDataexchangeFfdxManifestEnabled = ffc(language.AmericanEnglish, "config.dataexchange.ffdx.manifestEnabled", "Determines whether to require+validate a manifest from other DX instances in the network. Must be supported by the connector", i18n.StringType)
	ConfigDataexchangeFfdxURL             = ffc(language.AmericanEnglish, "config.dataexchange.ffdx.url", "The URL of the Data Exchange instance", "URL "+i18n.StringType)

	ConfigDataexchangeFfdxProxyURL = ffc(language.AmericanEnglish, "config.dataexchange.ffdx.proxy.url", "Optional HTTP proxy server to use when connecting to the Data Exchange", "URL "+i18n.StringType)

	ConfigDebugPort = ffc(language.AmericanEnglish, "config.debug.port", "An HTTP port on which to enable the go debugger", i18n.IntType)

	ConfigDownloadWorkerCount       = ffc(language.AmericanEnglish, "config.download.worker.count", "The number of download workers", i18n.IntType)
	ConfigDownloadWorkerQueueLength = ffc(language.AmericanEnglish, "config.download.worker.queueLength", "The length of the work queue in the channel to the workers - defaults to 2x the worker count", i18n.IntType)

	ConfigEventAggregatorBatchSize         = ffc(language.AmericanEnglish, "config.event.aggregator.batchSize", "The maximum number of records to read from the DB before performing an aggregation run", i18n.ByteSizeType)
	ConfigEventAggregatorBatchTimeout      = ffc(language.AmericanEnglish, "config.event.aggregator.batchTimeout", "How long to wait for new events to arrive before performing aggregation on a page of events", i18n.TimeDurationType)
	ConfigEventAggregatorFirstEvent        = ffc(language.AmericanEnglish, "config.event.aggregator.firstEvent", "The first event the aggregator should process, if no previous offest is stored in the DB. Valid options are `oldest` or `newest`", i18n.StringType)
	ConfigEventAggregatorPollTimeout       = ffc(language.AmericanEnglish, "config.event.aggregator.pollTimeout", "The time to wait without a notification of new events, before trying a select on the table", i18n.TimeDurationType)
	ConfigEventAggregatorRewindQueueLength = ffc(language.AmericanEnglish, "config.event.aggregator.rewindQueueLength", "The size of the queue into the rewind dispatcher", i18n.IntType)
	ConfigEventAggregatorRewindTimout      = ffc(language.AmericanEnglish, "config.event.aggregator.rewindTimeout", "The minimum time to wait for rewinds to accumulate before resolving them", i18n.TimeDurationType)
	ConfigEventAggregatorRewindQueryLimit  = ffc(language.AmericanEnglish, "config.event.aggregator.rewindQueryLimit", "Safety limit on the maximum number of records to search when performing queries to search for rewinds", i18n.IntType)
	ConfigEventDbeventsBufferSize          = ffc(language.AmericanEnglish, "config.event.dbevents.bufferSize", "The size of the buffer of change events", i18n.ByteSizeType)

	ConfigEventDispatcherBatchTimeout = ffc(language.AmericanEnglish, "config.event.dispatcher.batchTimeout", "A short time to wait for new events to arrive before re-polling for new events", i18n.TimeDurationType)
	ConfigEventDispatcherBufferLength = ffc(language.AmericanEnglish, "config.event.dispatcher.bufferLength", "The number of events + attachments an individual dispatcher should hold in memory ready for delivery to the subscription", i18n.IntType)
	ConfigEventDispatcherPollTimeout  = ffc(language.AmericanEnglish, "config.event.dispatcher.pollTimeout", "The time to wait without a notification of new events, before trying a select on the table", i18n.TimeDurationType)

	ConfigEventTransportsDefault = ffc(language.AmericanEnglish, "config.event.transports.default", "The default event transport for new subscriptions", i18n.StringType)
	ConfigEventTransportsEnabled = ffc(language.AmericanEnglish, "config.event.transports.enabled", "Which event interface plugins are enabled", i18n.BooleanType)

	ConfigHistogramsMaxChartRows = ffc(language.AmericanEnglish, "config.histograms.maxChartRows", "The maximum rows to fetch for each histogram bucket", i18n.IntType)

	ConfigHTTPAddress      = ffc(language.AmericanEnglish, "config.http.address", "The IP address on which the HTTP API should listen", "IP Address "+i18n.StringType)
	ConfigHTTPPort         = ffc(language.AmericanEnglish, "config.http.port", "The port on which the HTTP API should listen", i18n.IntType)
	ConfigHTTPPublicURL    = ffc(language.AmericanEnglish, "config.http.publicURL", "The fully qualified public URL for the API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.", "URL "+i18n.StringType)
	ConfigHTTPReadTimeout  = ffc(language.AmericanEnglish, "config.http.readTimeout", "The maximum time to wait when reading from an HTTP connection", i18n.TimeDurationType)
	ConfigHTTPWriteTimeout = ffc(language.AmericanEnglish, "config.http.writeTimeout", "The maximum time to wait when writing to an HTTP connection", i18n.TimeDurationType)

	ConfigIdentityType = ffc(language.AmericanEnglish, "config.identity.type", "The Identity plugin to use", i18n.StringType)

	ConfigIdentityManagerCacheLimit = ffc(language.AmericanEnglish, "config.identity.manager.cache.limit", "The identity manager cache limit in count of items", i18n.IntType)

	ConfigLogCompress   = ffc(language.AmericanEnglish, "config.log.compress", "Determines if the rotated log files should be compressed using gzip", i18n.BooleanType)
	ConfigLogFilename   = ffc(language.AmericanEnglish, "config.log.filename", "Filename is the file to write logs to.  Backup log files will be retained in the same directory", i18n.StringType)
	ConfigLogFilesize   = ffc(language.AmericanEnglish, "config.log.filesize", "MaxSize is the maximum size the log file before it gets rotated", i18n.ByteSizeType)
	ConfigLogForceColor = ffc(language.AmericanEnglish, "config.log.forceColor", "Force color to be enabled, even when a non-TTY output is detected", i18n.BooleanType)
	ConfigLogLevel      = ffc(language.AmericanEnglish, "config.log.level", "The log level - error, warn, info, debug, trace", i18n.StringType)
	ConfigLogMaxAge     = ffc(language.AmericanEnglish, "config.log.maxAge", "The maximum time to retain old log files based on the timestamp encoded in their filename.", i18n.TimeDurationType)
	ConfigLogMaxBackups = ffc(language.AmericanEnglish, "config.log.maxBackups", "Maximum number of old log files to retain", i18n.IntType)
	ConfigLogNoColor    = ffc(language.AmericanEnglish, "config.log.noColor", "Force color to be disabled, event when TTY output is detected", i18n.BooleanType)
	ConfigLogTimeFormat = ffc(language.AmericanEnglish, "config.log.timeFormat", "Custom time format for logs", i18n.TimeFormatType)
	ConfigLogUtc        = ffc(language.AmericanEnglish, "config.log.utc", "Use UTC timestamps for logs", i18n.BooleanType)

	ConfigMessageWriterBatchMaxInserts = ffc(language.AmericanEnglish, "config.message.writer.batchMaxInserts", "The maximum number of database inserts to include when writing a single batch of messages + data", i18n.IntType)
	ConfigMessageWriterBatchTimeout    = ffc(language.AmericanEnglish, "config.message.writer.batchTimeout", "How long to wait for more messages to arrive before flushing the batch", i18n.TimeDurationType)
	ConfigMessageWriterCount           = ffc(language.AmericanEnglish, "config.message.writer.count", "The number of message writer workers", i18n.IntType)

	ConfigMetricsAddress      = ffc(language.AmericanEnglish, "config.metrics.address", "The IP address on which the metrics HTTP API should listen", i18n.IntType)
	ConfigMetricsEnabled      = ffc(language.AmericanEnglish, "config.metrics.enabled", "Enables the metrics API", i18n.BooleanType)
	ConfigMetricsPath         = ffc(language.AmericanEnglish, "config.metrics.path", "The path from which to serve the Prometheus metrics", i18n.StringType)
	ConfigMetricsPort         = ffc(language.AmericanEnglish, "config.metrics.port", "The port on which the metrics HTTP API should listen", i18n.IntType)
	ConfigMetricsPublicURL    = ffc(language.AmericanEnglish, "config.metrics.publicURL", "The fully qualified public URL for the metrics API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.", "URL "+i18n.StringType)
	ConfigMetricsReadTimeout  = ffc(language.AmericanEnglish, "config.metrics.readTimeout", "The maximum time to wait when reading from an HTTP connection", i18n.TimeDurationType)
	ConfigMetricsWriteTimeout = ffc(language.AmericanEnglish, "config.metrics.writeTimeout", "The maximum time to wait when writing to an HTTP connection", i18n.TimeDurationType)

	ConfigNamespacesDefault               = ffc(language.AmericanEnglish, "config.namespaces.default", "The default namespace - must be in the predefined list", i18n.StringType)
	ConfigNamespacesPredefined            = ffc(language.AmericanEnglish, "config.namespaces.predefined", "A list of namespaces to ensure exists, without requiring a broadcast from the network", "List "+i18n.StringType)
	ConfigNamespacesPredefinedName        = ffc(language.AmericanEnglish, "config.namespaces.predefined[].name", "The name of the namespace (must be unique)", i18n.StringType)
	ConfigNamespacesPredefinedDescription = ffc(language.AmericanEnglish, "config.namespaces.predefined[].description", "A description for the namespace", i18n.StringType)

	ConfigNodeDescription = ffc(language.AmericanEnglish, "config.node.description", "The description of this FireFly node", i18n.StringType)
	ConfigNodeName        = ffc(language.AmericanEnglish, "config.node.name", "The name of this FireFly node", i18n.StringType)

	ConfigOpupdateWorkerBatchMaxInserts = ffc(language.AmericanEnglish, "config.opupdate.worker.batchMaxInserts", "The maximum number of database inserts to include when writing a single batch of messages + data", i18n.IntType)
	ConfigOpupdateWorkerBatchTimeout    = ffc(language.AmericanEnglish, "config.opupdate.worker.batchTimeout", "How long to wait for more messages to arrive before flushing the batch", i18n.TimeDurationType)
	ConfigOpupdateWorkerCount           = ffc(language.AmericanEnglish, "config.opupdate.worker.count", "The number of operation update works", i18n.IntType)
	ConfigOpupdateWorkerQueueLength     = ffc(language.AmericanEnglish, "config.opupdate.worker.queueLength", "The size of the queue for the Operation Update worker", i18n.IntType)

	ConfigOrchestratorStartupAttempts = ffc(language.AmericanEnglish, "config.orchestrator.startupAttempts", "The number of times to attempt to connect to core infrastructure on startup", i18n.StringType)

	ConfigOrgDescription = ffc(language.AmericanEnglish, "config.org.description", "A description of the organization to which this FireFly node belongs", i18n.StringType)
	ConfigOrgIdentity    = ffc(language.AmericanEnglish, "config.org.identity", "`DEPRECATED` Please use `org.key` instead", i18n.StringType)
	ConfigOrgKey         = ffc(language.AmericanEnglish, "config.org.key", "The signing identity allocated to the organization (can be the same as the nodes)", i18n.StringType)
	ConfigOrgName        = ffc(language.AmericanEnglish, "config.org.name", "The name of the organization to which this FireFly node belongs", i18n.StringType)

	ConfigPrivatemessagingBatchAgentTimeout = ffc(language.AmericanEnglish, "config.privatemessaging.batch.agentTimeout", "How long to keep around a batching agent for a sending identity before disposal", i18n.TimeDurationType)
	ConfigPrivatemessagingBatchPayloadLimit = ffc(language.AmericanEnglish, "config.privatemessaging.batch.payloadLimit", "The maximum payload size of a private message Data Exchange payload", i18n.ByteSizeType)
	ConfigPrivatemessagingBatchSize         = ffc(language.AmericanEnglish, "config.privatemessaging.batch.size", "The maximum number of messages in a batch for private messages", i18n.IntType)
	ConfigPrivatemessagingBatchTimeout      = ffc(language.AmericanEnglish, "config.privatemessaging.batch.timeout", "The timeout to wait for a batch to fill, before sending", i18n.TimeDurationType)

	ConfigPublicstorageType = ffc(language.AmericanEnglish, "config.publicstorage.type", "`DEPRECATED` Please use `config.sharedstorage.type` instead", i18n.StringType)

	ConfigPublicstorageIpfsAPIURL          = ffc(language.AmericanEnglish, "config.publicstorage.ipfs.api.url", "The URL for the IPFS API", "URL "+i18n.StringType)
	ConfigPublicstorageIpfsAPIProxyURL     = ffc(language.AmericanEnglish, "config.publicstorage.ipfs.api.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS API", "URL "+i18n.StringType)
	ConfigPublicstorageIpfsGatewayURL      = ffc(language.AmericanEnglish, "config.publicstorage.ipfs.gateway.url", "The URL for the IPFS Gateway", "URL "+i18n.StringType)
	ConfigPublicstorageIpfsGatewayProxyURL = ffc(language.AmericanEnglish, "config.publicstorage.ipfs.gateway.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS Gateway", "URL "+i18n.StringType)
	ConfigSharedstorageType                = ffc(language.AmericanEnglish, "config.sharedstorage.type", "The Shared Storage plugin to use", i18n.StringType)
	ConfigSharedstorageIpfsAPIURL          = ffc(language.AmericanEnglish, "config.sharedstorage.ipfs.api.url", "The URL for the IPFS API", "URL "+i18n.StringType)
	ConfigSharedstorageIpfsAPIProxyURL     = ffc(language.AmericanEnglish, "config.sharedstorage.ipfs.api.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS API", "URL "+i18n.StringType)
	ConfigSharedstorageIpfsGatewayURL      = ffc(language.AmericanEnglish, "config.sharedstorage.ipfs.gateway.url", "The URL for the IPFS Gateway", "URL "+i18n.StringType)
	ConfigSharedstorageIpfsGatewayProxyURL = ffc(language.AmericanEnglish, "config.sharedstorage.ipfs.gateway.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS Gateway", "URL "+i18n.StringType)

	ConfigSubscriptionMax               = ffc(language.AmericanEnglish, "config.subscription.max", "The maximum number of pre-defined subscriptions that can exist (note for high fan-out consider connecting a dedicated pub/sub broker to the dispatcher)", i18n.IntType)
	ConfigSubscriptionDefaultsBatchSize = ffc(language.AmericanEnglish, "config.subscription.defaults.batchSize", "Default read ahead to enable for subscriptions that do not explicitly configure readahead", i18n.IntType)

	ConfigTokensConnector = ffc(language.AmericanEnglish, "config.tokens[].connector", "The name of the Tokens Connector. This will be used in the FireFly API path to refer to this specific Token Connector", i18n.StringType)
	ConfigTokensName      = ffc(language.AmericanEnglish, "config.tokens[].name", "The name of the Tokens Connector. This will be used in the FireFly API path to refer to this specific Token Connector", i18n.StringType)
	ConfigTokensPlugin    = ffc(language.AmericanEnglish, "config.tokens[].plugin", "The name of the Tokens Connector plugin to use", i18n.StringType)
	ConfigTokensURL       = ffc(language.AmericanEnglish, "config.tokens[].url", "The URL of the Token Connector", "URL "+i18n.StringType)

	ConfigTokensProxyURL = ffc(language.AmericanEnglish, "config.tokens[].proxy.url", "Optional HTTP proxy server to use when connecting to the Token Connector", "URL "+i18n.StringType)

	ConfigUIEnabled = ffc(language.AmericanEnglish, "config.ui.enabled", "Enables the web user interface", i18n.BooleanType)
	ConfigUIPath    = ffc(language.AmericanEnglish, "config.ui.path", "The file system path which contains the static HTML, CSS, and JavaScript files for the user interface", i18n.StringType)

	ConfigAPIOASPanicOnMissingDescription = ffc(language.AmericanEnglish, "config.api.oas.panicOnMissingDescription", "Used for testing purposes only", i18n.IgnoredType)

	ConfigAdminWebSocketBlockedWarnInternal = ffc(language.AmericanEnglish, "config.admin.ws.blockedWarnInterval", "How often to log warnings in core, when an admin change event listener falls behind the stream they requested and misses events", i18n.TimeDurationType)
	ConfigAdminWebSocketEventQueueLength    = ffc(language.AmericanEnglish, "config.admin.ws.eventQueueLength", "Server-side queue length for events waiting for delivery over an admin change event listener websocket", i18n.IntType)
)
