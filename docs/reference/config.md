---
layout: default
title: Configuration Reference
parent: Reference
nav_order: 3
---

# Configuration Reference
{: .no_toc }

<!-- ## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc} -->

---


## admin

|Key|Default Value|Description|
|---|-------------|-----------|
|address|`127.0.0.1`|The IP address on which the admin HTTP API should listen|
|enabled|`<nil>`|Enables the admin HTTP API|
|port|`5001`|The port on which the admin HTTP API should listen|
|preinit|`<nil>`|Enables the pre-init mode. This mode will let the FireFly Core process start, but not initialize any plugins, besides the database to read any configuration overrides. This allows the admin HTTP API to be used to define custom configuration before starting the rest of FireFly Core.|
|publicURL|`<nil>`|The fully qualified public URL for the admin API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.|
|readTimeout|`15s`|The maximum time to wait in seconds when reading from an HTTP connection|
|writeTimeout|`15s`|The maximum time to wait in seconds when writing to an HTTP connection|

## admin.tls

|Key|Default Value|Description|
|---|-------------|-----------|
|caFile|`<nil>`|The path to the CA file for the admin API|
|certFile|`<nil>`|The path to the certificate file for the admin API|
|clientAuth|`<nil>`|Enables or disables client auth for the admin API|
|enabled|`false`|Enables or disables TLS on the admin API|
|keyFile|`<nil>`|The path to the private key file for the admin API|

## admin.ws

|Key|Default Value|Description|
|---|-------------|-----------|
|blockedWarnInterval|`<nil>`|How often to log warnings in core, when an admin change event listener falls behind the stream they requested and misses events|
|eventQueueLength|`<nil>`|Server-side queue length for events waiting for delivery over an admin change event listener websocket|
|readBufferSize|`<nil>`|The size in bytes of the read buffer for the WebSocket connection|
|writeBufferSize|`<nil>`|The size in bytes of the write buffer for the WebSocket connection|

## api

|Key|Default Value|Description|
|---|-------------|-----------|
|defaultFilterLimit|`<nil>`|The maximum number of rows to return if no limit is specified on an API request|
|maxFilterLimit|`<nil>`|The maximum number of rows to return if no limit is specified on an API request|
|requestMaxTimeout|`<nil>`|The maximum amount of time, in milliseconds that an HTTP client can specify in a `Request-Timeout` header to keep a specific request open|
|requestTimeout|`<nil>`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|shutdownTimeout|`<nil>`|The maximum amount of time, in milliseconds to wait for any open HTTP requests to finish before shutting down the HTTP server|

## api.oas

|Key|Default Value|Description|
|---|-------------|-----------|
|panicOnMissingDescription|`<nil>`|Used when building FireFly to verify all structures and APIs have documentation|

## asset.manager

|Key|Default Value|Description|
|---|-------------|-----------|
|keyNormalization|`<nil>`|Mechanism to normalize keys before using them. Valid options are `blockchain_plugin` - use blockchain plugin (default) or `none` - do not attempt normalization|

## batch.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## batch.manager

|Key|Default Value|Description|
|---|-------------|-----------|
|minimumPollDelay|`<nil>`|The minimum time the batch manager waits between polls on the DB - to prevent thrashing|
|pollTimeout|`<nil>`|How long to wait without any notifications of new messages before doing a page query|
|readPageSize|`<nil>`|The size of each page of messages read from the database into memory when assembling batches|

## batch.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## blobreceiver.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initialDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## blobreceiver.worker

|Key|Default Value|Description|
|---|-------------|-----------|
|batchMaxInserts|`<nil>`|The maximum number of items the blob receiver worker will insert in a batch|
|batchTimeout|`<nil>`|The maximum amount of the the blob receiver worker will wait|
|count|`<nil>`|The number of blob receiver worker|

## blockchain

|Key|Default Value|Description|
|---|-------------|-----------|
|type|`<nil>`|A string defining which type of blockchain plugin to use. This tells FireFly which type of configuration to load for the rest of the `blockchain` section.|

## blockchain.ethereum.addressResolver

