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

var ffm = i18n.FFM

//revive:disable
var (
	ConfigGlobalConnectionTimeout = ffm("config.global.connectionTimeout", "The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted")
	ConfigGlobalRequestTimeout    = ffm("config.global.requestTimeout", "The maximum amount of time, in milliseconds that a request is allowed to remain open")

	ConfigGlobalRetryEnabled      = ffm("config.global.retry.enabled", "Enables retries")
	ConfigGlobalRetryFactor       = ffm("config.global.retry.factor", "The retry backoff factor")
	ConfigGlobalRetryInitDelay    = ffm("config.global.retry.initDelay", "The initial retry delay")
	ConfigGlobalRetryInitialDelay = ffm("config.global.retry.initialDelay", "The initial retry delay")
	ConfigGlobalRetryMaxDelay     = ffm("config.global.retry.maxDelay", "The maximum retry delay")
	ConfigGlobalRetryMaxAttempts  = ffm("config.global.retry.maxAttempts", "The maximum number of times to retry")
	ConfigGlobalRetryCount        = ffm("config.global.retry.count", "The maximum number of times to retry")
	ConfigGlobalInitWaitTime      = ffm("config.global.retry.initWaitTime", "The initial retry delay")
	ConfigGlobalMaxWaitTime       = ffm("config.global.retry.maxWaitTime", "The maximum retry delay")

	ConfigGlobalUsername = ffm("config.global.auth.username", "Username")
	ConfigGlobalPassword = ffm("config.global.auth.password", "Password")

	ConfigGlobalSize = ffm("config.global.cache.size", "The size of the cache")
	ConfigGlobalTTL  = ffm("config.global.cache.ttl", "The time to live (TTL) for the cache")

	ConfigGlobaltWsHeartbeatInterval     = ffm("config.global.ws.heartbeatInterval", "The number of milliseconds to wait between heartbeat signals on the WebSocket connection")
	ConfigGlobalWsInitialConnectAttempts = ffm("config.global.ws.initialConnectAttempts", "The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing")
	ConfigGlobalWsPath                   = ffm("config.global.ws.path", "The WebSocket sever URL to which FireFly should connect")
	ConfigGlobalWsReadBufferSize         = ffm("config.global.ws.readBufferSize", "The size in bytes of the read buffer for the WebSocket connection")
	ConfigGlobalWsWriteBufferSize        = ffm("config.global.ws.writeBufferSize", "The size in bytes of the write buffer for the WebSocket connection")

	ConfigGlobalTLSCaFile           = ffm("config.global.tls.caFile", "The path to the CA file for the admin API")
	ConfigGlobalTLSCertFile         = ffm("config.global.tls.certFile", "The path to the certificate file for the admin API")
	ConfigGlobalTLSClientAuth       = ffm("config.global.tls.clientAuth", "Enables or disables client auth for the admin API")
	ConfigGlobalTLSEnabled          = ffm("config.global.tls.enabled", "Enables or disables TLS on the admin API")
	ConfigGlobalTLSKeyFile          = ffm("config.global.tls.keyFile", "The path to the private key file for the admin API")
	ConfigGlobalTLSHandshakeTimeout = ffm("config.global.tlsHandshakeTimeout", "The maximum amount of time, in milliseconds to wait for a successful TLS handshake")

	ConfigAdminAddress      = ffm("config.admin.address", "The IP address on which the admin HTTP API should listen")
	ConfigAdminEnabled      = ffm("config.admin.enabled", "Enables the admin HTTP API")
	ConfigAdminPort         = ffm("config.admin.port", "The port on which the admin HTTP API should listen")
	ConfigAdminPreInit      = ffm("config.admin.preinit", "Enables the pre-init mode. This mode will let the FireFly Core process start, but not initialize any plugins, besides the database to read any configuration overrides. This allows the admin HTTP API to be used to define custom configuration before starting the rest of FireFly Core.")
	ConfigAdminPublicURL    = ffm("config.admin.publicURL", "The fully qualified public URL for the admin API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.")
	ConfigAdminReadTimeout  = ffm("config.admin.readTimeout", "The maximum time to wait in seconds when reading from an HTTP connection")
	ConfigAdminWriteTimeout = ffm("config.admin.writeTimeout", "The maximum time to wait in seconds when writing to an HTTP connection")

	ConfigAPIDefaultFilterLimit = ffm("config.api.defaultFilterLimit", "The maximum number of rows to return if no limit is specified on an API request")
	ConfigAPIMaxFilterLimit     = ffm("config.api.maxFilterLimit", "The maximum number of rows to return if no limit is specified on an API request")
	ConfigAPIRequestMaxTimeout  = ffm("config.api.requestMaxTimeout", "The maximum amount of time, in milliseconds that an HTTP client can specify in a `Request-Timeout` header to keep a specific request open")
	// ConfigAPIRequestTimeout     = ffm("config.api.requestTimeout", "TBD")
	ConfigAPIShutdownTimeout = ffm("config.api.shutdownTimeout", "The maximum amount of time, in milliseconds to wait for any open HTTP requests to finish before shutting down the HTTP server")

	ConfigAssetManagerKeyNormalization = ffm("config.asset.manager.keyNormalization", "Mechanism to normalize keys before using them. Valid options are `blockchain_plugin` - use blockchain plugin (default) or `none` - do not attempt normalization")

	ConfigBatchManagerMinimumPollDelay = ffm("config.batch.manager.minimumPollDelay", "The minimum time the batch manager waits between polls on the DB - to prevent thrashing")
	ConfigBatchManagerPollTimeout      = ffm("config.batch.manager.pollTimeout", "How long to wait without any notifications of new messages before doing a page query")
	ConfigBatchManagerReadPageSize     = ffm("config.batch.manager.readPageSize", "The size of each page of messages read from the database into memory when assembling batches")

	// ConfigBatchRetryFactor    = ffm("config.batch.retry.factor", "TBD")
	// ConfigBatchRetryInitDelay = ffm("config.batch.retry.initDelay", "TBD")
	// ConfigBatchRetryMaxDelay  = ffm("config.batch.retry.maxDelay", "TBD")

	// ConfigBlobreceiverRetryFactor       = ffm("config.blobreceiver.retry.factor", "TBD")
	// ConfigBlobreceiverRetryInitialDelay = ffm("config.blobreceiver.retry.initialDelay", "TBD")
	// ConfigBlobreceiverRetryMaxDelay     = ffm("config.blobreceiver.retry.maxDelay", "TBD")

	ConfigBlobreceiverWorkerBatchMaxInserts = ffm("config.blobreceiver.worker.batchMaxInserts", "The maximum number of items the blob receiver worker will insert in a batch")
	ConfigBlobreceiverWorkerBatchTimeout    = ffm("config.blobreceiver.worker.batchTimeout", "The maximum amount of the the blob receiver worker will wait")
	ConfigBlobreceiverWorkerCount           = ffm("config.blobreceiver.worker.count", "The number of blob receiver worker")

	ConfigBlockchainType = ffm("config.blockchain.type", "A string defining which type of blockchain plugin to use. This tells FireFly which type of configuration to load for the rest of the `blockchain` section.")

	ConfigBlockchainEthereumAddressResolverBodyTemplate = ffm("config.blockchain.ethereum.addressResolver.bodyTemplate", "TBD")
	// ConfigBlockchainEthereumAddressResolverConnectionTimeout     = ffm("config.blockchain.ethereum.addressResolver.connectionTimeout", "TBD")
	ConfigBlockchainEthereumAddressResolverCustomClient          = ffm("config.blockchain.ethereum.addressResolver.customClient", "TBD")
	ConfigBlockchainEthereumAddressResolverExpectContinueTimeout = ffm("config.blockchain.ethereum.addressResolver.expectContinueTimeout", "TBD")
	ConfigBlockchainEthereumAddressResolverHeaders               = ffm("config.blockchain.ethereum.addressResolver.headers", "TBD")
	ConfigBlockchainEthereumAddressResolverIdleTimeout           = ffm("config.blockchain.ethereum.addressResolver.idleTimeout", "TBD")
	ConfigBlockchainEthereumAddressResolverMaxIdleConns          = ffm("config.blockchain.ethereum.addressResolver.maxIdleConns", "TBD")
	ConfigBlockchainEthereumAddressResolverMethod                = ffm("config.blockchain.ethereum.addressResolver.method", "The HTTP method to use when making requests to the address resolver")
	// ConfigBlockchainEthereumAddressResolverRequestTimeout        = ffm("config.blockchain.ethereum.addressResolver.requestTimeout", "TBD")
	ConfigBlockchainEthereumAddressResolverResponseField  = ffm("config.blockchain.ethereum.addressResolver.responseField", "TBD")
	ConfigBlockchainEthereumAddressResolverRetainOriginal = ffm("config.blockchain.ethereum.addressResolver.retainOriginal", "TBD")
	// ConfigBlockchainEthereumAddressResolverTLSHandshakeTimeout = ffm("config.blockchain.ethereum.addressResolver.tlsHandshakeTimeout", "TBD")
	ConfigBlockchainEthereumAddressResolverURL         = ffm("config.blockchain.ethereum.addressResolver.url", "The URL of the address resolver")
	ConfigBlockchainEthereumAddressResolverURLTemplate = ffm("config.blockchain.ethereum.addressResolver.urlTemplate", "TBD")

	// ConfigBlockchainEthereumAddressResolverAuthPassword = ffm("config.blockchain.ethereum.addressResolver.auth.password", "TBD")
	// ConfigBlockchainEthereumAddressResolverAuthUsername = ffm("config.blockchain.ethereum.addressResolver.auth.username", "TBD")

	// ConfigBlockchainEthereumAddressResolverCacheSize = ffm("config.blockchain.ethereum.addressResolver.cache.size", "TBD")
	// ConfigBlockchainEthereumAddressResolverCacheTTL  = ffm("config.blockchain.ethereum.addressResolver.cache.ttl", "TBD")

	ConfigBlockchainEthereumAddressResolverProxyURL = ffm("config.blockchain.ethereum.addressResolver.proxy.url", "The URL of the address resolver proxy")

	// ConfigBlockchainEthereumAddressResolverRetryCount        = ffm("config.blockchain.ethereum.addressResolver.retry.count", "The number of times to retry requests to the address resolver")
	// ConfigBlockchainEthereumAddressResolverRetryEnabled      = ffm("config.blockchain.ethereum.addressResolver.retry.enabled", "Enables retrying requests to the address resolver")
	// ConfigBlockchainEthereumAddressResolverRetryInitWaitTime = ffm("config.blockchain.ethereum.addressResolver.retry.initWaitTime", "Initial retry delay for requests to the address resolver")
	// ConfigBlockchainEthereumAddressResolverRetryMaxWaitTime  = ffm("config.blockchain.ethereum.addressResolver.retry.maxWaitTime", "Maximum retry delay for requests to the address resolver")

	ConfigBlockchainEthereumEthconnectBatchSize    = ffm("config.blockchain.ethereum.ethconnect.batchSize", "The maximum number of transactions to send in a single request to Ethconnect")
	ConfigBlockchainEthereumEthconnectBatchTimeout = ffm("config.blockchain.ethereum.ethconnect.batchTimeout", "The maximum amount of time in milliseconds to wait for a batch to complete")
	// ConfigBlockchainEthereumEthconnectConnectionTimeout     = ffm("config.blockchain.ethereum.ethconnect.connectionTimeout", "TBD")
	ConfigBlockchainEthereumEthconnectCustomClient          = ffm("config.blockchain.ethereum.ethconnect.customClient", "TBD")
	ConfigBlockchainEthereumEthconnectExpectContinueTimeout = ffm("config.blockchain.ethereum.ethconnect.expectContinueTimeout", "TBD")
	ConfigBlockchainEthereumEthconnectHeaders               = ffm("config.blockchain.ethereum.ethconnect.headers", "TBD")
	ConfigBlockchainEthereumEthconnectIdleTimeout           = ffm("config.blockchain.ethereum.ethconnect.idleTimeout", "TBD")
	ConfigBlockchainEthereumEthconnectInstance              = ffm("config.blockchain.ethereum.ethconnect.instance", "The Ethereum address of the FireFly BatchPin smart contract that has been deployed to the blockchain")
	ConfigBlockchainEthereumEthconnectMaxIdleConns          = ffm("config.blockchain.ethereum.ethconnect.maxIdleConns", "TBD")
	ConfigBlockchainEthereumEthconnectPrefixLong            = ffm("config.blockchain.ethereum.ethconnect.prefixLong", "The prefix that will be used for Ethconnect specific HTTP headers when FireFly makes requests to Ethconnect")
	ConfigBlockchainEthereumEthconnectPrefixShort           = ffm("config.blockchain.ethereum.ethconnect.prefixShort", "The prefix that will be used for Ethconnect specific query parameters when FireFly makes requests to Ethconnect")
	// ConfigBlockchainEthereumEthconnectRequestTimeout        = ffm("config.blockchain.ethereum.ethconnect.requestTimeout", "TBD")
	// ConfigBlockchainEthereumEthconnectTLSHandshakeTimeout = ffm("config.blockchain.ethereum.ethconnect.tlsHandshakeTimeout", "TBD")
	ConfigBlockchainEthereumEthconnectTopic = ffm("config.blockchain.ethereum.ethconnect.topic", "TBD")
	ConfigBlockchainEthereumEthconnectURL   = ffm("config.blockchain.ethereum.ethconnect.url", "The URL of the Ethconnect instance")

	// ConfigBlockchainEthereumEthconnectAuthPassword = ffm("config.blockchain.ethereum.ethconnect.auth.password", "TBD")
	// ConfigBlockchainEthereumEthconnectAuthUsername = ffm("config.blockchain.ethereum.ethconnect.auth.username", "TBD")

	ConfigBlockchainEthereumEthconnectProxyURL = ffm("config.blockchain.ethereum.ethconnect.proxy.url", "The URL of the Ethconnect proxy")

	// ConfigBlockchainEthereumEthconnectRetryCount        = ffm("config.blockchain.ethereum.ethconnect.retry.count", "The number of times to retry requests to Ethconnect")
	// ConfigBlockchainEthereumEthconnectRetryEnabled      = ffm("config.blockchain.ethereum.ethconnect.retry.enabled", "Enables retrying requests to Ethconnect")
	// ConfigBlockchainEthereumEthconnectRetryInitWaitTime = ffm("config.blockchain.ethereum.ethconnect.retry.initWaitTime", "Initial retry delay for requests to Ethconnect")
	// ConfigBlockchainEthereumEthconnectRetryMaxWaitTime  = ffm("config.blockchain.ethereum.ethconnect.retry.maxWaitTime", "Maximum retry delay for requests to Ethconnect")

	// ConfigBlockchainEthereumEthconnectWsHeartbeatInterval      = ffm("config.blockchain.ethereum.ethconnect.ws.heartbeatInterval", "The number of milliseconds to wait between heartbeat signals on the WebSocket connection")
	// ConfigBlockchainEthereumEthconnectWsInitialConnectAttempts = ffm("config.blockchain.ethereum.ethconnect.ws.initialConnectAttempts", "The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing")
	// ConfigBlockchainEthereumEthconnectWsPath                   = ffm("config.blockchain.ethereum.ethconnect.ws.path", "The WebSocket sever URL to which FireFly should connect")
	// ConfigBlockchainEthereumEthconnectWsReadBufferSize         = ffm("config.blockchain.ethereum.ethconnect.ws.readBufferSize", "The size in bytes of the read buffer for the WebSocket connection")
	// ConfigBlockchainEthereumEthconnectWsWriteBufferSize        = ffm("config.blockchain.ethereum.ethconnect.ws.writeBufferSize", "The size in bytes of the write buffer for the WebSocket connection")

	ConfigBlockchainFabricFabconnectBatchSize    = ffm("config.blockchain.fabric.fabconnect.batchSize", "The maximum number of transactions to send in a single request to Fabconnect")
	ConfigBlockchainFabricFabconnectBatchTimeout = ffm("config.blockchain.fabric.fabconnect.batchTimeout", "The maximum amount of time in milliseconds to wait for a batch to complete")
	ConfigBlockchainFabricFabconnectChaincode    = ffm("config.blockchain.fabric.fabconnect.chaincode", "The name of the Fabric chaincode that FireFly will use for BatchPin transactions")
	ConfigBlockchainFabricFabconnectChannel      = ffm("config.blockchain.fabric.fabconnect.channel", "The Fabric channel that FireFly will use for BatchPin transactions")
	// ConfigBlockchainFabricFabconnectConnectionTimeout     = ffm("config.blockchain.fabric.fabconnect.connectionTimeout", "TBD")
	ConfigBlockchainFabricFabconnectCustomClient          = ffm("config.blockchain.fabric.fabconnect.customClient", "TBD")
	ConfigBlockchainFabricFabconnectExpectContinueTimeout = ffm("config.blockchain.fabric.fabconnect.expectContinueTimeout", "TBD")
	ConfigBlockchainFabricFabconnectHeaders               = ffm("config.blockchain.fabric.fabconnect.headers", "TBD")
	ConfigBlockchainFabricFabconnectIdleTimeout           = ffm("config.blockchain.fabric.fabconnect.idleTimeout", "TBD")
	ConfigBlockchainFabricFabconnectMaxIdleConns          = ffm("config.blockchain.fabric.fabconnect.maxIdleConns", "TBD")
	ConfigBlockchainFabricFabconnectPrefixLong            = ffm("config.blockchain.fabric.fabconnect.prefixLong", "The prefix that will be used for Fabconnect specific HTTP headers when FireFly makes requests to Fabconnect")
	ConfigBlockchainFabricFabconnectPrefixShort           = ffm("config.blockchain.fabric.fabconnect.prefixShort", "The prefix that will be used for Fabconnect specific query parameters when FireFly makes requests to Fabconnect")
	// ConfigBlockchainFabricFabconnectRequestTimeout        = ffm("config.blockchain.fabric.fabconnect.requestTimeout", "TBD")
	ConfigBlockchainFabricFabconnectSigner = ffm("config.blockchain.fabric.fabconnect.signer", "The Fabric signing key to use when submitting transactions to Fabconnect")
	// ConfigBlockchainFabricFabconnectTLSHandshakeTimeout = ffm("config.blockchain.fabric.fabconnect.tlsHandshakeTimeout", "TBD")
	ConfigBlockchainFabricFabconnectTopic = ffm("config.blockchain.fabric.fabconnect.topic", "TBD")
	ConfigBlockchainFabricFabconnectURL   = ffm("config.blockchain.fabric.fabconnect.url", "The URL of the Fabconnect instance")

	// ConfigBlockchainFabricFabconnectAuthPassword = ffm("config.blockchain.fabric.fabconnect.auth.password", "TBD")
	// ConfigBlockchainFabricFabconnectAuthUsername = ffm("config.blockchain.fabric.fabconnect.auth.username", "TBD")

	ConfigBlockchainFabricFabconnectProxyURL = ffm("config.blockchain.fabric.fabconnect.proxy.url", "The URL for the Fabconnect proxy")

	// ConfigBlockchainFabricFabconnectRetryCount        = ffm("config.blockchain.fabric.fabconnect.retry.count", "TBD")
	// ConfigBlockchainFabricFabconnectRetryEnabled      = ffm("config.blockchain.fabric.fabconnect.retry.enabled", "TBD")
	// ConfigBlockchainFabricFabconnectRetryInitWaitTime = ffm("config.blockchain.fabric.fabconnect.retry.initWaitTime", "TBD")
	// ConfigBlockchainFabricFabconnectRetryMaxWaitTime  = ffm("config.blockchain.fabric.fabconnect.retry.maxWaitTime", "TBD")

	// ConfigBlockchainFabricFabconnectWsHeartbeatInterval      = ffm("config.blockchain.fabric.fabconnect.ws.heartbeatInterval", "The number of milliseconds to wait between heartbeat signals on the WebSocket connection")
	// ConfigBlockchainFabricFabconnectWsInitialConnectAttempts = ffm("config.blockchain.fabric.fabconnect.ws.initialConnectAttempts", "The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing")
	// ConfigBlockchainFabricFabconnectWsPath                   = ffm("config.blockchain.fabric.fabconnect.ws.path", "The WebSocket sever URL to which FireFly should connect")
	// ConfigBlockchainFabricFabconnectWsReadBufferSize         = ffm("config.blockchain.fabric.fabconnect.ws.readBufferSize", "The size in bytes of the read buffer for the WebSocket connection")
	// ConfigBlockchainFabricFabconnectWsWriteBufferSize        = ffm("config.blockchain.fabric.fabconnect.ws.writeBufferSize", "The size in bytes of the write buffer for the WebSocket connection")

	// ConfigBlockchaineventCacheSize = ffm("config.blockchainevent.cache.size", "TBD")
	// ConfigBlockchaineventCacheTTL  = ffm("config.blockchainevent.cache.ttl", "TBD")

	ConfigBroadcastBatchAgentTimeout = ffm("config.broadcast.batch.agentTimeout", "TBD")
	ConfigBroadcastBatchPayloadLimit = ffm("config.broadcast.batch.payloadLimit", "TBD")
	ConfigBroadcastBatchSize         = ffm("config.broadcast.batch.size", "TBD")
	ConfigBroadcastBatchTimeout      = ffm("config.broadcast.batch.timeout", "TBD")

	ConfigCorsCredentials = ffm("config.cors.credentials", "TBD")
	ConfigCorsDebug       = ffm("config.cors.debug", "TBD")
	ConfigCorsEnabled     = ffm("config.cors.enabled", "TBD")
	ConfigCorsHeaders     = ffm("config.cors.headers", "TBD")
	ConfigCorsMaxAge      = ffm("config.cors.maxAge", "TBD")
	ConfigCorsMethods     = ffm("config.cors.methods", "TBD")
	ConfigCorsOrigins     = ffm("config.cors.origins", "TBD")

	ConfigDatabaseMaxChartRows = ffm("config.database.maxChartRows", "TBD")
	ConfigDatabaseType         = ffm("config.database.type", "TBD")

	ConfigDatabasePostgresMaxConnIdleTime = ffm("config.database.postgres.maxConnIdleTime", "TBD")
	ConfigDatabasePostgresMaxConnLifetime = ffm("config.database.postgres.maxConnLifetime", "TBD")
	ConfigDatabasePostgresMaxConns        = ffm("config.database.postgres.maxConns", "TBD")
	ConfigDatabasePostgresMaxIdleConns    = ffm("config.database.postgres.maxIdleConns", "TBD")
	ConfigDatabasePostgresURL             = ffm("config.database.postgres.url", "TBD")

	ConfigDatabasePostgresMigrationsAuto      = ffm("config.database.postgres.migrations.auto", "TBD")
	ConfigDatabasePostgresMigrationsDirectory = ffm("config.database.postgres.migrations.directory", "TBD")

	ConfigDatabaseSqlite3MaxConnIdleTime = ffm("config.database.sqlite3.maxConnIdleTime", "TBD")
	ConfigDatabaseSqlite3MaxConnLifetime = ffm("config.database.sqlite3.maxConnLifetime", "TBD")
	ConfigDatabaseSqlite3MaxConns        = ffm("config.database.sqlite3.maxConns", "TBD")
	ConfigDatabaseSqlite3MaxIdleConns    = ffm("config.database.sqlite3.maxIdleConns", "TBD")
	ConfigDatabaseSqlite3URL             = ffm("config.database.sqlite3.url", "TBD")

	ConfigDatabaseSqlite3MigrationsAuto      = ffm("config.database.sqlite3.migrations.auto", "TBD")
	ConfigDatabaseSqlite3MigrationsDirectory = ffm("config.database.sqlite3.migrations.directory", "TBD")

	ConfigDataexchangeType = ffm("config.dataexchange.type", "TBD")

	// ConfigDataexchangeFfdxConnectionTimeout     = ffm("config.dataexchange.ffdx.connectionTimeout", "TBD")
	ConfigDataexchangeFfdxCustomClient          = ffm("config.dataexchange.ffdx.customClient", "TBD")
	ConfigDataexchangeFfdxExpectContinueTimeout = ffm("config.dataexchange.ffdx.expectContinueTimeout", "TBD")
	ConfigDataexchangeFfdxHeaders               = ffm("config.dataexchange.ffdx.headers", "TBD")
	ConfigDataexchangeFfdxIdleTimeout           = ffm("config.dataexchange.ffdx.idleTimeout", "TBD")
	ConfigDataexchangeFfdxInitEnabled           = ffm("config.dataexchange.ffdx.initEnabled", "TBD")
	ConfigDataexchangeFfdxManifestEnabled       = ffm("config.dataexchange.ffdx.manifestEnabled", "TBD")
	ConfigDataexchangeFfdxMaxIdleConns          = ffm("config.dataexchange.ffdx.maxIdleConns", "TBD")
	// ConfigDataexchangeFfdxRequestTimeout        = ffm("config.dataexchange.ffdx.requestTimeout", "TBD")
	// ConfigDataexchangeFfdxTLSHandshakeTimeout = ffm("config.dataexchange.ffdx.tlsHandshakeTimeout", "TBD")
	ConfigDataexchangeFfdxURL = ffm("config.dataexchange.ffdx.url", "TBD")

	// ConfigDataexchangeFfdxAuthPassword = ffm("config.dataexchange.ffdx.auth.password", "TBD")
	// ConfigDataexchangeFfdxAuthUsername = ffm("config.dataexchange.ffdx.auth.username", "TBD")

	ConfigDataexchangeFfdxProxyURL = ffm("config.dataexchange.ffdx.proxy.url", "TBD")

	// ConfigDataexchangeFfdxRetryCount        = ffm("config.dataexchange.ffdx.retry.count", "TBD")
	// ConfigDataexchangeFfdxRetryEnabled      = ffm("config.dataexchange.ffdx.retry.enabled", "TBD")
	// ConfigDataexchangeFfdxRetryInitWaitTime = ffm("config.dataexchange.ffdx.retry.initWaitTime", "TBD")
	// ConfigDataexchangeFfdxRetryMaxWaitTime  = ffm("config.dataexchange.ffdx.retry.maxWaitTime", "TBD")

	// ConfigDataexchangeFfdxWsHeartbeatInterval      = ffm("config.dataexchange.ffdx.ws.heartbeatInterval", "The number of milliseconds to wait between heartbeat signals on the WebSocket connection")
	// ConfigDataexchangeFfdxWsInitialConnectAttempts = ffm("config.dataexchange.ffdx.ws.initialConnectAttempts", "The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing")
	// ConfigDataexchangeFfdxWsPath                   = ffm("config.dataexchange.ffdx.ws.path", "The WebSocket sever URL to which FireFly should connect")
	// ConfigDataexchangeFfdxWsReadBufferSize         = ffm("config.dataexchange.ffdx.ws.readBufferSize", "The size in bytes of the read buffer for the WebSocket connection")
	// ConfigDataexchangeFfdxWsWriteBufferSize        = ffm("config.dataexchange.ffdx.ws.writeBufferSize", "The size in bytes of the write buffer for the WebSocket connection")

	ConfigDebugPort = ffm("config.debug.port", "TBD")

	// ConfigDownloadRetryFactor       = ffm("config.download.retry.factor", "TBD")
	// ConfigDownloadRetryInitialDelay = ffm("config.download.retry.initialDelay", "TBD")
	// ConfigDownloadRetryMaxAttempts  = ffm("config.download.retry.maxAttempts", "TBD")
	// ConfigDownloadRetryMaxDelay     = ffm("config.download.retry.maxDelay", "TBD")

	ConfigDownloadWorkerCount       = ffm("config.download.worker.count", "TBD")
	ConfigDownloadWorkerQueueLength = ffm("config.download.worker.queueLength", "TBD")

	ConfigEventAggregatorBatchSize            = ffm("config.event.aggregator.batchSize", "TBD")
	ConfigEventAggregatorBatchTimeout         = ffm("config.event.aggregator.batchTimeout", "TBD")
	ConfigEventAggregatorFirstEvent           = ffm("config.event.aggregator.firstEvent", "TBD")
	ConfigEventAggregatorOpCorrelationRetries = ffm("config.event.aggregator.opCorrelationRetries", "TBD")
	ConfigEventAggregatorPollTimeout          = ffm("config.event.aggregator.pollTimeout", "TBD")

	// ConfigEventAggregatorRetryFactor    = ffm("config.event.aggregator.retry.factor", "TBD")
	// ConfigEventAggregatorRetryInitDelay = ffm("config.event.aggregator.retry.initDelay", "TBD")
	// ConfigEventAggregatorRetryMaxDelay  = ffm("config.event.aggregator.retry.maxDelay", "TBD")

	ConfigEventDbeventsBufferSize = ffm("config.event.dbevents.bufferSize", "TBD")

	ConfigEventDispatcherBatchTimeout = ffm("config.event.dispatcher.batchTimeout", "TBD")
	ConfigEventDispatcherBufferLength = ffm("config.event.dispatcher.bufferLength", "TBD")
	ConfigEventDispatcherPollTimeout  = ffm("config.event.dispatcher.pollTimeout", "TBD")

	// ConfigEventDispatcherRetryFactor    = ffm("config.event.dispatcher.retry.factor", "TBD")
	// ConfigEventDispatcherRetryInitDelay = ffm("config.event.dispatcher.retry.initDelay", "TBD")
	// ConfigEventDispatcherRetryMaxDelay  = ffm("config.event.dispatcher.retry.maxDelay", "TBD")

	// ConfigEventListenerTopicCacheSize = ffm("config.event.listenerTopic.cache.size", "TBD")
	// ConfigEventListenerTopicCacheTTL  = ffm("config.event.listenerTopic.cache.ttl", "TBD")

	ConfigEventTransportsDefault = ffm("config.event.transports.default", "TBD")
	ConfigEventTransportsEnabled = ffm("config.event.transports.enabled", "TBD")

	// ConfigGroupCacheSize = ffm("config.group.cache.size", "TBD")
	// ConfigGroupCacheTTL  = ffm("config.group.cache.ttl", "TBD")

	ConfigHTTPAddress      = ffm("config.http.address", "The IP address on which the HTTP API should listen")
	ConfigHTTPPort         = ffm("config.http.port", "The port on which the HTTP API should listen")
	ConfigHTTPPublicURL    = ffm("config.http.publicURL", "The fully qualified public URL for the API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.")
	ConfigHTTPReadTimeout  = ffm("config.http.readTimeout", "The maximum time to wait in seconds when reading from an HTTP connection")
	ConfigHTTPWriteTimeout = ffm("config.http.writeTimeout", "The maximum time to wait in seconds when writing to an HTTP connection")

	// ConfigHTTPTLSCaFile     = ffm("config.http.tls.caFile", "TBD")
	// ConfigHTTPTLSCertFile   = ffm("config.http.tls.certFile", "TBD")
	// ConfigHTTPTLSClientAuth = ffm("config.http.tls.clientAuth", "TBD")
	// ConfigHTTPTLSEnabled    = ffm("config.http.tls.enabled", "TBD")
	// ConfigHTTPTLSKeyFile    = ffm("config.http.tls.keyFile", "TBD")

	ConfigIdentityType = ffm("config.identity.type", "TBD")

	ConfigIdentityManagerCacheLimit = ffm("config.identity.manager.cache.limit", "TBD")
	// ConfigIdentityManagerCacheTTL   = ffm("config.identity.manager.cache.ttl", "TBD")

	ConfigLogCompress   = ffm("config.log.compress", "TBD")
	ConfigLogFilename   = ffm("config.log.filename", "TBD")
	ConfigLogFilesize   = ffm("config.log.filesize", "TBD")
	ConfigLogForceColor = ffm("config.log.forceColor", "TBD")
	ConfigLogLevel      = ffm("config.log.level", "TBD")
	ConfigLogMaxAge     = ffm("config.log.maxAge", "TBD")
	ConfigLogMaxBackups = ffm("config.log.maxBackups", "TBD")
	ConfigLogNoColor    = ffm("config.log.noColor", "TBD")
	ConfigLogTimeFormat = ffm("config.log.timeFormat", "TBD")
	ConfigLogUtc        = ffm("config.log.utc", "TBD")

	// ConfigMessageCacheSize = ffm("config.message.cache.size", "TBD")
	// ConfigMessageCacheTTL  = ffm("config.message.cache.ttl", "TBD")

	ConfigMessageWriterBatchMaxInserts = ffm("config.message.writer.batchMaxInserts", "TBD")
	ConfigMessageWriterBatchTimeout    = ffm("config.message.writer.batchTimeout", "TBD")
	ConfigMessageWriterCount           = ffm("config.message.writer.count", "TBD")

	ConfigMetricsAddress      = ffm("config.metrics.address", "The IP address on which the metrics HTTP API should listen")
	ConfigMetricsEnabled      = ffm("config.metrics.enabled", "Enables the metrics API")
	ConfigMetricsPath         = ffm("config.metrics.path", "TBD")
	ConfigMetricsPort         = ffm("config.metrics.port", "The port on which the metrics HTTP API should listen")
	ConfigMetricsPublicURL    = ffm("config.metrics.publicURL", "The fully qualified public URL for the metrics API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.")
	ConfigMetricsReadTimeout  = ffm("config.metrics.readTimeout", "The maximum time to wait in seconds when reading from an HTTP connection")
	ConfigMetricsWriteTimeout = ffm("config.metrics.writeTimeout", "The maximum time to wait in seconds when writing to an HTTP connection")

	// ConfigMetricsTLSCaFile     = ffm("config.metrics.tls.caFile", "TBD")
	// ConfigMetricsTLSCertFile   = ffm("config.metrics.tls.certFile", "TBD")
	// ConfigMetricsTLSClientAuth = ffm("config.metrics.tls.clientAuth", "TBD")
	// ConfigMetricsTLSEnabled    = ffm("config.metrics.tls.enabled", "TBD")
	// ConfigMetricsTLSKeyFile    = ffm("config.metrics.tls.keyFile", "TBD")

	ConfigNamespacesDefault    = ffm("config.namespaces.default", "TBD")
	ConfigNamespacesPredefined = ffm("config.namespaces.predefined", "TBD")

	ConfigNodeDescription = ffm("config.node.description", "TBD")
	ConfigNodeName        = ffm("config.node.name", "TBD")

	// ConfigOpupdateRetryFactor       = ffm("config.opupdate.retry.factor", "TBD")
	// ConfigOpupdateRetryInitialDelay = ffm("config.opupdate.retry.initialDelay", "TBD")
	// ConfigOpupdateRetryMaxDelay     = ffm("config.opupdate.retry.maxDelay", "TBD")

	ConfigOpupdateWorkerBatchMaxInserts = ffm("config.opupdate.worker.batchMaxInserts", "TBD")
	ConfigOpupdateWorkerBatchTimeout    = ffm("config.opupdate.worker.batchTimeout", "TBD")
	ConfigOpupdateWorkerCount           = ffm("config.opupdate.worker.count", "TBD")
	ConfigOpupdateWorkerQueueLength     = ffm("config.opupdate.worker.queueLength", "TBD")

	ConfigOrchestratorStartupAttempts = ffm("config.orchestrator.startupAttempts", "TBD")

	ConfigOrgDescription = ffm("config.org.description", "TBD")
	ConfigOrgIdentity    = ffm("config.org.identity", "TBD")
	ConfigOrgKey         = ffm("config.org.key", "TBD")
	ConfigOrgName        = ffm("config.org.name", "TBD")

	ConfigPrivatemessagingOpCorrelationRetries = ffm("config.privatemessaging.opCorrelationRetries", "TBD")

	ConfigPrivatemessagingBatchAgentTimeout = ffm("config.privatemessaging.batch.agentTimeout", "TBD")
	ConfigPrivatemessagingBatchPayloadLimit = ffm("config.privatemessaging.batch.payloadLimit", "TBD")
	ConfigPrivatemessagingBatchSize         = ffm("config.privatemessaging.batch.size", "TBD")
	ConfigPrivatemessagingBatchTimeout      = ffm("config.privatemessaging.batch.timeout", "TBD")

	// ConfigPrivatemessagingRetryFactor    = ffm("config.privatemessaging.retry.factor", "TBD")
	// ConfigPrivatemessagingRetryInitDelay = ffm("config.privatemessaging.retry.initDelay", "TBD")
	// ConfigPrivatemessagingRetryMaxDelay  = ffm("config.privatemessaging.retry.maxDelay", "TBD")

	ConfigPublicstorageType = ffm("config.publicstorage.type", "TBD")

	// ConfigPublicstorageIpfsAPIConnectionTimeout     = ffm("config.publicstorage.ipfs.api.connectionTimeout", "TBD")
	ConfigPublicstorageIpfsAPICustomClient          = ffm("config.publicstorage.ipfs.api.customClient", "TBD")
	ConfigPublicstorageIpfsAPIExpectContinueTimeout = ffm("config.publicstorage.ipfs.api.expectContinueTimeout", "TBD")
	ConfigPublicstorageIpfsAPIHeaders               = ffm("config.publicstorage.ipfs.api.headers", "TBD")
	ConfigPublicstorageIpfsAPIIdleTimeout           = ffm("config.publicstorage.ipfs.api.idleTimeout", "TBD")
	ConfigPublicstorageIpfsAPIMaxIdleConns          = ffm("config.publicstorage.ipfs.api.maxIdleConns", "TBD")
	// ConfigPublicstorageIpfsAPIRequestTimeout        = ffm("config.publicstorage.ipfs.api.requestTimeout", "TBD")
	// ConfigPublicstorageIpfsAPITLSHandshakeTimeout = ffm("config.publicstorage.ipfs.api.tlsHandshakeTimeout", "TBD")
	ConfigPublicstorageIpfsAPIURL = ffm("config.publicstorage.ipfs.api.url", "TBD")

	// ConfigPublicstorageIpfsAPIAuthPassword = ffm("config.publicstorage.ipfs.api.auth.password", "TBD")
	// ConfigPublicstorageIpfsAPIAuthUsername = ffm("config.publicstorage.ipfs.api.auth.username", "TBD")

	ConfigPublicstorageIpfsAPIProxyURL = ffm("config.publicstorage.ipfs.api.proxy.url", "TBD")

	// ConfigPublicstorageIpfsAPIRetryCount        = ffm("config.publicstorage.ipfs.api.retry.count", "TBD")
	// ConfigPublicstorageIpfsAPIRetryEnabled      = ffm("config.publicstorage.ipfs.api.retry.enabled", "TBD")
	// ConfigPublicstorageIpfsAPIRetryInitWaitTime = ffm("config.publicstorage.ipfs.api.retry.initWaitTime", "TBD")
	// ConfigPublicstorageIpfsAPIRetryMaxWaitTime  = ffm("config.publicstorage.ipfs.api.retry.maxWaitTime", "TBD")

	// ConfigPublicstorageIpfsGatewayConnectionTimeout     = ffm("config.publicstorage.ipfs.gateway.connectionTimeout", "TBD")
	ConfigPublicstorageIpfsGatewayCustomClient          = ffm("config.publicstorage.ipfs.gateway.customClient", "TBD")
	ConfigPublicstorageIpfsGatewayExpectContinueTimeout = ffm("config.publicstorage.ipfs.gateway.expectContinueTimeout", "TBD")
	ConfigPublicstorageIpfsGatewayHeaders               = ffm("config.publicstorage.ipfs.gateway.headers", "TBD")
	ConfigPublicstorageIpfsGatewayIdleTimeout           = ffm("config.publicstorage.ipfs.gateway.idleTimeout", "TBD")
	ConfigPublicstorageIpfsGatewayMaxIdleConns          = ffm("config.publicstorage.ipfs.gateway.maxIdleConns", "TBD")
	// ConfigPublicstorageIpfsGatewayRequestTimeout        = ffm("config.publicstorage.ipfs.gateway.requestTimeout", "TBD")
	// ConfigPublicstorageIpfsGatewayTLSHandshakeTimeout = ffm("config.publicstorage.ipfs.gateway.tlsHandshakeTimeout", "TBD")
	ConfigPublicstorageIpfsGatewayURL = ffm("config.publicstorage.ipfs.gateway.url", "TBD")

	// ConfigPublicstorageIpfsGatewayAuthPassword = ffm("config.publicstorage.ipfs.gateway.auth.password", "TBD")
	// ConfigPublicstorageIpfsGatewayAuthUsername = ffm("config.publicstorage.ipfs.gateway.auth.username", "TBD")

	ConfigPublicstorageIpfsGatewayProxyURL = ffm("config.publicstorage.ipfs.gateway.proxy.url", "TBD")

	// ConfigPublicstorageIpfsGatewayRetryCount        = ffm("config.publicstorage.ipfs.gateway.retry.count", "TBD")
	// ConfigPublicstorageIpfsGatewayRetryEnabled      = ffm("config.publicstorage.ipfs.gateway.retry.enabled", "TBD")
	// ConfigPublicstorageIpfsGatewayRetryInitWaitTime = ffm("config.publicstorage.ipfs.gateway.retry.initWaitTime", "TBD")
	// ConfigPublicstorageIpfsGatewayRetryMaxWaitTime  = ffm("config.publicstorage.ipfs.gateway.retry.maxWaitTime", "TBD")

	ConfigSharedstorageType = ffm("config.sharedstorage.type", "TBD")

	// ConfigSharedstorageIpfsAPIConnectionTimeout     = ffm("config.sharedstorage.ipfs.api.connectionTimeout", "TBD")
	ConfigSharedstorageIpfsAPICustomClient          = ffm("config.sharedstorage.ipfs.api.customClient", "TBD")
	ConfigSharedstorageIpfsAPIExpectContinueTimeout = ffm("config.sharedstorage.ipfs.api.expectContinueTimeout", "TBD")
	ConfigSharedstorageIpfsAPIHeaders               = ffm("config.sharedstorage.ipfs.api.headers", "TBD")
	ConfigSharedstorageIpfsAPIIdleTimeout           = ffm("config.sharedstorage.ipfs.api.idleTimeout", "TBD")
	ConfigSharedstorageIpfsAPIMaxIdleConns          = ffm("config.sharedstorage.ipfs.api.maxIdleConns", "TBD")
	// ConfigSharedstorageIpfsAPIRequestTimeout        = ffm("config.sharedstorage.ipfs.api.requestTimeout", "TBD")
	// ConfigSharedstorageIpfsAPITLSHandshakeTimeout = ffm("config.sharedstorage.ipfs.api.tlsHandshakeTimeout", "TBD")
	ConfigSharedstorageIpfsAPIURL = ffm("config.sharedstorage.ipfs.api.url", "TBD")

	// ConfigSharedstorageIpfsAPIAuthPassword = ffm("config.sharedstorage.ipfs.api.auth.password", "TBD")
	// ConfigSharedstorageIpfsAPIAuthUsername = ffm("config.sharedstorage.ipfs.api.auth.username", "TBD")

	ConfigSharedstorageIpfsAPIProxyURL = ffm("config.sharedstorage.ipfs.api.proxy.url", "TBD")

	// ConfigSharedstorageIpfsAPIRetryCount        = ffm("config.sharedstorage.ipfs.api.retry.count", "TBD")
	// ConfigSharedstorageIpfsAPIRetryEnabled      = ffm("config.sharedstorage.ipfs.api.retry.enabled", "TBD")
	// ConfigSharedstorageIpfsAPIRetryInitWaitTime = ffm("config.sharedstorage.ipfs.api.retry.initWaitTime", "TBD")
	// ConfigSharedstorageIpfsAPIRetryMaxWaitTime  = ffm("config.sharedstorage.ipfs.api.retry.maxWaitTime", "TBD")

	// ConfigSharedstorageIpfsGatewayConnectionTimeout     = ffm("config.sharedstorage.ipfs.gateway.connectionTimeout", "TBD")
	ConfigSharedstorageIpfsGatewayCustomClient          = ffm("config.sharedstorage.ipfs.gateway.customClient", "TBD")
	ConfigSharedstorageIpfsGatewayExpectContinueTimeout = ffm("config.sharedstorage.ipfs.gateway.expectContinueTimeout", "TBD")
	ConfigSharedstorageIpfsGatewayHeaders               = ffm("config.sharedstorage.ipfs.gateway.headers", "TBD")
	ConfigSharedstorageIpfsGatewayIdleTimeout           = ffm("config.sharedstorage.ipfs.gateway.idleTimeout", "TBD")
	ConfigSharedstorageIpfsGatewayMaxIdleConns          = ffm("config.sharedstorage.ipfs.gateway.maxIdleConns", "TBD")
	// ConfigSharedstorageIpfsGatewayRequestTimeout        = ffm("config.sharedstorage.ipfs.gateway.requestTimeout", "TBD")
	// ConfigSharedstorageIpfsGatewayTLSHandshakeTimeout = ffm("config.sharedstorage.ipfs.gateway.tlsHandshakeTimeout", "TBD")
	ConfigSharedstorageIpfsGatewayURL = ffm("config.sharedstorage.ipfs.gateway.url", "TBD")

	// ConfigSharedstorageIpfsGatewayAuthPassword = ffm("config.sharedstorage.ipfs.gateway.auth.password", "TBD")
	// ConfigSharedstorageIpfsGatewayAuthUsername = ffm("config.sharedstorage.ipfs.gateway.auth.username", "TBD")

	ConfigSharedstorageIpfsGatewayProxyURL = ffm("config.sharedstorage.ipfs.gateway.proxy.url", "TBD")

	// ConfigSharedstorageIpfsGatewayRetryCount        = ffm("config.sharedstorage.ipfs.gateway.retry.count", "TBD")
	// ConfigSharedstorageIpfsGatewayRetryEnabled      = ffm("config.sharedstorage.ipfs.gateway.retry.enabled", "TBD")
	// ConfigSharedstorageIpfsGatewayRetryInitWaitTime = ffm("config.sharedstorage.ipfs.gateway.retry.initWaitTime", "TBD")
	// ConfigSharedstorageIpfsGatewayRetryMaxWaitTime  = ffm("config.sharedstorage.ipfs.gateway.retry.maxWaitTime", "TBD")

	ConfigSubscriptionMax = ffm("config.subscription.max", "TBD")

	ConfigSubscriptionDefaultsBatchSize = ffm("config.subscription.defaults.batchSize", "TBD")

	// ConfigSubscriptionRetryFactor    = ffm("config.subscription.retry.factor", "TBD")
	// ConfigSubscriptionRetryInitDelay = ffm("config.subscription.retry.initDelay", "TBD")
	// ConfigSubscriptionRetryMaxDelay  = ffm("config.subscription.retry.maxDelay", "TBD")

	// ConfigTokensConnectionTimeout     = ffm("config.tokens[].connectionTimeout", "TBD")
	ConfigTokensConnector             = ffm("config.tokens[].connector", "TBD")
	ConfigTokensCustomClient          = ffm("config.tokens[].customClient", "TBD")
	ConfigTokensExpectContinueTimeout = ffm("config.tokens[].expectContinueTimeout", "TBD")
	ConfigTokensHeaders               = ffm("config.tokens[].headers", "TBD")
	ConfigTokensIdleTimeout           = ffm("config.tokens[].idleTimeout", "TBD")
	ConfigTokensMaxIdleConns          = ffm("config.tokens[].maxIdleConns", "TBD")
	ConfigTokensName                  = ffm("config.tokens[].name", "TBD")
	ConfigTokensPlugin                = ffm("config.tokens[].plugin", "TBD")
	// ConfigTokensRequestTimeout        = ffm("config.tokens[].requestTimeout", "TBD")
	// ConfigTokensTLSHandshakeTimeout = ffm("config.tokens[].tlsHandshakeTimeout", "TBD")
	ConfigTokensURL = ffm("config.tokens[].url", "TBD")

	// ConfigTokensAuthPassword = ffm("config.tokens[].auth.password", "TBD")
	// ConfigTokensAuthUsername = ffm("config.tokens[].auth.username", "TBD")

	ConfigTokensProxyURL = ffm("config.tokens[].proxy.url", "TBD")

	// ConfigTokensRetryCount        = ffm("config.tokens[].retry.count", "TBD")
	// ConfigTokensRetryEnabled      = ffm("config.tokens[].retry.enabled", "TBD")
	// ConfigTokensRetryInitWaitTime = ffm("config.tokens[].retry.initWaitTime", "TBD")
	// ConfigTokensRetryMaxWaitTime  = ffm("config.tokens[].retry.maxWaitTime", "TBD")

	// ConfigTokensWsHeartbeatInterval      = ffm("config.tokens[].ws.heartbeatInterval", "The number of milliseconds to wait between heartbeat signals on the WebSocket connection")
	// ConfigTokensWsInitialConnectAttempts = ffm("config.tokens[].ws.initialConnectAttempts", "The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing")
	// ConfigTokensWsPath                   = ffm("config.tokens[].ws.path", "The WebSocket sever URL to which FireFly should connect")
	// ConfigTokensWsReadBufferSize         = ffm("config.tokens[].ws.readBufferSize", "The size in bytes of the read buffer for the WebSocket connection")
	// ConfigTokensWsWriteBufferSize        = ffm("config.tokens[].ws.writeBufferSize", "The size in bytes of the write buffer for the WebSocket connection")

	// ConfigTransactionCacheSize = ffm("config.transaction.cache.size", "TBD")
	// ConfigTransactionCacheTTL  = ffm("config.transaction.cache.ttl", "TBD")

	ConfigUIEnabled = ffm("config.ui.enabled", "Enables the web user interface")
	ConfigUIPath    = ffm("config.ui.path", "The file system path which contains the static HTML, CSS, and JavaScript files for the user interface")

	// ConfigValidatorCacheSize = ffm("config.validator.cache.size", "TBD")
	// ConfigValidatorCacheTTL  = ffm("config.validator.cache.ttl", "TBD")
)
