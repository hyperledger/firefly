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
	"github.com/hyperledger/firefly/pkg/i18n"
)

var ffc = i18n.FFC
var timeDurationType = "[`time.Duration`](https://pkg.go.dev/time#Duration)"
var byteSizeType = "[`BytesSize`](https://pkg.go.dev/github.com/docker/go-units#BytesSize)"
var goTemplateType = "[Go Template](https://pkg.go.dev/text/template) `string`"
var stringType = "`string`"
var intType = "`int`"
var booleanType = "`boolean`"
var floatType = "`boolean`"
var mapStringStringType = "`map[string]string`"
var IgnoredType = "IGNORE"

//revive:disable
var (
	ConfigGlobalConnectionTimeout = ffc("config.global.connectionTimeout", "The maximum amount of time that a connection is allowed to remain with no data transmitted", timeDurationType)
	ConfigGlobalRequestTimeout    = ffc("config.global.requestTimeout", "The maximum amount of time that a request is allowed to remain open", timeDurationType)

	ConfigGlobalRetryEnabled      = ffc("config.global.retry.enabled", "Enables retries", booleanType)
	ConfigGlobalRetryFactor       = ffc("config.global.retry.factor", "The retry backoff factor", floatType)
	ConfigGlobalRetryInitDelay    = ffc("config.global.retry.initDelay", "The initial retry delay", timeDurationType)
	ConfigGlobalRetryInitialDelay = ffc("config.global.retry.initialDelay", "The initial retry delay", timeDurationType)
	ConfigGlobalRetryMaxDelay     = ffc("config.global.retry.maxDelay", "The maximum retry delay", timeDurationType)
	ConfigGlobalRetryMaxAttempts  = ffc("config.global.retry.maxAttempts", "The maximum number attempts", intType)
	ConfigGlobalRetryCount        = ffc("config.global.retry.count", "The maximum number of times to retry", intType)
	ConfigGlobalInitWaitTime      = ffc("config.global.retry.initWaitTime", "The initial retry delay", timeDurationType)
	ConfigGlobalMaxWaitTime       = ffc("config.global.retry.maxWaitTime", "The maximum retry delay", timeDurationType)

	ConfigGlobalUsername = ffc("config.global.auth.username", "Username", stringType)
	ConfigGlobalPassword = ffc("config.global.auth.password", "Password", stringType)

	ConfigGlobalSize = ffc("config.global.cache.size", "The size of the cache", byteSizeType)
	ConfigGlobalTTL  = ffc("config.global.cache.ttl", "The time to live (TTL) for the cache", timeDurationType)

	ConfigGlobaltWsHeartbeatInterval     = ffc("config.global.ws.heartbeatInterval", "The amount of time to wait between heartbeat signals on the WebSocket connection", timeDurationType)
	ConfigGlobalWsInitialConnectAttempts = ffc("config.global.ws.initialConnectAttempts", "The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing", intType)
	ConfigGlobalWsPath                   = ffc("config.global.ws.path", "The WebSocket sever URL to which FireFly should connect", "WebSocket URL "+stringType)
	ConfigGlobalWsReadBufferSize         = ffc("config.global.ws.readBufferSize", "The size in bytes of the read buffer for the WebSocket connection", byteSizeType)
	ConfigGlobalWsWriteBufferSize        = ffc("config.global.ws.writeBufferSize", "The size in bytes of the write buffer for the WebSocket connection", byteSizeType)

	ConfigGlobalTLSCaFile           = ffc("config.global.tls.caFile", "The path to the CA file for TLS on this API", stringType)
	ConfigGlobalTLSCertFile         = ffc("config.global.tls.certFile", "The path to the certificate file for TLS on this API", stringType)
	ConfigGlobalTLSClientAuth       = ffc("config.global.tls.clientAuth", "Enables or disables client auth for TLS on this API", stringType)
	ConfigGlobalTLSEnabled          = ffc("config.global.tls.enabled", "Enables or disables TLS on this API", booleanType)
	ConfigGlobalTLSKeyFile          = ffc("config.global.tls.keyFile", "The path to the private key file for TLS on this API", stringType)
	ConfigGlobalTLSHandshakeTimeout = ffc("config.global.tlsHandshakeTimeout", "The maximum amount of time to wait for a successful TLS handshake", timeDurationType)

	ConfigGlobalBodyTemplate          = ffc("config.global.bodyTemplate", "The body go template string to use when making HTTP requests", goTemplateType)
	ConfigGlobalCustomClient          = ffc("config.global.customClient", "Used for testing purposes only", IgnoredType)
	ConfigGlobalExpectContinueTimeout = ffc("config.global.expectContinueTimeout", "See [ExpectContinueTimeout in the Go docs](https://pkg.go.dev/net/http#Transport)", timeDurationType)
	ConfigGlobalHeaders               = ffc("config.global.headers", "Adds custom headers to HTTP requests", mapStringStringType)
	ConfigGlobalIdleTimeout           = ffc("config.global.idleTimeout", "The max duration to hold a HTTP keepalive connection between calls", timeDurationType)
	ConfigGlobalMaxIdleConns          = ffc("config.global.maxIdleConns", "The max number of idle connections to hold pooled", intType)
	ConfigGlobalMethod                = ffc("config.global.method", "The HTTP method to use when making requests to the Address Resolver", stringType)

	ConfigGlobalMigrationsAuto      = ffc("config.global.migrations.auto", "Enables automatic database migrations", booleanType)
	ConfigGlobalMigrationsDirectory = ffc("config.global.migrations.directory", "The directory containing the numerically ordered migration DDL files to apply to the database", stringType)

	ConfigAdminAddress      = ffc("config.admin.address", "The IP address on which the admin HTTP API should listen", "IP Address "+stringType)
	ConfigAdminEnabled      = ffc("config.admin.enabled", "Enables the admin HTTP API", booleanType)
	ConfigAdminPort         = ffc("config.admin.port", "The port on which the admin HTTP API should listen", intType)
	ConfigAdminPreInit      = ffc("config.admin.preinit", "Enables the pre-init mode. This mode will let the FireFly Core process start, but not initialize any plugins, besides the database to read any configuration overrides. This allows the admin HTTP API to be used to define custom configuration before starting the rest of FireFly Core.", booleanType)
	ConfigAdminPublicURL    = ffc("config.admin.publicURL", "The fully qualified public URL for the admin API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.", "URL "+stringType)
	ConfigAdminReadTimeout  = ffc("config.admin.readTimeout", "The maximum time to wait when reading from an HTTP connection", timeDurationType)
	ConfigAdminWriteTimeout = ffc("config.admin.writeTimeout", "The maximum time to wait when writing to an HTTP connection", timeDurationType)

	ConfigAPIDefaultFilterLimit = ffc("config.api.defaultFilterLimit", "The maximum number of rows to return if no limit is specified on an API request", intType)
	ConfigAPIMaxFilterLimit     = ffc("config.api.maxFilterLimit", "The largest value of `limit` that an HTTP client can specify in a request", intType)
	ConfigAPIRequestMaxTimeout  = ffc("config.api.requestMaxTimeout", "The maximum amount of time that an HTTP client can specify in a `Request-Timeout` header to keep a specific request open", timeDurationType)

	ConfigAPIShutdownTimeout = ffc("config.api.shutdownTimeout", "The maximum amount of time to wait for any open HTTP requests to finish before shutting down the HTTP server", timeDurationType)

	ConfigAssetManagerKeyNormalization = ffc("config.asset.manager.keyNormalization", "Mechanism to normalize keys before using them. Valid options are `blockchain_plugin` - use blockchain plugin (default) or `none` - do not attempt normalization", stringType)

	ConfigBatchManagerMinimumPollDelay = ffc("config.batch.manager.minimumPollDelay", "The minimum time the batch manager waits between polls on the DB - to prevent thrashing", timeDurationType)
	ConfigBatchManagerPollTimeout      = ffc("config.batch.manager.pollTimeout", "How long to wait without any notifications of new messages before doing a page query", timeDurationType)
	ConfigBatchManagerReadPageSize     = ffc("config.batch.manager.readPageSize", "The size of each page of messages read from the database into memory when assembling batches", intType)

	ConfigBlobreceiverWorkerBatchMaxInserts = ffc("config.blobreceiver.worker.batchMaxInserts", "The maximum number of items the blob receiver worker will insert in a batch", intType)
	ConfigBlobreceiverWorkerBatchTimeout    = ffc("config.blobreceiver.worker.batchTimeout", "The maximum amount of the the blob receiver worker will wait", timeDurationType)
	ConfigBlobreceiverWorkerCount           = ffc("config.blobreceiver.worker.count", "The number of blob receiver workers", intType)

	ConfigBlockchainType = ffc("config.blockchain.type", "A string defining which type of blockchain plugin to use. This tells FireFly which type of configuration to load for the rest of the `blockchain` section.", stringType)

	ConfigBlockchainEthereumAddressResolverBodyTemplate          = ffc("config.blockchain.ethereum.addressResolver.bodyTemplate", "The body go template string to use when making HTTP requests", goTemplateType)
	ConfigBlockchainEthereumAddressResolverCustomClient          = ffc("config.blockchain.ethereum.addressResolver.customClient", "Used for testing purposes only", IgnoredType)
	ConfigBlockchainEthereumAddressResolverExpectContinueTimeout = ffc("config.blockchain.ethereum.addressResolver.expectContinueTimeout", "See [ExpectContinueTimeout in the Go docs](https://pkg.go.dev/net/http#Transport)", timeDurationType)
	ConfigBlockchainEthereumAddressResolverHeaders               = ffc("config.blockchain.ethereum.addressResolver.headers", "Adds custom headers to HTTP requests", stringType)
	ConfigBlockchainEthereumAddressResolverIdleTimeout           = ffc("config.blockchain.ethereum.addressResolver.idleTimeout", "The max duration to hold a HTTP keepalive connection between calls", timeDurationType)
	ConfigBlockchainEthereumAddressResolverMaxIdleConns          = ffc("config.blockchain.ethereum.addressResolver.maxIdleConns", "The max number of idle connections to hold pooled", intType)
	ConfigBlockchainEthereumAddressResolverMethod                = ffc("config.blockchain.ethereum.addressResolver.method", "The HTTP method to use when making requests to the Address Resolver", stringType)

	ConfigBlockchainEthereumAddressResolverResponseField  = ffc("config.blockchain.ethereum.addressResolver.responseField", "The name of a JSON field that is provided in the response, that contains the ethereum address (default `address`)", stringType)
	ConfigBlockchainEthereumAddressResolverRetainOriginal = ffc("config.blockchain.ethereum.addressResolver.retainOriginal", "When true the original pre-resolved string is retained after the lookup, and passed down to Ethconnect as the from address", booleanType)
	ConfigBlockchainEthereumAddressResolverURL            = ffc("config.blockchain.ethereum.addressResolver.url", "The URL of the Address Resolver", stringType)
	ConfigBlockchainEthereumAddressResolverURLTemplate    = ffc("config.blockchain.ethereum.addressResolver.urlTemplate", "The URL Go template string to use when calling the Address Resolver", goTemplateType)

	ConfigBlockchainEthereumAddressResolverProxyURL = ffc("config.blockchain.ethereum.addressResolver.proxy.url", "Optional HTTP proxy server to use when connecting to the Address Resolver", "URL "+stringType)

	ConfigBlockchainEthereumEthconnectBatchSize    = ffc("config.blockchain.ethereum.ethconnect.batchSize", "The number of events Ethconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream.", intType)
	ConfigBlockchainEthereumEthconnectBatchTimeout = ffc("config.blockchain.ethereum.ethconnect.batchTimeout", "How long Ethconnect should wait for new events to arrive and fill a batch, before sending the batch to FireFly core. Only applies when automatically creating a new event stream.", timeDurationType)
	ConfigBlockchainEthereumEthconnectInstance     = ffc("config.blockchain.ethereum.ethconnect.instance", "The Ethereum address of the FireFly BatchPin smart contract that has been deployed to the blockchain", "Address "+stringType)
	ConfigBlockchainEthereumEthconnectPrefixLong   = ffc("config.blockchain.ethereum.ethconnect.prefixLong", "The prefix that will be used for Ethconnect specific HTTP headers when FireFly makes requests to Ethconnect", stringType)
	ConfigBlockchainEthereumEthconnectPrefixShort  = ffc("config.blockchain.ethereum.ethconnect.prefixShort", "The prefix that will be used for Ethconnect specific query parameters when FireFly makes requests to Ethconnect", stringType)
	ConfigBlockchainEthereumEthconnectTopic        = ffc("config.blockchain.ethereum.ethconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single ethconnect", stringType)
	ConfigBlockchainEthereumEthconnectURL          = ffc("config.blockchain.ethereum.ethconnect.url", "The URL of the Ethconnect instance", "URL "+stringType)

	ConfigBlockchainEthereumEthconnectProxyURL = ffc("config.blockchain.ethereum.ethconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Ethconnect", "URL "+stringType)

	ConfigBlockchainFabricFabconnectBatchSize    = ffc("config.blockchain.fabric.fabconnect.batchSize", "The number of events Fabconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream.", intType)
	ConfigBlockchainFabricFabconnectBatchTimeout = ffc("config.blockchain.fabric.fabconnect.batchTimeout", "The maximum amount of time to wait for a batch to complete", timeDurationType)
	ConfigBlockchainFabricFabconnectChaincode    = ffc("config.blockchain.fabric.fabconnect.chaincode", "The name of the Fabric chaincode that FireFly will use for BatchPin transactions", stringType)
	ConfigBlockchainFabricFabconnectChannel      = ffc("config.blockchain.fabric.fabconnect.channel", "The Fabric channel that FireFly will use for BatchPin transactions", stringType)
	ConfigBlockchainFabricFabconnectPrefixLong   = ffc("config.blockchain.fabric.fabconnect.prefixLong", "The prefix that will be used for Fabconnect specific HTTP headers when FireFly makes requests to Fabconnect", stringType)
	ConfigBlockchainFabricFabconnectPrefixShort  = ffc("config.blockchain.fabric.fabconnect.prefixShort", "The prefix that will be used for Fabconnect specific query parameters when FireFly makes requests to Fabconnect", stringType)
	ConfigBlockchainFabricFabconnectSigner       = ffc("config.blockchain.fabric.fabconnect.signer", "The Fabric signing key to use when submitting transactions to Fabconnect", stringType)
	ConfigBlockchainFabricFabconnectTopic        = ffc("config.blockchain.fabric.fabconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single Fabconnect", stringType)
	ConfigBlockchainFabricFabconnectURL          = ffc("config.blockchain.fabric.fabconnect.url", "The URL of the Fabconnect instance", "URL "+stringType)

	ConfigBlockchainFabricFabconnectProxyURL = ffc("config.blockchain.fabric.fabconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Fabconnect", "URL "+stringType)

	ConfigBroadcastBatchAgentTimeout = ffc("config.broadcast.batch.agentTimeout", "How long to keep around a batching agent for a sending identity before disposal", stringType)
	ConfigBroadcastBatchPayloadLimit = ffc("config.broadcast.batch.payloadLimit", "The maximum payload size of a batch for broadcast messages", byteSizeType)
	ConfigBroadcastBatchSize         = ffc("config.broadcast.batch.size", "The maximum number of messages that can be packed into a batch", intType)
	ConfigBroadcastBatchTimeout      = ffc("config.broadcast.batch.timeout", "The timeout to wait for a batch to fill, before sending", timeDurationType)

	ConfigCorsCredentials = ffc("config.cors.credentials", "CORS setting to control whether a browser allows credentials to be sent to this API", booleanType)
	ConfigCorsDebug       = ffc("config.cors.debug", "Whether debug is enabled for the CORS implementation", booleanType)
	ConfigCorsEnabled     = ffc("config.cors.enabled", "Whether CORS is enabled", booleanType)
	ConfigCorsHeaders     = ffc("config.cors.headers", "CORS setting to control the allowed headers", stringType)
	ConfigCorsMaxAge      = ffc("config.cors.maxAge", "The maximum age a browser should rely on CORS checks", timeDurationType)
	ConfigCorsMethods     = ffc("config.cors.methods", " CORS setting to control the allowed methods", stringType)
	ConfigCorsOrigins     = ffc("config.cors.origins", "CORS setting to control the allowed origins", stringType)

	ConfigDatabaseMaxChartRows = ffc("config.database.maxChartRows", "The maximum rows to fetch for each histogram bucket", intType)
	ConfigDatabaseType         = ffc("config.database.type", "The type of the database interface plugin to use", intType)

	ConfigDatabasePostgresMaxConnIdleTime = ffc("config.database.postgres.maxConnIdleTime", "The maximum amount of time a database connection can be idle", timeDurationType)
	ConfigDatabasePostgresMaxConnLifetime = ffc("config.database.postgres.maxConnLifetime", "The maximum amount of time to keep a database connection open", timeDurationType)
	ConfigDatabasePostgresMaxConns        = ffc("config.database.postgres.maxConns", "Maximum connections to the database", intType)
	ConfigDatabasePostgresMaxIdleConns    = ffc("config.database.postgres.maxIdleConns", "The maximum number of idle connections to the database", intType)
	ConfigDatabasePostgresURL             = ffc("config.database.postgres.url", "The PostgreSQL connection string for the database", stringType)

	ConfigDatabaseSqlite3MaxConnIdleTime = ffc("config.database.sqlite3.maxConnIdleTime", "The maximum amount of time a database connection can be idle", timeDurationType)
	ConfigDatabaseSqlite3MaxConnLifetime = ffc("config.database.sqlite3.maxConnLifetime", "The maximum amount of time to keep a database connection open", timeDurationType)
	ConfigDatabaseSqlite3MaxConns        = ffc("config.database.sqlite3.maxConns", "Maximum connections to the database", intType)
	ConfigDatabaseSqlite3MaxIdleConns    = ffc("config.database.sqlite3.maxIdleConns", "The maximum number of idle connections to the database", intType)
	ConfigDatabaseSqlite3URL             = ffc("config.database.sqlite3.url", "The SQLite connection string for the database", stringType)

	ConfigDataexchangeType = ffc("config.dataexchange.type", "The Data Exchange plugin to use", stringType)

	ConfigDataexchangeFfdxInitEnabled     = ffc("config.dataexchange.ffdx.initEnabled", "Instructs FireFly to always post all current nodes to the `/init` API before connecting or reconnecting to the connector", booleanType)
	ConfigDataexchangeFfdxManifestEnabled = ffc("config.dataexchange.ffdx.manifestEnabled", "Determines whether to require+validate a manifest from other DX instances in the network. Must be supported by the connector", stringType)
	ConfigDataexchangeFfdxURL             = ffc("config.dataexchange.ffdx.url", "The URL of the Data Exchange instance", "URL "+stringType)

	ConfigDataexchangeFfdxProxyURL = ffc("config.dataexchange.ffdx.proxy.url", "Optional HTTP proxy server to use when connecting to the Data Exchange", "URL "+stringType)

	ConfigDebugPort = ffc("config.debug.port", "An HTTP port on which to enable the go debugger", intType)

	ConfigDownloadWorkerCount       = ffc("config.download.worker.count", "The number of download workers", intType)
	ConfigDownloadWorkerQueueLength = ffc("config.download.worker.queueLength", "The length of the work queue in the channel to the workers - defaults to 2x the worker count", intType)

	ConfigEventAggregatorBatchSize            = ffc("config.event.aggregator.batchSize", "The maximum number of records to read from the DB before performing an aggregation run", byteSizeType)
	ConfigEventAggregatorBatchTimeout         = ffc("config.event.aggregator.batchTimeout", "How long to wait for new events to arrive before performing aggregation on a page of events", timeDurationType)
	ConfigEventAggregatorFirstEvent           = ffc("config.event.aggregator.firstEvent", "The first event the aggregator should process, if no previous offest is stored in the DB. Valid options are `oldest` or `newest`", stringType)
	ConfigEventAggregatorOpCorrelationRetries = ffc("config.event.aggregator.opCorrelationRetries", "How many times to correlate an event for an operation (such as tx submission) back to an operation. Needed because the operation update might come back before we are finished persisting the ID of the request", intType)
	ConfigEventAggregatorPollTimeout          = ffc("config.event.aggregator.pollTimeout", "The time to wait without a notification of new events, before trying a select on the table", timeDurationType)
	ConfigEventAggregatorRewindQueueLength    = ffc("config.event.aggregator.rewindQueueLength", "The size of the queue into the rewind dispatcher", intType)
	ConfigEventAggregatorRewindTimout         = ffc("config.event.aggregator.rewindTimeout", "The minimum time to wait for rewinds to accumulate before resolving them", timeDurationType)

	ConfigEventDbeventsBufferSize = ffc("config.event.dbevents.bufferSize", "The size of the buffer of change events", byteSizeType)

	ConfigEventDispatcherBatchTimeout = ffc("config.event.dispatcher.batchTimeout", "A short time to wait for new events to arrive before re-polling for new events", timeDurationType)
	ConfigEventDispatcherBufferLength = ffc("config.event.dispatcher.bufferLength", "The number of events + attachments an individual dispatcher should hold in memory ready for delivery to the subscription", intType)
	ConfigEventDispatcherPollTimeout  = ffc("config.event.dispatcher.pollTimeout", "The time to wait without a notification of new events, before trying a select on the table", timeDurationType)

	ConfigEventTransportsDefault = ffc("config.event.transports.default", "The default event transport for new subscriptions", stringType)
	ConfigEventTransportsEnabled = ffc("config.event.transports.enabled", "Which event interface plugins are enabled", booleanType)

	ConfigHTTPAddress      = ffc("config.http.address", "The IP address on which the HTTP API should listen", "IP Address "+stringType)
	ConfigHTTPPort         = ffc("config.http.port", "The port on which the HTTP API should listen", intType)
	ConfigHTTPPublicURL    = ffc("config.http.publicURL", "The fully qualified public URL for the API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.", "URL "+stringType)
	ConfigHTTPReadTimeout  = ffc("config.http.readTimeout", "The maximum time to wait when reading from an HTTP connection", timeDurationType)
	ConfigHTTPWriteTimeout = ffc("config.http.writeTimeout", "The maximum time to wait when writing to an HTTP connection", timeDurationType)

	ConfigIdentityType = ffc("config.identity.type", "The Identity plugin to use", stringType)

	ConfigIdentityManagerCacheLimit = ffc("config.identity.manager.cache.limit", "The identity manager cache limit in count of items", intType)

	ConfigLogCompress   = ffc("config.log.compress", "Sets whether to compress log backups", booleanType)
	ConfigLogFilename   = ffc("config.log.filename", "The filename to which logs will be written", stringType)
	ConfigLogFilesize   = ffc("config.log.filesize", "The size at which to roll log files", byteSizeType)
	ConfigLogForceColor = ffc("config.log.forceColor", "Forces colored log output to be enabled, even if a TTY is not detected", booleanType)
	ConfigLogLevel      = ffc("config.log.level", "Sets the log level. Valid options are `ERROR`, `INFO`, `DEBUG`, `TRACE`", "Log level "+stringType)
	ConfigLogMaxAge     = ffc("config.log.maxAge", "The maximum age at which log files will be rolled", timeDurationType)
	ConfigLogMaxBackups = ffc("config.log.maxBackups", "The maximum number of old log files to keep", intType)
	ConfigLogNoColor    = ffc("config.log.noColor", "Forces colored log output to be disabled, even if a TTY is detected", booleanType)
	ConfigLogTimeFormat = ffc("config.log.timeFormat", "A string format for timestamps", stringType)
	ConfigLogUtc        = ffc("config.log.utc", "Sets log timestamps to the UTC timezone", booleanType)

	ConfigMessageWriterBatchMaxInserts = ffc("config.message.writer.batchMaxInserts", "The maximum number of database inserts to include when writing a single batch of messages + data", intType)
	ConfigMessageWriterBatchTimeout    = ffc("config.message.writer.batchTimeout", "How long to wait for more messages to arrive before flushing the batch", timeDurationType)
	ConfigMessageWriterCount           = ffc("config.message.writer.count", "The number of message writer workers", intType)

	ConfigMetricsAddress      = ffc("config.metrics.address", "The IP address on which the metrics HTTP API should listen", intType)
	ConfigMetricsEnabled      = ffc("config.metrics.enabled", "Enables the metrics API", booleanType)
	ConfigMetricsPath         = ffc("config.metrics.path", "The path from which to serve the Prometheus metrics", stringType)
	ConfigMetricsPort         = ffc("config.metrics.port", "The port on which the metrics HTTP API should listen", intType)
	ConfigMetricsPublicURL    = ffc("config.metrics.publicURL", "The fully qualified public URL for the metrics API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.", "URL "+stringType)
	ConfigMetricsReadTimeout  = ffc("config.metrics.readTimeout", "The maximum time to wait when reading from an HTTP connection", timeDurationType)
	ConfigMetricsWriteTimeout = ffc("config.metrics.writeTimeout", "The maximum time to wait when writing to an HTTP connection", timeDurationType)

	ConfigNamespacesDefault    = ffc("config.namespaces.default", "The default namespace - must be in the predefined list", stringType)
	ConfigNamespacesPredefined = ffc("config.namespaces.predefined", "A list of namespaces to ensure exists, without requiring a broadcast from the network", "List "+stringType)

	ConfigNodeDescription = ffc("config.node.description", "The description of this FireFly node", stringType)
	ConfigNodeName        = ffc("config.node.name", "The name of this FireFly node", stringType)

	ConfigOpupdateWorkerBatchMaxInserts = ffc("config.opupdate.worker.batchMaxInserts", "The maximum number of database inserts to include when writing a single batch of messages + data", intType)
	ConfigOpupdateWorkerBatchTimeout    = ffc("config.opupdate.worker.batchTimeout", "How long to wait for more messages to arrive before flushing the batch", timeDurationType)
	ConfigOpupdateWorkerCount           = ffc("config.opupdate.worker.count", "The number of operation update works", intType)
	ConfigOpupdateWorkerQueueLength     = ffc("config.opupdate.worker.queueLength", "The size of the queue for the Operation Update worker", intType)

	ConfigOrchestratorStartupAttempts = ffc("config.orchestrator.startupAttempts", "The number of times to attempt to connect to core infrastructure on startup", stringType)

	ConfigOrgDescription = ffc("config.org.description", "A description of the organization to which this FireFly node belongs", stringType)
	ConfigOrgIdentity    = ffc("config.org.identity", "`DEPRECATED` Please use `org.key` instead", stringType)
	ConfigOrgKey         = ffc("config.org.key", "The signing identity allocated to the organization (can be the same as the nodes)", stringType)
	ConfigOrgName        = ffc("config.org.name", "The name of the organization to which this FireFly node belongs", stringType)

	ConfigPrivatemessagingOpCorrelationRetries = ffc("config.privatemessaging.opCorrelationRetries", "How many times to correlate an event for an operation (such as tx submission) back to an operation. Needed because the operation update might come back before we are finished persisting the ID of the request", intType)

	ConfigPrivatemessagingBatchAgentTimeout = ffc("config.privatemessaging.batch.agentTimeout", "How long to keep around a batching agent for a sending identity before disposal", timeDurationType)
	ConfigPrivatemessagingBatchPayloadLimit = ffc("config.privatemessaging.batch.payloadLimit", "The maximum payload size of a private message Data Exchange payload", byteSizeType)
	ConfigPrivatemessagingBatchSize         = ffc("config.privatemessaging.batch.size", "The maximum number of messages in a batch for private messages", intType)
	ConfigPrivatemessagingBatchTimeout      = ffc("config.privatemessaging.batch.timeout", "The timeout to wait for a batch to fill, before sending", timeDurationType)

	ConfigPublicstorageType = ffc("config.publicstorage.type", "`DEPRECATED` Please use `config.sharedstorage.type` instead", stringType)

	ConfigPublicstorageIpfsAPIURL          = ffc("config.publicstorage.ipfs.api.url", "The URL for the IPFS API", "URL "+stringType)
	ConfigPublicstorageIpfsAPIProxyURL     = ffc("config.publicstorage.ipfs.api.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS API", "URL "+stringType)
	ConfigPublicstorageIpfsGatewayURL      = ffc("config.publicstorage.ipfs.gateway.url", "The URL for the IPFS Gateway", "URL "+stringType)
	ConfigPublicstorageIpfsGatewayProxyURL = ffc("config.publicstorage.ipfs.gateway.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS Gateway", "URL "+stringType)
	ConfigSharedstorageType                = ffc("config.sharedstorage.type", "The Shared Storage plugin to use", stringType)
	ConfigSharedstorageIpfsAPIURL          = ffc("config.sharedstorage.ipfs.api.url", "The URL for the IPFS API", "URL "+stringType)
	ConfigSharedstorageIpfsAPIProxyURL     = ffc("config.sharedstorage.ipfs.api.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS API", "URL "+stringType)
	ConfigSharedstorageIpfsGatewayURL      = ffc("config.sharedstorage.ipfs.gateway.url", "The URL for the IPFS Gateway", "URL "+stringType)
	ConfigSharedstorageIpfsGatewayProxyURL = ffc("config.sharedstorage.ipfs.gateway.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS Gateway", "URL "+stringType)

	ConfigSubscriptionMax               = ffc("config.subscription.max", "The maximum number of pre-defined subscriptions that can exist (note for high fan-out consider connecting a dedicated pub/sub broker to the dispatcher)", intType)
	ConfigSubscriptionDefaultsBatchSize = ffc("config.subscription.defaults.batchSize", "Default read ahead to enable for subscriptions that do not explicitly configure readahead", intType)

	ConfigTokensConnector = ffc("config.tokens[].connector", "The name of the Tokens Connector. This will be used in the FireFly API path to refer to this specific Token Connector", stringType)
	ConfigTokensName      = ffc("config.tokens[].name", "The name of the Tokens Connector. This will be used in the FireFly API path to refer to this specific Token Connector", stringType)
	ConfigTokensPlugin    = ffc("config.tokens[].plugin", "The name of the Tokens Connector plugin to use", stringType)
	ConfigTokensURL       = ffc("config.tokens[].url", "The URL of the Token Connector", "URL "+stringType)

	ConfigTokensProxyURL = ffc("config.tokens[].proxy.url", "Optional HTTP proxy server to use when connecting to the Token Connector", "URL "+stringType)

	ConfigUIEnabled = ffc("config.ui.enabled", "Enables the web user interface", booleanType)
	ConfigUIPath    = ffc("config.ui.path", "The file system path which contains the static HTML, CSS, and JavaScript files for the user interface", stringType)

	ConfigAPIOASPanicOnMissingDescription = ffc("config.api.oas.panicOnMissingDescription", "Used for testing purposes only", IgnoredType)

	ConfigAdminWebSocketBlockedWarnInternal = ffc("config.admin.ws.blockedWarnInterval", "How often to log warnings in core, when an admin change event listener falls behind the stream they requested and misses events", timeDurationType)
	ConfigAdminWebSocketEventQueueLength    = ffc("config.admin.ws.eventQueueLength", "Server-side queue length for events waiting for delivery over an admin change event listener websocket", intType)
)