|Key|Default Value|Description|
|---|-------------|-----------|
|bodyTemplate|`<nil>`|TBD|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|maxIdleConns|`100`|TBD|
|method|`GET`|The HTTP method to use when making requests to the address resolver|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|responseField|`address`|TBD|
|retainOriginal|`<nil>`|TBD|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|The URL of the address resolver|
|urlTemplate|`<nil>`|TBD|

## blockchain.ethereum.addressResolver.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## blockchain.ethereum.addressResolver.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`1000`|The size of the cache|
|ttl|`24h`|The time to live (TTL) for the cache|

## blockchain.ethereum.addressResolver.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|The URL of the address resolver proxy|

## blockchain.ethereum.addressResolver.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## blockchain.ethereum.ethconnect

|Key|Default Value|Description|
|---|-------------|-----------|
|batchSize|`50`|The maximum number of transactions to send in a single request to Ethconnect|
|batchTimeout|`500`|The maximum amount of time in milliseconds to wait for a batch to complete|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|instance|`<nil>`|The Ethereum address of the FireFly BatchPin smart contract that has been deployed to the blockchain|
|maxIdleConns|`100`|TBD|
|prefixLong|`firefly`|The prefix that will be used for Ethconnect specific HTTP headers when FireFly makes requests to Ethconnect|
|prefixShort|`fly`|The prefix that will be used for Ethconnect specific query parameters when FireFly makes requests to Ethconnect|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|topic|`<nil>`|TBD|
|url|`<nil>`|The URL of the Ethconnect instance|

## blockchain.ethereum.ethconnect.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## blockchain.ethereum.ethconnect.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|The URL of the Ethconnect proxy|

## blockchain.ethereum.ethconnect.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## blockchain.ethereum.ethconnect.ws

|Key|Default Value|Description|
|---|-------------|-----------|
|heartbeatInterval|`30s`|The number of milliseconds to wait between heartbeat signals on the WebSocket connection|
|initialConnectAttempts|`5`|The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing|
|path|`<nil>`|The WebSocket sever URL to which FireFly should connect|
|readBufferSize|`16Kb`|The size in bytes of the read buffer for the WebSocket connection|
|writeBufferSize|`16Kb`|The size in bytes of the write buffer for the WebSocket connection|

## blockchain.fabric.fabconnect

|Key|Default Value|Description|
|---|-------------|-----------|
|batchSize|`50`|The maximum number of transactions to send in a single request to Fabconnect|
|batchTimeout|`500`|The maximum amount of time in milliseconds to wait for a batch to complete|
|chaincode|`<nil>`|The name of the Fabric chaincode that FireFly will use for BatchPin transactions|
|channel|`<nil>`|The Fabric channel that FireFly will use for BatchPin transactions|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|maxIdleConns|`100`|TBD|
|prefixLong|`firefly`|The prefix that will be used for Fabconnect specific HTTP headers when FireFly makes requests to Fabconnect|
|prefixShort|`fly`|The prefix that will be used for Fabconnect specific query parameters when FireFly makes requests to Fabconnect|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|signer|`<nil>`|The Fabric signing key to use when submitting transactions to Fabconnect|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|topic|`<nil>`|TBD|
|url|`<nil>`|The URL of the Fabconnect instance|

## blockchain.fabric.fabconnect.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## blockchain.fabric.fabconnect.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|The URL for the Fabconnect proxy|

## blockchain.fabric.fabconnect.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## blockchain.fabric.fabconnect.ws

|Key|Default Value|Description|
|---|-------------|-----------|
|heartbeatInterval|`30s`|The number of milliseconds to wait between heartbeat signals on the WebSocket connection|
|initialConnectAttempts|`5`|The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing|
|path|`<nil>`|The WebSocket sever URL to which FireFly should connect|
|readBufferSize|`16Kb`|The size in bytes of the read buffer for the WebSocket connection|
|writeBufferSize|`16Kb`|The size in bytes of the write buffer for the WebSocket connection|

## blockchainevent.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## broadcast.batch

|Key|Default Value|Description|
|---|-------------|-----------|
|agentTimeout|`<nil>`|TBD|
|payloadLimit|`<nil>`|TBD|
|size|`<nil>`|TBD|
|timeout|`<nil>`|TBD|

## cors

