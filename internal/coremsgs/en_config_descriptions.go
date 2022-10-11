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

var ffc = func(key, translation, fieldType string) i18n.ConfigMessageKey {
	return i18n.FFC(language.AmericanEnglish, key, translation, fieldType)
}

//revive:disable
var (
	ConfigGlobalMigrationsAuto      = ffc("config.global.migrations.auto", "Enables automatic database migrations", i18n.BooleanType)
	ConfigGlobalMigrationsDirectory = ffc("config.global.migrations.directory", "The directory containing the numerically ordered migration DDL files to apply to the database", i18n.StringType)
	ConfigGlobalShutdownTimeout     = ffc("config.global.shutdownTimeout", "The maximum amount of time to wait for any open HTTP requests to finish before shutting down the HTTP server", i18n.TimeDurationType)

	ConfigLegacyAdmin     = ffc("config.admin.enabled", "Deprecated - use spi.enabled instead", i18n.BooleanType)
	ConfigSPIAddress      = ffc("config.spi.address", "The IP address on which the admin HTTP API should listen", "IP Address "+i18n.StringType)
	ConfigSPIEnabled      = ffc("config.spi.enabled", "Enables the admin HTTP API", i18n.BooleanType)
	ConfigSPIPort         = ffc("config.spi.port", "The port on which the admin HTTP API should listen", i18n.IntType)
	ConfigSPIPublicURL    = ffc("config.spi.publicURL", "The fully qualified public URL for the admin API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation", "URL "+i18n.StringType)
	ConfigSPIReadTimeout  = ffc("config.spi.readTimeout", "The maximum time to wait when reading from an HTTP connection", i18n.TimeDurationType)
	ConfigSPIWriteTimeout = ffc("config.spi.writeTimeout", "The maximum time to wait when writing to an HTTP connection", i18n.TimeDurationType)

	ConfigAPIDefaultFilterLimit        = ffc("config.api.defaultFilterLimit", "The maximum number of rows to return if no limit is specified on an API request", i18n.IntType)
	ConfigAPIMaxFilterLimit            = ffc("config.api.maxFilterLimit", "The largest value of `limit` that an HTTP client can specify in a request", i18n.IntType)
	ConfigAPIRequestMaxTimeout         = ffc("config.api.requestMaxTimeout", "The maximum amount of time that an HTTP client can specify in a `Request-Timeout` header to keep a specific request open", i18n.TimeDurationType)
	ConfigAssetManagerKeyNormalization = ffc("config.asset.manager.keyNormalization", "Mechanism to normalize keys before using them. Valid options are `blockchain_plugin` - use blockchain plugin (default) or `none` - do not attempt normalization (deprecated - use namespaces.predefined[].asset.manager.keyNormalization)", i18n.StringType)

	ConfigBatchManagerMinimumPollDelay = ffc("config.batch.manager.minimumPollDelay", "The minimum time the batch manager waits between polls on the DB - to prevent thrashing", i18n.TimeDurationType)
	ConfigBatchManagerPollTimeout      = ffc("config.batch.manager.pollTimeout", "How long to wait without any notifications of new messages before doing a page query", i18n.TimeDurationType)
	ConfigBatchManagerReadPageSize     = ffc("config.batch.manager.readPageSize", "The size of each page of messages read from the database into memory when assembling batches", i18n.IntType)

	ConfigBlobreceiverWorkerBatchMaxInserts = ffc("config.blobreceiver.worker.batchMaxInserts", "The maximum number of items the blob receiver worker will insert in a batch", i18n.IntType)
	ConfigBlobreceiverWorkerBatchTimeout    = ffc("config.blobreceiver.worker.batchTimeout", "The maximum amount of the the blob receiver worker will wait", i18n.TimeDurationType)
	ConfigBlobreceiverWorkerCount           = ffc("config.blobreceiver.worker.count", "The number of blob receiver workers", i18n.IntType)

	ConfigBlockchainType = ffc("config.blockchain.type", "A string defining which type of blockchain plugin to use. This tells FireFly which type of configuration to load for the rest of the `blockchain` section", i18n.StringType)

	ConfigBlockchainEthereumAddressResolverBodyTemplate          = ffc("config.blockchain.ethereum.addressResolver.bodyTemplate", "The body go template string to use when making HTTP requests", i18n.GoTemplateType)
	ConfigBlockchainEthereumAddressResolverCustomClient          = ffc("config.blockchain.ethereum.addressResolver.customClient", "Used for testing purposes only", i18n.IgnoredType)
	ConfigBlockchainEthereumAddressResolverExpectContinueTimeout = ffc("config.blockchain.ethereum.addressResolver.expectContinueTimeout", "See [ExpectContinueTimeout in the Go docs](https://pkg.go.dev/net/http#Transport)", i18n.TimeDurationType)
	ConfigBlockchainEthereumAddressResolverHeaders               = ffc("config.blockchain.ethereum.addressResolver.headers", "Adds custom headers to HTTP requests", i18n.StringType)
	ConfigBlockchainEthereumAddressResolverIdleTimeout           = ffc("config.blockchain.ethereum.addressResolver.idleTimeout", "The max duration to hold a HTTP keepalive connection between calls", i18n.TimeDurationType)
	ConfigBlockchainEthereumAddressResolverMaxIdleConns          = ffc("config.blockchain.ethereum.addressResolver.maxIdleConns", "The max number of idle connections to hold pooled", i18n.IntType)
	ConfigBlockchainEthereumAddressResolverMethod                = ffc("config.blockchain.ethereum.addressResolver.method", "The HTTP method to use when making requests to the Address Resolver", i18n.StringType)

	ConfigBlockchainEthereumAddressResolverResponseField  = ffc("config.blockchain.ethereum.addressResolver.responseField", "The name of a JSON field that is provided in the response, that contains the ethereum address (default `address`)", i18n.StringType)
	ConfigBlockchainEthereumAddressResolverRetainOriginal = ffc("config.blockchain.ethereum.addressResolver.retainOriginal", "When true the original pre-resolved string is retained after the lookup, and passed down to Ethconnect as the from address", i18n.BooleanType)
	ConfigBlockchainEthereumAddressResolverURL            = ffc("config.blockchain.ethereum.addressResolver.url", "The URL of the Address Resolver", i18n.StringType)
	ConfigBlockchainEthereumAddressResolverURLTemplate    = ffc("config.blockchain.ethereum.addressResolver.urlTemplate", "The URL Go template string to use when calling the Address Resolver", i18n.GoTemplateType)

	ConfigBlockchainEthereumAddressResolverProxyURL = ffc("config.blockchain.ethereum.addressResolver.proxy.url", "Optional HTTP proxy server to use when connecting to the Address Resolver", "URL "+i18n.StringType)

	ConfigBlockchainEthereumEthconnectBatchSize    = ffc("config.blockchain.ethereum.ethconnect.batchSize", "The number of events Ethconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream", i18n.IntType)
	ConfigBlockchainEthereumEthconnectBatchTimeout = ffc("config.blockchain.ethereum.ethconnect.batchTimeout", "How long Ethconnect should wait for new events to arrive and fill a batch, before sending the batch to FireFly core. Only applies when automatically creating a new event stream", i18n.TimeDurationType)
	ConfigBlockchainEthereumEthconnectInstance     = ffc("config.blockchain.ethereum.ethconnect.instance", "The Ethereum address of the FireFly BatchPin smart contract that has been deployed to the blockchain (deprecated - use namespaces.predefined[].multiparty.contract[].location.address)", "Address "+i18n.StringType)
	ConfigBlockchainEthereumEthconnectFromBlock    = ffc("config.blockchain.ethereum.ethconnect.fromBlock", "The first event this FireFly instance should listen to from the BatchPin smart contract. Default=0. Only affects initial creation of the event stream (deprecated - use namespaces.predefined[].multiparty.contract[].location.firstEvent)", "Address "+i18n.StringType)
	ConfigBlockchainEthereumEthconnectPrefixLong   = ffc("config.blockchain.ethereum.ethconnect.prefixLong", "The prefix that will be used for Ethconnect specific HTTP headers when FireFly makes requests to Ethconnect", i18n.StringType)
	ConfigBlockchainEthereumEthconnectPrefixShort  = ffc("config.blockchain.ethereum.ethconnect.prefixShort", "The prefix that will be used for Ethconnect specific query parameters when FireFly makes requests to Ethconnect", i18n.StringType)
	ConfigBlockchainEthereumEthconnectTopic        = ffc("config.blockchain.ethereum.ethconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single ethconnect", i18n.StringType)
	ConfigBlockchainEthereumEthconnectURL          = ffc("config.blockchain.ethereum.ethconnect.url", "The URL of the Ethconnect instance", "URL "+i18n.StringType)
	ConfigBlockchainEthereumEthconnectProxyURL     = ffc("config.blockchain.ethereum.ethconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Ethconnect", "URL "+i18n.StringType)

	ConfigBlockchainEthereumFFTMURL      = ffc("config.blockchain.ethereum.fftm.url", "The URL of the FireFly Transaction Manager runtime, if enabled", i18n.StringType)
	ConfigBlockchainEthereumFFTMProxyURL = ffc("config.blockchain.ethereum.fftm.proxy.url", "Optional HTTP proxy server to use when connecting to the Transaction Manager", i18n.StringType)

	ConfigBlockchainFabricFabconnectBatchSize    = ffc("config.blockchain.fabric.fabconnect.batchSize", "The number of events Fabconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream", i18n.IntType)
	ConfigBlockchainFabricFabconnectBatchTimeout = ffc("config.blockchain.fabric.fabconnect.batchTimeout", "The maximum amount of time to wait for a batch to complete", i18n.TimeDurationType)
	ConfigBlockchainFabricFabconnectChaincode    = ffc("config.blockchain.fabric.fabconnect.chaincode", "The name of the Fabric chaincode that FireFly will use for BatchPin transactions (deprecated - use namespaces.predefined[].multiparty.contract[].location.chaincode)", i18n.StringType)
	ConfigBlockchainFabricFabconnectChannel      = ffc("config.blockchain.fabric.fabconnect.channel", "The Fabric channel that FireFly will use for BatchPin transactions (deprecated - use namespaces.predefined[].multiparty.contract[].location.channel)", i18n.StringType)
	ConfigBlockchainFabricFabconnectPrefixLong   = ffc("config.blockchain.fabric.fabconnect.prefixLong", "The prefix that will be used for Fabconnect specific HTTP headers when FireFly makes requests to Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectPrefixShort  = ffc("config.blockchain.fabric.fabconnect.prefixShort", "The prefix that will be used for Fabconnect specific query parameters when FireFly makes requests to Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectSigner       = ffc("config.blockchain.fabric.fabconnect.signer", "The Fabric signing key to use when submitting transactions to Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectTopic        = ffc("config.blockchain.fabric.fabconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single Fabconnect", i18n.StringType)
	ConfigBlockchainFabricFabconnectURL          = ffc("config.blockchain.fabric.fabconnect.url", "The URL of the Fabconnect instance", "URL "+i18n.StringType)
	ConfigBlockchainFabricFabconnectProxyURL     = ffc("config.blockchain.fabric.fabconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Fabconnect", "URL "+i18n.StringType)

	ConfigCacheEnabled = ffc("config.cache.enabled", "Enables caching, defaults to true", i18n.BooleanType)

	ConfigCacheAddressResolverLimit    = ffc("config.cache.addressresolver.limit", "Max number of cached items for address resolver", i18n.IntType)
	ConfigCacheAddressResolverTTL      = ffc("config.cache.addressresolver.ttl", "Time to live of cached items for address resolver", i18n.StringType)
	ConfigCacheBatchLimit              = ffc("config.cache.batch.limit", "Max number of cached items for batches", i18n.IntType)
	ConfigCacheBatchTTL                = ffc("config.cache.batch.ttl", "Time to live of cache items for batches", i18n.StringType)
	ConfigCacheBlockchainEventLimit    = ffc("config.cache.blockchainevent.limit", "Max number of cached blockchain events for transactions", i18n.IntType)
	ConfigCacheBlockchainEventTTL      = ffc("config.cache.blockchainevent.ttl", "Time to live of cached blockchain events for transactions", i18n.StringType)
	ConfigCacheTransactionSize         = ffc("config.cache.transaction.size", "Max size of cached transactions", i18n.ByteSizeType)
	ConfigCacheTransactionTTL          = ffc("config.cache.transaction.ttl", "Time to live of cached transactions", i18n.StringType)
	ConfigCacheEventListenerTopicLimit = ffc("config.cache.eventlistenertopic.limit", "Max number of cached items for blockchain listener topics", i18n.IntType)
	ConfigCacheEventListenerTopicTTL   = ffc("config.cache.eventlistenertopic.ttl", "Time to live of cached items for blockchain listener topics", i18n.StringType)
	ConfigCacheGroupLimit              = ffc("config.cache.group.limit", "Max number of cached items for groups", i18n.IntType)
	ConfigCacheGroupTTL                = ffc("config.cache.group.ttl", "Time to live of cached items for groups", i18n.StringType)
	ConfigCacheIdentityLimit           = ffc("config.cache.identity.limit", "Max number of cached identities for identity manager", i18n.IntType)
	ConfigCacheIdentityTTL             = ffc("config.cache.identity.ttl", "Time to live of cached identities for identity manager", i18n.StringType)
	ConfigCacheSigningKeyLimit         = ffc("config.cache.signingkey.limit", "Max number of cached signing keys for identity manager", i18n.IntType)
	ConfigCacheSigningKeyTTL           = ffc("config.cache.signingkey.ttl", "Time to live of cached signing keys for identity manager", i18n.StringType)
	ConfigCacheMessageSize             = ffc("config.cache.message.size", "Max size of cached messages for data manager", i18n.ByteSizeType)
	ConfigCacheMessageTTL              = ffc("config.cache.message.ttl", "Time to live of cached messages for data manager", i18n.StringType)
	ConfigCacheValidatorSize           = ffc("config.cache.validator.size", "Max size of cached validators for data manager", i18n.ByteSizeType)
	ConfigCacheValidatorTTL            = ffc("config.cache.validator.ttl", "Time to live of cached validators for data manager", i18n.StringType)
	ConfigCacheBlockchainLimit         = ffc("config.cache.blockchain.limit", "Max number of cached items for blockchain", i18n.IntType)
	ConfigCacheBlockchainTTL           = ffc("config.cache.blockchain.ttl", "Time to live of cached items for blockchain", i18n.StringType)
	ConfigCacheOperationsLimit         = ffc("config.cache.operations.limit", "Max number of cached items for operations", i18n.IntType)
	ConfigCacheOperationsTTL           = ffc("config.cache.operations.ttl", "Time to live of cached items for operations", i18n.StringType)

	ConfigPluginDatabase     = ffc("config.plugins.database", "The list of configured Database plugins", i18n.StringType)
	ConfigPluginDatabaseName = ffc("config.plugins.database[].name", "The name of the Database plugin", i18n.StringType)
	ConfigPluginDatabaseType = ffc("config.plugins.database[].type", "The type of the configured Database plugin", i18n.StringType)

	ConfigPluginDatabasePostgresMaxConnIdleTime = ffc("config.plugins.database[].postgres.maxConnIdleTime", "The maximum amount of time a database connection can be idle", i18n.TimeDurationType)
	ConfigPluginDatabasePostgresMaxConnLifetime = ffc("config.plugins.database[].postgres.maxConnLifetime", "The maximum amount of time to keep a database connection open", i18n.TimeDurationType)
	ConfigPluginDatabasePostgresMaxConns        = ffc("config.plugins.database[].postgres.maxConns", "Maximum connections to the database", i18n.IntType)
	ConfigPluginDatabasePostgresMaxIdleConns    = ffc("config.plugins.database[].postgres.maxIdleConns", "The maximum number of idle connections to the database", i18n.IntType)
	ConfigPluginDatabasePostgresURL             = ffc("config.plugins.database[].postgres.url", "The PostgreSQL connection string for the database", i18n.StringType)

	ConfigPluginDatabaseSqlite3MaxConnIdleTime = ffc("config.plugins.database[].sqlite3.maxConnIdleTime", "The maximum amount of time a database connection can be idle", i18n.TimeDurationType)
	ConfigPluginDatabaseSqlite3MaxConnLifetime = ffc("config.plugins.database[].sqlite3.maxConnLifetime", "The maximum amount of time to keep a database connection open", i18n.TimeDurationType)
	ConfigPluginDatabaseSqlite3MaxConns        = ffc("config.plugins.database[].sqlite3.maxConns", "Maximum connections to the database", i18n.IntType)
	ConfigPluginDatabaseSqlite3MaxIdleConns    = ffc("config.plugins.database[].sqlite3.maxIdleConns", "The maximum number of idle connections to the database", i18n.IntType)
	ConfigPluginDatabaseSqlite3URL             = ffc("config.plugins.database[].sqlite3.url", "The SQLite connection string for the database", i18n.StringType)

	ConfigPluginBlockchain     = ffc("config.plugins.blockchain", "The list of configured Blockchain plugins", i18n.StringType)
	ConfigPluginBlockchainName = ffc("config.plugins.blockchain[].name", "The name of the configured Blockchain plugin", i18n.StringType)
	ConfigPluginBlockchainType = ffc("config.plugins.blockchain[].type", "The type of the configured Blockchain Connector plugin", i18n.StringType)

	ConfigPluginBlockchainEthereumAddressResolverBodyTemplate          = ffc("config.plugins.blockchain[].ethereum.addressResolver.bodyTemplate", "The body go template string to use when making HTTP requests", i18n.GoTemplateType)
	ConfigPluginBlockchainEthereumAddressResolverCustomClient          = ffc("config.plugins.blockchain[].ethereum.addressResolver.customClient", "Used for testing purposes only", i18n.IgnoredType)
	ConfigPluginBlockchainEthereumAddressResolverExpectContinueTimeout = ffc("config.plugins.blockchain[].ethereum.addressResolver.expectContinueTimeout", "See [ExpectContinueTimeout in the Go docs](https://pkg.go.dev/net/http#Transport)", i18n.TimeDurationType)
	ConfigPluginBlockchainEthereumAddressResolverHeaders               = ffc("config.plugins.blockchain[].ethereum.addressResolver.headers", "Adds custom headers to HTTP requests", i18n.StringType)
	ConfigPluginBlockchainEthereumAddressResolverIdleTimeout           = ffc("config.plugins.blockchain[].ethereum.addressResolver.idleTimeout", "The max duration to hold a HTTP keepalive connection between calls", i18n.TimeDurationType)
	ConfigPluginBlockchainEthereumAddressResolverMaxIdleConns          = ffc("config.plugins.blockchain[].ethereum.addressResolver.maxIdleConns", "The max number of idle connections to hold pooled", i18n.IntType)
	ConfigPluginBlockchainEthereumAddressResolverMethod                = ffc("config.plugins.blockchain[].ethereum.addressResolver.method", "The HTTP method to use when making requests to the Address Resolver", i18n.StringType)

	ConfigPluginBlockchainEthereumAddressResolverResponseField  = ffc("config.plugins.blockchain[].ethereum.addressResolver.responseField", "The name of a JSON field that is provided in the response, that contains the ethereum address (default `address`)", i18n.StringType)
	ConfigPluginBlockchainEthereumAddressResolverRetainOriginal = ffc("config.plugins.blockchain[].ethereum.addressResolver.retainOriginal", "When true the original pre-resolved string is retained after the lookup, and passed down to Ethconnect as the from address", i18n.BooleanType)
	ConfigPluginBlockchainEthereumAddressResolverURL            = ffc("config.plugins.blockchain[].ethereum.addressResolver.url", "The URL of the Address Resolver", i18n.StringType)
	ConfigPluginBlockchainEthereumAddressResolverURLTemplate    = ffc("config.plugins.blockchain[].ethereum.addressResolver.urlTemplate", "The URL Go template string to use when calling the Address Resolver", i18n.GoTemplateType)

	ConfigPluginBlockchainEthereumAddressResolverProxyURL = ffc("config.plugins.blockchain[].ethereum.addressResolver.proxy.url", "Optional HTTP proxy server to use when connecting to the Address Resolver", "URL "+i18n.StringType)

	ConfigPluginBlockchainEthereumEthconnectBatchSize    = ffc("config.plugins.blockchain[].ethereum.ethconnect.batchSize", "The number of events Ethconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream", i18n.IntType)
	ConfigPluginBlockchainEthereumEthconnectBatchTimeout = ffc("config.plugins.blockchain[].ethereum.ethconnect.batchTimeout", "How long Ethconnect should wait for new events to arrive and fill a batch, before sending the batch to FireFly core. Only applies when automatically creating a new event stream", i18n.TimeDurationType)
	ConfigPluginBlockchainEthereumEthconnectInstance     = ffc("config.plugins.blockchain[].ethereum.ethconnect.instance", "The Ethereum address of the FireFly BatchPin smart contract that has been deployed to the blockchain", "Address "+i18n.StringType)
	ConfigPluginBlockchainEthereumEthconnectFromBlock    = ffc("config.plugins.blockchain[].ethereum.ethconnect.fromBlock", "The first event this FireFly instance should listen to from the BatchPin smart contract. Default=0. Only affects initial creation of the event stream", "Address "+i18n.StringType)
	ConfigPluginBlockchainEthereumEthconnectPrefixLong   = ffc("config.plugins.blockchain[].ethereum.ethconnect.prefixLong", "The prefix that will be used for Ethconnect specific HTTP headers when FireFly makes requests to Ethconnect", i18n.StringType)
	ConfigPluginBlockchainEthereumEthconnectPrefixShort  = ffc("config.plugins.blockchain[].ethereum.ethconnect.prefixShort", "The prefix that will be used for Ethconnect specific query parameters when FireFly makes requests to Ethconnect", i18n.StringType)
	ConfigPluginBlockchainEthereumEthconnectTopic        = ffc("config.plugins.blockchain[].ethereum.ethconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single ethconnect", i18n.StringType)
	ConfigPluginBlockchainEthereumEthconnectURL          = ffc("config.plugins.blockchain[].ethereum.ethconnect.url", "The URL of the Ethconnect instance", "URL "+i18n.StringType)
	ConfigPluginBlockchainEthereumEthconnectProxyURL     = ffc("config.plugins.blockchain[].ethereum.ethconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Ethconnect", "URL "+i18n.StringType)

	ConfigPluginBlockchainEthereumFFTMURL      = ffc("config.plugins.blockchain[].ethereum.fftm.url", "The URL of the FireFly Transaction Manager runtime, if enabled", i18n.StringType)
	ConfigPluginBlockchainEthereumFFTMProxyURL = ffc("config.plugins.blockchain[].ethereum.fftm.proxy.url", "Optional HTTP proxy server to use when connecting to the Transaction Manager", i18n.StringType)

	ConfigPluginBlockchainFabricFabconnectBatchSize    = ffc("config.plugins.blockchain[].fabric.fabconnect.batchSize", "The number of events Fabconnect should batch together for delivery to FireFly core. Only applies when automatically creating a new event stream", i18n.IntType)
	ConfigPluginBlockchainFabricFabconnectBatchTimeout = ffc("config.plugins.blockchain[].fabric.fabconnect.batchTimeout", "The maximum amount of time to wait for a batch to complete", i18n.TimeDurationType)
	ConfigPluginBlockchainFabricFabconnectPrefixLong   = ffc("config.plugins.blockchain[].fabric.fabconnect.prefixLong", "The prefix that will be used for Fabconnect specific HTTP headers when FireFly makes requests to Fabconnect", i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectPrefixShort  = ffc("config.plugins.blockchain[].fabric.fabconnect.prefixShort", "The prefix that will be used for Fabconnect specific query parameters when FireFly makes requests to Fabconnect", i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectSigner       = ffc("config.plugins.blockchain[].fabric.fabconnect.signer", "The Fabric signing key to use when submitting transactions to Fabconnect", i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectTopic        = ffc("config.plugins.blockchain[].fabric.fabconnect.topic", "The websocket listen topic that the node should register on, which is important if there are multiple nodes using a single Fabconnect", i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectURL          = ffc("config.plugins.blockchain[].fabric.fabconnect.url", "The URL of the Fabconnect instance", "URL "+i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectProxyURL     = ffc("config.plugins.blockchain[].fabric.fabconnect.proxy.url", "Optional HTTP proxy server to use when connecting to Fabconnect", "URL "+i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectChaincode    = ffc("config.plugins.blockchain[].fabric.fabconnect.chaincode", "The name of the Fabric chaincode that FireFly will use for BatchPin transactions (deprecated - use fireflyContract[].chaincode)", i18n.StringType)
	ConfigPluginBlockchainFabricFabconnectChannel      = ffc("config.plugins.blockchain[].fabric.fabconnect.channel", "The Fabric channel that FireFly will use for BatchPin transactions", i18n.StringType)

	ConfigBroadcastBatchAgentTimeout = ffc("config.broadcast.batch.agentTimeout", "How long to keep around a batching agent for a sending identity before disposal", i18n.StringType)
	ConfigBroadcastBatchPayloadLimit = ffc("config.broadcast.batch.payloadLimit", "The maximum payload size of a batch for broadcast messages", i18n.ByteSizeType)
	ConfigBroadcastBatchSize         = ffc("config.broadcast.batch.size", "The maximum number of messages that can be packed into a batch", i18n.IntType)
	ConfigBroadcastBatchTimeout      = ffc("config.broadcast.batch.timeout", "The timeout to wait for a batch to fill, before sending", i18n.TimeDurationType)

	ConfigDatabaseType = ffc("config.database.type", "The type of the database interface plugin to use", i18n.IntType)

	ConfigDatabasePostgresMaxConnIdleTime = ffc("config.database.postgres.maxConnIdleTime", "The maximum amount of time a database connection can be idle", i18n.TimeDurationType)
	ConfigDatabasePostgresMaxConnLifetime = ffc("config.database.postgres.maxConnLifetime", "The maximum amount of time to keep a database connection open", i18n.TimeDurationType)
	ConfigDatabasePostgresMaxConns        = ffc("config.database.postgres.maxConns", "Maximum connections to the database", i18n.IntType)
	ConfigDatabasePostgresMaxIdleConns    = ffc("config.database.postgres.maxIdleConns", "The maximum number of idle connections to the database", i18n.IntType)
	ConfigDatabasePostgresURL             = ffc("config.database.postgres.url", "The PostgreSQL connection string for the database", i18n.StringType)

	ConfigDatabaseSqlite3MaxConnIdleTime = ffc("config.database.sqlite3.maxConnIdleTime", "The maximum amount of time a database connection can be idle", i18n.TimeDurationType)
	ConfigDatabaseSqlite3MaxConnLifetime = ffc("config.database.sqlite3.maxConnLifetime", "The maximum amount of time to keep a database connection open", i18n.TimeDurationType)
	ConfigDatabaseSqlite3MaxConns        = ffc("config.database.sqlite3.maxConns", "Maximum connections to the database", i18n.IntType)
	ConfigDatabaseSqlite3MaxIdleConns    = ffc("config.database.sqlite3.maxIdleConns", "The maximum number of idle connections to the database", i18n.IntType)
	ConfigDatabaseSqlite3URL             = ffc("config.database.sqlite3.url", "The SQLite connection string for the database", i18n.StringType)

	ConfigDataexchangeType = ffc("config.dataexchange.type", "The Data Exchange plugin to use", i18n.StringType)

	ConfigDataexchangeFfdxInitEnabled     = ffc("config.dataexchange.ffdx.initEnabled", "Instructs FireFly to always post all current nodes to the `/init` API before connecting or reconnecting to the connector", i18n.BooleanType)
	ConfigDataexchangeFfdxManifestEnabled = ffc("config.dataexchange.ffdx.manifestEnabled", "Determines whether to require+validate a manifest from other DX instances in the network. Must be supported by the connector", i18n.StringType)
	ConfigDataexchangeFfdxURL             = ffc("config.dataexchange.ffdx.url", "The URL of the Data Exchange instance", "URL "+i18n.StringType)

	ConfigDataexchangeFfdxProxyURL = ffc("config.dataexchange.ffdx.proxy.url", "Optional HTTP proxy server to use when connecting to the Data Exchange", "URL "+i18n.StringType)

	ConfigPluginDataexchange     = ffc("config.plugins.dataexchange", "The array of configured Data Exchange plugins ", i18n.StringType)
	ConfigPluginDataexchangeType = ffc("config.plugins.dataexchange[].type", "The Data Exchange plugin to use", i18n.StringType)
	ConfigPluginDataexchangeName = ffc("config.plugins.dataexchange[].name", "The name of the configured Data Exchange plugin", i18n.StringType)

	ConfigPluginDataexchangeFfdxInitEnabled     = ffc("config.plugins.dataexchange[].ffdx.initEnabled", "Instructs FireFly to always post all current nodes to the `/init` API before connecting or reconnecting to the connector", i18n.BooleanType)
	ConfigPluginDataexchangeFfdxManifestEnabled = ffc("config.plugins.dataexchange[].ffdx.manifestEnabled", "Determines whether to require+validate a manifest from other DX instances in the network. Must be supported by the connector", i18n.StringType)
	ConfigPluginDataexchangeFfdxURL             = ffc("config.plugins.dataexchange[].ffdx.url", "The URL of the Data Exchange instance", "URL "+i18n.StringType)

	ConfigPluginDataexchangeFfdxProxyURL = ffc("config.plugins.dataexchange[].ffdx.proxy.url", "Optional HTTP proxy server to use when connecting to the Data Exchange", "URL "+i18n.StringType)

	ConfigDebugPort    = ffc("config.debug.port", "An HTTP port on which to enable the go debugger", i18n.IntType)
	ConfigDebugAddress = ffc("config.debug.address", "The HTTP interface the go debugger binds to", i18n.StringType)

	ConfigDownloadWorkerCount       = ffc("config.download.worker.count", "The number of download workers", i18n.IntType)
	ConfigDownloadWorkerQueueLength = ffc("config.download.worker.queueLength", "The length of the work queue in the channel to the workers - defaults to 2x the worker count", i18n.IntType)

	ConfigEventAggregatorBatchSize         = ffc("config.event.aggregator.batchSize", "The maximum number of records to read from the DB before performing an aggregation run", i18n.ByteSizeType)
	ConfigEventAggregatorBatchTimeout      = ffc("config.event.aggregator.batchTimeout", "How long to wait for new events to arrive before performing aggregation on a page of events", i18n.TimeDurationType)
	ConfigEventAggregatorFirstEvent        = ffc("config.event.aggregator.firstEvent", "The first event the aggregator should process, if no previous offest is stored in the DB. Valid options are `oldest` or `newest`", i18n.StringType)
	ConfigEventAggregatorPollTimeout       = ffc("config.event.aggregator.pollTimeout", "The time to wait without a notification of new events, before trying a select on the table", i18n.TimeDurationType)
	ConfigEventAggregatorRewindQueueLength = ffc("config.event.aggregator.rewindQueueLength", "The size of the queue into the rewind dispatcher", i18n.IntType)
	ConfigEventAggregatorRewindTimout      = ffc("config.event.aggregator.rewindTimeout", "The minimum time to wait for rewinds to accumulate before resolving them", i18n.TimeDurationType)
	ConfigEventAggregatorRewindQueryLimit  = ffc("config.event.aggregator.rewindQueryLimit", "Safety limit on the maximum number of records to search when performing queries to search for rewinds", i18n.IntType)
	ConfigEventDbeventsBufferSize          = ffc("config.event.dbevents.bufferSize", "The size of the buffer of change events", i18n.ByteSizeType)

	ConfigEventDispatcherBatchTimeout = ffc("config.event.dispatcher.batchTimeout", "A short time to wait for new events to arrive before re-polling for new events", i18n.TimeDurationType)
	ConfigEventDispatcherBufferLength = ffc("config.event.dispatcher.bufferLength", "The number of events + attachments an individual dispatcher should hold in memory ready for delivery to the subscription", i18n.IntType)
	ConfigEventDispatcherPollTimeout  = ffc("config.event.dispatcher.pollTimeout", "The time to wait without a notification of new events, before trying a select on the table", i18n.TimeDurationType)

	ConfigEventTransportsDefault = ffc("config.event.transports.default", "The default event transport for new subscriptions", i18n.StringType)
	ConfigEventTransportsEnabled = ffc("config.event.transports.enabled", "Which event interface plugins are enabled", i18n.BooleanType)

	ConfigHistogramsMaxChartRows = ffc("config.histograms.maxChartRows", "The maximum rows to fetch for each histogram bucket", i18n.IntType)

	ConfigHTTPAddress      = ffc("config.http.address", "The IP address on which the HTTP API should listen", "IP Address "+i18n.StringType)
	ConfigHTTPPort         = ffc("config.http.port", "The port on which the HTTP API should listen", i18n.IntType)
	ConfigHTTPPublicURL    = ffc("config.http.publicURL", "The fully qualified public URL for the API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation", "URL "+i18n.StringType)
	ConfigHTTPReadTimeout  = ffc("config.http.readTimeout", "The maximum time to wait when reading from an HTTP connection", i18n.TimeDurationType)
	ConfigHTTPWriteTimeout = ffc("config.http.writeTimeout", "The maximum time to wait when writing to an HTTP connection", i18n.TimeDurationType)

	ConfigPluginIdentity     = ffc("config.plugins.identity", "The list of available Identity plugins", i18n.StringType)
	ConfigPluginIdentityType = ffc("config.plugins.identity[].type", "The type of a configured Identity plugin", i18n.StringType)
	ConfigPluginIdentityName = ffc("config.plugins.identity[].name", "The name of a configured Identity plugin", i18n.StringType)

	ConfigIdentityManagerLegacySystemIdentitites = ffc("config.identity.manager.legacySystemIdentities", "Whether the identity manager should resolve legacy identities registered on the ff_system namespace", i18n.BooleanType)

	ConfigLogCompress   = ffc("config.log.compress", "Determines if the rotated log files should be compressed using gzip", i18n.BooleanType)
	ConfigLogFilename   = ffc("config.log.filename", "Filename is the file to write logs to.  Backup log files will be retained in the same directory", i18n.StringType)
	ConfigLogFilesize   = ffc("config.log.filesize", "MaxSize is the maximum size the log file before it gets rotated", i18n.ByteSizeType)
	ConfigLogForceColor = ffc("config.log.forceColor", "Force color to be enabled, even when a non-TTY output is detected", i18n.BooleanType)
	ConfigLogLevel      = ffc("config.log.level", "The log level - error, warn, info, debug, trace", i18n.StringType)
	ConfigLogMaxAge     = ffc("config.log.maxAge", "The maximum time to retain old log files based on the timestamp encoded in their filename", i18n.TimeDurationType)
	ConfigLogMaxBackups = ffc("config.log.maxBackups", "Maximum number of old log files to retain", i18n.IntType)
	ConfigLogNoColor    = ffc("config.log.noColor", "Force color to be disabled, event when TTY output is detected", i18n.BooleanType)
	ConfigLogTimeFormat = ffc("config.log.timeFormat", "Custom time format for logs", i18n.TimeFormatType)
	ConfigLogUtc        = ffc("config.log.utc", "Use UTC timestamps for logs", i18n.BooleanType)

	ConfigMessageWriterBatchMaxInserts = ffc("config.message.writer.batchMaxInserts", "The maximum number of database inserts to include when writing a single batch of messages + data", i18n.IntType)
	ConfigMessageWriterBatchTimeout    = ffc("config.message.writer.batchTimeout", "How long to wait for more messages to arrive before flushing the batch", i18n.TimeDurationType)
	ConfigMessageWriterCount           = ffc("config.message.writer.count", "The number of message writer workers", i18n.IntType)

	ConfigMetricsAddress      = ffc("config.metrics.address", "The IP address on which the metrics HTTP API should listen", i18n.IntType)
	ConfigMetricsEnabled      = ffc("config.metrics.enabled", "Enables the metrics API", i18n.BooleanType)
	ConfigMetricsPath         = ffc("config.metrics.path", "The path from which to serve the Prometheus metrics", i18n.StringType)
	ConfigMetricsPort         = ffc("config.metrics.port", "The port on which the metrics HTTP API should listen", i18n.IntType)
	ConfigMetricsPublicURL    = ffc("config.metrics.publicURL", "The fully qualified public URL for the metrics API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation", "URL "+i18n.StringType)
	ConfigMetricsReadTimeout  = ffc("config.metrics.readTimeout", "The maximum time to wait when reading from an HTTP connection", i18n.TimeDurationType)
	ConfigMetricsWriteTimeout = ffc("config.metrics.writeTimeout", "The maximum time to wait when writing to an HTTP connection", i18n.TimeDurationType)

	ConfigNamespacesDefault                      = ffc("config.namespaces.default", "The default namespace - must be in the predefined list", i18n.StringType)
	ConfigNamespacesPredefined                   = ffc("config.namespaces.predefined", "A list of namespaces to ensure exists, without requiring a broadcast from the network", "List "+i18n.StringType)
	ConfigNamespacesPredefinedName               = ffc("config.namespaces.predefined[].name", "The name of the namespace (must be unique)", i18n.StringType)
	ConfigNamespacesPredefinedDescription        = ffc("config.namespaces.predefined[].description", "A description for the namespace", i18n.StringType)
	ConfigNamespacesPredefinedPlugins            = ffc("config.namespaces.predefined[].plugins", "The list of plugins for this namespace", i18n.StringType)
	ConfigNamespacesPredefinedDefaultKey         = ffc("config.namespaces.predefined[].defaultKey", "A default signing key for blockchain transactions within this namespace", i18n.StringType)
	ConfigNamespacesPredefinedKeyNormalization   = ffc("config.namespaces.predefined[].asset.manager.keyNormalization", "Mechanism to normalize keys before using them. Valid options are `blockchain_plugin` - use blockchain plugin (default) or `none` - do not attempt normalization", i18n.StringType)
	ConfigNamespacesMultipartyEnabled            = ffc("config.namespaces.predefined[].multiparty.enabled", "Enables multi-party mode for this namespace (defaults to true if an org name or key is configured, either here or at the root level)", i18n.BooleanType)
	ConfigNamespacesMultipartyNetworkNamespace   = ffc("config.namespaces.predefined[].multiparty.networknamespace", "The shared namespace name to be sent in multiparty messages, if it differs from the local namespace name", i18n.StringType)
	ConfigNamespacesMultipartyOrgName            = ffc("config.namespaces.predefined[].multiparty.org.name", "A short name for the local root organization within this namespace", i18n.StringType)
	ConfigNamespacesMultipartyOrgDesc            = ffc("config.namespaces.predefined[].multiparty.org.description", "A description for the local root organization within this namespace", i18n.StringType)
	ConfigNamespacesMultipartyOrgKey             = ffc("config.namespaces.predefined[].multiparty.org.key", "The signing key allocated to the root organization within this namespace", i18n.StringType)
	ConfigNamespacesMultipartyNodeName           = ffc("config.namespaces.predefined[].multiparty.node.name", "The node name for this namespace", i18n.StringType)
	ConfigNamespacesMultipartyNodeDescription    = ffc("config.namespaces.predefined[].multiparty.node.description", "A description for the node in this namespace", i18n.StringType)
	ConfigNamespacesMultipartyContract           = ffc("config.namespaces.predefined[].contract", "A list containing configuration for the multi-party blockchain contract", i18n.StringType)
	ConfigNamespacesMultipartyContractFirstEvent = ffc("config.namespaces.predefined[].multiparty.contract[].firstEvent", "The first event the contract should process. Valid options are `oldest` or `newest`", i18n.StringType)
	ConfigNamespacesMultipartyContractLocation   = ffc("config.namespaces.predefined[].multiparty.contract[].location", "A blockchain-specific contract location. For example, an Ethereum contract address, or a Fabric chaincode name and channe", i18n.StringType)

	ConfigNodeDescription = ffc("config.node.description", "The description of this FireFly node", i18n.StringType)
	ConfigNodeName        = ffc("config.node.name", "The name of this FireFly node", i18n.StringType)

	ConfigOpupdateWorkerBatchMaxInserts = ffc("config.opupdate.worker.batchMaxInserts", "The maximum number of database inserts to include when writing a single batch of messages + data", i18n.IntType)
	ConfigOpupdateWorkerBatchTimeout    = ffc("config.opupdate.worker.batchTimeout", "How long to wait for more messages to arrive before flushing the batch", i18n.TimeDurationType)
	ConfigOpupdateWorkerCount           = ffc("config.opupdate.worker.count", "The number of operation update works", i18n.IntType)
	ConfigOpupdateWorkerQueueLength     = ffc("config.opupdate.worker.queueLength", "The size of the queue for the Operation Update worker", i18n.IntType)

	ConfigOrchestratorStartupAttempts = ffc("config.orchestrator.startupAttempts", "The number of times to attempt to connect to core infrastructure on startup", i18n.StringType)

	ConfigOrgDescription = ffc("config.org.description", "A description of the organization to which this FireFly node belongs (deprecated - should be set on each multi-party namespace instead)", i18n.StringType)
	ConfigOrgKey         = ffc("config.org.key", "The signing key allocated to the organization (deprecated - should be set on each multi-party namespace instead)", i18n.StringType)
	ConfigOrgName        = ffc("config.org.name", "The name of the organization to which this FireFly node belongs (deprecated - should be set on each multi-party namespace instead)", i18n.StringType)

	ConfigPrivatemessagingBatchAgentTimeout = ffc("config.privatemessaging.batch.agentTimeout", "How long to keep around a batching agent for a sending identity before disposal", i18n.TimeDurationType)
	ConfigPrivatemessagingBatchPayloadLimit = ffc("config.privatemessaging.batch.payloadLimit", "The maximum payload size of a private message Data Exchange payload", i18n.ByteSizeType)
	ConfigPrivatemessagingBatchSize         = ffc("config.privatemessaging.batch.size", "The maximum number of messages in a batch for private messages", i18n.IntType)
	ConfigPrivatemessagingBatchTimeout      = ffc("config.privatemessaging.batch.timeout", "The timeout to wait for a batch to fill, before sending", i18n.TimeDurationType)

	ConfigSharedstorageType                = ffc("config.sharedstorage.type", "The Shared Storage plugin to use", i18n.StringType)
	ConfigSharedstorageIpfsAPIURL          = ffc("config.sharedstorage.ipfs.api.url", "The URL for the IPFS API", "URL "+i18n.StringType)
	ConfigSharedstorageIpfsAPIProxyURL     = ffc("config.sharedstorage.ipfs.api.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS API", "URL "+i18n.StringType)
	ConfigSharedstorageIpfsGatewayURL      = ffc("config.sharedstorage.ipfs.gateway.url", "The URL for the IPFS Gateway", "URL "+i18n.StringType)
	ConfigSharedstorageIpfsGatewayProxyURL = ffc("config.sharedstorage.ipfs.gateway.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS Gateway", "URL "+i18n.StringType)

	ConfigPluginSharedstorage                    = ffc("config.plugins.sharedstorage", "The list of configured Shared Storage plugins", i18n.StringType)
	ConfigPluginSharedstorageName                = ffc("config.plugins.sharedstorage[].name", "The name of the Shared Storage plugin to use", i18n.StringType)
	ConfigPluginSharedstorageType                = ffc("config.plugins.sharedstorage[].type", "The Shared Storage plugin to use", i18n.StringType)
	ConfigPluginSharedstorageIpfsAPIURL          = ffc("config.plugins.sharedstorage[].ipfs.api.url", "The URL for the IPFS API", "URL "+i18n.StringType)
	ConfigPluginSharedstorageIpfsAPIProxyURL     = ffc("config.plugins.sharedstorage[].ipfs.api.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS API", "URL "+i18n.StringType)
	ConfigPluginSharedstorageIpfsGatewayURL      = ffc("config.plugins.sharedstorage[].ipfs.gateway.url", "The URL for the IPFS Gateway", "URL "+i18n.StringType)
	ConfigPluginSharedstorageIpfsGatewayProxyURL = ffc("config.plugins.sharedstorage[].ipfs.gateway.proxy.url", "Optional HTTP proxy server to use when connecting to the IPFS Gateway", "URL "+i18n.StringType)

	ConfigSubscriptionMax               = ffc("config.subscription.max", "The maximum number of pre-defined subscriptions that can exist (note for high fan-out consider connecting a dedicated pub/sub broker to the dispatcher)", i18n.IntType)
	ConfigSubscriptionDefaultsBatchSize = ffc("config.subscription.defaults.batchSize", "Default read ahead to enable for subscriptions that do not explicitly configure readahead", i18n.IntType)

	ConfigTokensName     = ffc("config.tokens[].name", "A name to identify this token plugin", i18n.StringType)
	ConfigTokensPlugin   = ffc("config.tokens[].plugin", "The type of the token plugin to use", i18n.StringType)
	ConfigTokensURL      = ffc("config.tokens[].url", "The URL of the token connector", "URL "+i18n.StringType)
	ConfigTokensProxyURL = ffc("config.tokens[].proxy.url", "Optional HTTP proxy server to use when connecting to the token connector", "URL "+i18n.StringType)

	ConfigPluginTokens              = ffc("config.plugins.tokens", "The token plugin configurations", i18n.StringType)
	ConfigPluginTokensName          = ffc("config.plugins.tokens[].name", "A name to identify this token plugin", i18n.StringType)
	ConfigPluginTokensBroadcastName = ffc("config.plugins.tokens[].broadcastName", "The name to be used in broadcast messages related to this token plugin, if it differs from the local plugin name", i18n.StringType)
	ConfigPluginTokensType          = ffc("config.plugins.tokens[].type", "The type of the token plugin to use", i18n.StringType)
	ConfigPluginTokensURL           = ffc("config.plugins.tokens[].fftokens.url", "The URL of the token connector", "URL "+i18n.StringType)
	ConfigPluginTokensProxyURL      = ffc("config.plugins.tokens[].fftokens.proxy.url", "Optional HTTP proxy server to use when connecting to the token connector", "URL "+i18n.StringType)

	ConfigUIEnabled = ffc("config.ui.enabled", "Enables the web user interface", i18n.BooleanType)
	ConfigUIPath    = ffc("config.ui.path", "The file system path which contains the static HTML, CSS, and JavaScript files for the user interface", i18n.StringType)

	ConfigAPIOASPanicOnMissingDescription = ffc("config.api.oas.panicOnMissingDescription", "Used for testing purposes only", i18n.IgnoredType)

	ConfigSPIWebSocketBlockedWarnInternal = ffc("config.spi.ws.blockedWarnInterval", "How often to log warnings in core, when an admin change event listener falls behind the stream they requested and misses events", i18n.TimeDurationType)
	ConfigSPIWebSocketEventQueueLength    = ffc("config.spi.ws.eventQueueLength", "Server-side queue length for events waiting for delivery over an admin change event listener websocket", i18n.IntType)

	ConfigPluginsAuth     = ffc("config.plugins.auth", "Authorization plugin configuration", i18n.MapStringStringType)
	ConfigPluginsAuthName = ffc("config.plugins.auth[].name", "The name of the auth plugin to use", i18n.StringType)
	ConfigPluginsAuthType = ffc("config.plugins.auth[].type", "The type of the auth plugin to use", i18n.StringType)

	ConfigPluginsEventSystemReadAhead           = ffc("config.events.system.readAhead", "", i18n.IgnoredType)
	ConfigPluginsEventWebhooksURL               = ffc("config.events.webhooks.url", "", i18n.IgnoredType)
	ConfigPluginsEventWebSocketsReadBufferSize  = ffc("config.events.websockets.readBufferSize", "WebSocket read buffer size", i18n.ByteSizeType)
	ConfigPluginsEventWebSocketsWriteBufferSize = ffc("config.events.websockets.writeBufferSize", "WebSocket write buffer size", i18n.ByteSizeType)
)