|Key|Default Value|Description|
|---|-------------|-----------|
|credentials|`<nil>`|TBD|
|debug|`<nil>`|TBD|
|enabled|`<nil>`|TBD|
|headers|`<nil>`|TBD|
|maxAge|`<nil>`|TBD|
|methods|`<nil>`|TBD|
|origins|`<nil>`|TBD|

## database

|Key|Default Value|Description|
|---|-------------|-----------|
|maxChartRows|`<nil>`|TBD|
|type|`<nil>`|TBD|

## database.postgres

|Key|Default Value|Description|
|---|-------------|-----------|
|maxConnIdleTime|`1m`|TBD|
|maxConnLifetime|`<nil>`|TBD|
|maxConns|`50`|TBD|
|maxIdleConns|`<nil>`|TBD|
|url|`<nil>`|TBD|

## database.postgres.migrations

|Key|Default Value|Description|
|---|-------------|-----------|
|auto|`false`|TBD|
|directory|`./db/migrations/postgres`|TBD|

## database.sqlite3

|Key|Default Value|Description|
|---|-------------|-----------|
|maxConnIdleTime|`1m`|TBD|
|maxConnLifetime|`<nil>`|TBD|
|maxConns|`1`|TBD|
|maxIdleConns|`<nil>`|TBD|
|url|`<nil>`|TBD|

## database.sqlite3.migrations

|Key|Default Value|Description|
|---|-------------|-----------|
|auto|`false`|TBD|
|directory|`./db/migrations/sqlite`|TBD|

## dataexchange

|Key|Default Value|Description|
|---|-------------|-----------|
|type|`<nil>`|TBD|

## dataexchange.ffdx

|Key|Default Value|Description|
|---|-------------|-----------|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|initEnabled|`false`|TBD|
|manifestEnabled|`false`|TBD|
|maxIdleConns|`100`|TBD|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|TBD|

## dataexchange.ffdx.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## dataexchange.ffdx.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|TBD|

## dataexchange.ffdx.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## dataexchange.ffdx.ws

|Key|Default Value|Description|
|---|-------------|-----------|
|heartbeatInterval|`30s`|The number of milliseconds to wait between heartbeat signals on the WebSocket connection|
|initialConnectAttempts|`5`|The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing|
|path|`<nil>`|The WebSocket sever URL to which FireFly should connect|
|readBufferSize|`16Kb`|The size in bytes of the read buffer for the WebSocket connection|
|writeBufferSize|`16Kb`|The size in bytes of the write buffer for the WebSocket connection|

## debug

|Key|Default Value|Description|
|---|-------------|-----------|
|port|`<nil>`|TBD|

## download.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initialDelay|`<nil>`|The initial retry delay|
|maxAttempts|`<nil>`|The maximum number of times to retry|
|maxDelay|`<nil>`|The maximum retry delay|

## download.worker

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`<nil>`|TBD|
|queueLength|`<nil>`|TBD|

## event.aggregator

|Key|Default Value|Description|
|---|-------------|-----------|
|batchSize|`<nil>`|TBD|
|batchTimeout|`<nil>`|TBD|
|firstEvent|`<nil>`|TBD|
|opCorrelationRetries|`<nil>`|TBD|
|pollTimeout|`<nil>`|TBD|
|rewindQueueLength|`<nil>`|TBD|
|rewindTimeout|`<nil>`|TBD|

## event.aggregator.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## event.dbevents

|Key|Default Value|Description|
|---|-------------|-----------|
|bufferSize|`<nil>`|TBD|

## event.dispatcher

|Key|Default Value|Description|
|---|-------------|-----------|
|batchTimeout|`<nil>`|TBD|
|bufferLength|`<nil>`|TBD|
|pollTimeout|`<nil>`|TBD|

## event.dispatcher.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## event.listenerTopic.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## event.transports

|Key|Default Value|Description|
|---|-------------|-----------|
|default|`<nil>`|TBD|
|enabled|`<nil>`|TBD|

## group.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## http

|Key|Default Value|Description|
|---|-------------|-----------|
|address|`127.0.0.1`|The IP address on which the HTTP API should listen|
|port|`5000`|The port on which the HTTP API should listen|
|publicURL|`<nil>`|The fully qualified public URL for the API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.|
|readTimeout|`15s`|The maximum time to wait in seconds when reading from an HTTP connection|
|writeTimeout|`15s`|The maximum time to wait in seconds when writing to an HTTP connection|

## http.tls

|Key|Default Value|Description|
|---|-------------|-----------|
|caFile|`<nil>`|The path to the CA file for the admin API|
|certFile|`<nil>`|The path to the certificate file for the admin API|
|clientAuth|`<nil>`|Enables or disables client auth for the admin API|
|enabled|`false`|Enables or disables TLS on the admin API|
|keyFile|`<nil>`|The path to the private key file for the admin API|

## identity

|Key|Default Value|Description|
|---|-------------|-----------|
|type|`<nil>`|TBD|

## identity.manager.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|limit|`<nil>`|TBD|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## log

|Key|Default Value|Description|
|---|-------------|-----------|
|compress|`<nil>`|TBD|
|filename|`<nil>`|TBD|
|filesize|`<nil>`|TBD|
|forceColor|`<nil>`|TBD|
|level|`<nil>`|TBD|
|maxAge|`<nil>`|TBD|
|maxBackups|`<nil>`|TBD|
|noColor|`<nil>`|TBD|
|timeFormat|`<nil>`|TBD|
|utc|`<nil>`|TBD|

## message.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## message.writer

|Key|Default Value|Description|
|---|-------------|-----------|
|batchMaxInserts|`<nil>`|TBD|
|batchTimeout|`<nil>`|TBD|
|count|`<nil>`|TBD|

## metrics

|Key|Default Value|Description|
|---|-------------|-----------|
|address|`127.0.0.1`|The IP address on which the metrics HTTP API should listen|
|enabled|`true`|Enables the metrics API|
|path|`/metrics`|TBD|
|port|`6000`|The port on which the metrics HTTP API should listen|
|publicURL|`<nil>`|The fully qualified public URL for the metrics API. This is used for building URLs in HTTP responses and in OpenAPI Spec generation.|
|readTimeout|`15s`|The maximum time to wait in seconds when reading from an HTTP connection|
|writeTimeout|`15s`|The maximum time to wait in seconds when writing to an HTTP connection|

## metrics.tls

|Key|Default Value|Description|
|---|-------------|-----------|
|caFile|`<nil>`|The path to the CA file for the admin API|
|certFile|`<nil>`|The path to the certificate file for the admin API|
|clientAuth|`<nil>`|Enables or disables client auth for the admin API|
|enabled|`false`|Enables or disables TLS on the admin API|
|keyFile|`<nil>`|The path to the private key file for the admin API|

## namespaces

|Key|Default Value|Description|
|---|-------------|-----------|
|default|`<nil>`|TBD|
|predefined|`<nil>`|TBD|

## node

|Key|Default Value|Description|
|---|-------------|-----------|
|description|`<nil>`|TBD|
|name|`<nil>`|TBD|

## opupdate.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initialDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## opupdate.worker

|Key|Default Value|Description|
|---|-------------|-----------|
|batchMaxInserts|`<nil>`|TBD|
|batchTimeout|`<nil>`|TBD|
|count|`<nil>`|TBD|
|queueLength|`<nil>`|TBD|

## orchestrator

|Key|Default Value|Description|
|---|-------------|-----------|
|startupAttempts|`<nil>`|TBD|

## org

|Key|Default Value|Description|
|---|-------------|-----------|
|description|`<nil>`|TBD|
|identity|`<nil>`|TBD|
|key|`<nil>`|TBD|
|name|`<nil>`|TBD|

## privatemessaging

|Key|Default Value|Description|
|---|-------------|-----------|
|opCorrelationRetries|`<nil>`|TBD|

## privatemessaging.batch

|Key|Default Value|Description|
|---|-------------|-----------|
|agentTimeout|`<nil>`|TBD|
|payloadLimit|`<nil>`|TBD|
|size|`<nil>`|TBD|
|timeout|`<nil>`|TBD|

## privatemessaging.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## publicstorage

|Key|Default Value|Description|
|---|-------------|-----------|
|type|`<nil>`|TBD|

## publicstorage.ipfs.api

|Key|Default Value|Description|
|---|-------------|-----------|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|maxIdleConns|`100`|TBD|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|TBD|

## publicstorage.ipfs.api.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## publicstorage.ipfs.api.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|TBD|

## publicstorage.ipfs.api.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## publicstorage.ipfs.gateway

|Key|Default Value|Description|
|---|-------------|-----------|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|maxIdleConns|`100`|TBD|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|TBD|

## publicstorage.ipfs.gateway.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## publicstorage.ipfs.gateway.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|TBD|

## publicstorage.ipfs.gateway.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## sharedstorage

|Key|Default Value|Description|
|---|-------------|-----------|
|type|`<nil>`|TBD|

## sharedstorage.ipfs.api

|Key|Default Value|Description|
|---|-------------|-----------|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|maxIdleConns|`100`|TBD|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|TBD|

## sharedstorage.ipfs.api.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## sharedstorage.ipfs.api.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|TBD|

## sharedstorage.ipfs.api.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## sharedstorage.ipfs.gateway

|Key|Default Value|Description|
|---|-------------|-----------|
|connectionTimeout|`30s`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`1s`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`475ms`|TBD|
|maxIdleConns|`100`|TBD|
|requestTimeout|`30s`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`10s`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|TBD|

## sharedstorage.ipfs.gateway.auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## sharedstorage.ipfs.gateway.proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|TBD|

## sharedstorage.ipfs.gateway.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`5`|The maximum number of times to retry|
|enabled|`false`|Enables retries|
|initWaitTime|`250ms`|The initial retry delay|
|maxWaitTime|`30s`|The maximum retry delay|

## subscription

|Key|Default Value|Description|
|---|-------------|-----------|
|max|`<nil>`|TBD|

## subscription.defaults

|Key|Default Value|Description|
|---|-------------|-----------|
|batchSize|`<nil>`|TBD|

## subscription.retry

|Key|Default Value|Description|
|---|-------------|-----------|
|factor|`<nil>`|The retry backoff factor|
|initDelay|`<nil>`|The initial retry delay|
|maxDelay|`<nil>`|The maximum retry delay|

## tokens[]

|Key|Default Value|Description|
|---|-------------|-----------|
|connectionTimeout|`<nil>`|The maximum amount of time, in milliseconds that a connection is allowed to remain with no data transmitted|
|connector|`<nil>`|TBD|
|customClient|`<nil>`|TBD|
|expectContinueTimeout|`<nil>`|TBD|
|headers|`<nil>`|TBD|
|idleTimeout|`<nil>`|TBD|
|maxIdleConns|`<nil>`|TBD|
|name|`<nil>`|TBD|
|plugin|`<nil>`|TBD|
|requestTimeout|`<nil>`|The maximum amount of time, in milliseconds that a request is allowed to remain open|
|tlsHandshakeTimeout|`<nil>`|The maximum amount of time, in milliseconds to wait for a successful TLS handshake|
|url|`<nil>`|TBD|

## tokens[].auth

|Key|Default Value|Description|
|---|-------------|-----------|
|password|`<nil>`|Password|
|username|`<nil>`|Username|

## tokens[].proxy

|Key|Default Value|Description|
|---|-------------|-----------|
|url|`<nil>`|TBD|

## tokens[].retry

|Key|Default Value|Description|
|---|-------------|-----------|
|count|`<nil>`|The maximum number of times to retry|
|enabled|`<nil>`|Enables retries|
|initWaitTime|`<nil>`|The initial retry delay|
|maxWaitTime|`<nil>`|The maximum retry delay|

## tokens[].ws

|Key|Default Value|Description|
|---|-------------|-----------|
|heartbeatInterval|`<nil>`|The number of milliseconds to wait between heartbeat signals on the WebSocket connection|
|initialConnectAttempts|`<nil>`|The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing|
|path|`<nil>`|The WebSocket sever URL to which FireFly should connect|
|readBufferSize|`<nil>`|The size in bytes of the read buffer for the WebSocket connection|
|writeBufferSize|`<nil>`|The size in bytes of the write buffer for the WebSocket connection|

## transaction.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|

## ui

|Key|Default Value|Description|
|---|-------------|-----------|
|enabled|`<nil>`|Enables the web user interface|
|path|`<nil>`|The file system path which contains the static HTML, CSS, and JavaScript files for the user interface|

## validator.cache

|Key|Default Value|Description|
|---|-------------|-----------|
|size|`<nil>`|The size of the cache|
|ttl|`<nil>`|The time to live (TTL) for the cache|