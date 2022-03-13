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

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// The following keys can be access from the root configuration.
// Plugins are responsible for defining their own keys using the Config interface
var (
	// APIDefaultFilterLimit is the default limit that will be applied to filtered queries on the API
	APIDefaultFilterLimit = rootKey("api.defaultFilterLimit")
	// APIMaxFilterLimit is the maximum limit that can be specified by an API call
	APIMaxFilterLimit = rootKey("api.maxFilterLimit")
	// APIMaxFilterSkip is the maximum skip value that can be specified on the API
	APIMaxFilterSkip = rootKey("api.maxFilterLimit")
	// APIRequestTimeout is the server side timeout for API calls (context timeout), to avoid the server continuing processing when the client gives up
	APIRequestTimeout = rootKey("api.requestTimeout")
	// APIRequestMaxTimeout is the maximum timeout an application can set using a Request-Timeout header
	APIRequestMaxTimeout = rootKey("api.requestMaxTimeout")
	// APIShutdownTimeout is the amount of time to wait for any in-flight requests to finish before killing the HTTP server
	APIShutdownTimeout = rootKey("api.shutdownTimeout")
	// BatchManagerReadPageSize is the size of each page of messages read from the database into memory when assembling batches
	BatchManagerReadPageSize = rootKey("batch.manager.readPageSize")
	// BatchManagerReadPollTimeout is how long without any notifications of new messages to wait, before doing a page query
	BatchManagerReadPollTimeout = rootKey("batch.manager.pollTimeout")
	// BatchManagerMinimumPollTime is the minimum duration between polls, to avoid continual polling at high throughput
	BatchManagerMinimumPollTime = rootKey("batch.manager.minimumPollTime")
	// BatchRetryFactor is the retry backoff factor for database operations performed by the batch manager
	BatchRetryFactor = rootKey("batch.retry.factor")
	// BatchRetryInitDelay is the retry initial delay for database operations
	BatchRetryInitDelay = rootKey("batch.retry.initDelay")
	// BatchRetryMaxDelay is the maximum delay between retry attempts
	BatchRetryMaxDelay = rootKey("batch.retry.maxDelay")
	// BlockchainType is the name of the blockchain interface plugin being used by this firefly node
	BlockchainType = rootKey("blockchain.type")
	// BroadcastBatchAgentTimeout how long to keep around a batching agent for a sending identity before disposal
	BroadcastBatchAgentTimeout = rootKey("broadcast.batch.agentTimeout")
	// BroadcastBatchSize is the maximum number of messages that can be packed into a batch
	BroadcastBatchSize = rootKey("broadcast.batch.size")
	// BroadcastBatchPayloadLimit is the maximum payload size of a batch for broadcast messages
	BroadcastBatchPayloadLimit = rootKey("broadcast.batch.payloadLimit")
	// BroadcastBatchTimeout is the timeout to wait for a batch to fill, before sending
	BroadcastBatchTimeout = rootKey("broadcast.batch.timeout")
	// PrivateMessagingBatchAgentTimeout how long to keep around a batching agent for a sending identity before disposal
	PrivateMessagingBatchAgentTimeout = rootKey("privatemessaging.batch.agentTimeout")
	// PrivateMessagingBatchSize is the maximum size of a batch for broadcast messages
	PrivateMessagingBatchSize = rootKey("privatemessaging.batch.size")
	// PrivateMessagingBatchPayloadLimit is the maximum payload size of a private message data exchange payload
	PrivateMessagingBatchPayloadLimit = rootKey("privatemessaging.batch.payloadLimit")
	// PrivateMessagingBatchTimeout is the timeout to wait for a batch to fill, before sending
	PrivateMessagingBatchTimeout = rootKey("privatemessaging.batch.timeout")
	// PrivateMessagingOpCorrelationRetries how many times to correlate an event for an operation (such as tx submission) back to an operation.
	// Needed because the operation update might come back before we are finished persisting the ID of the request
	PrivateMessagingOpCorrelationRetries = rootKey("privatemessaging.opCorrelationRetries")
	// PrivateMessagingRetryFactor the backoff factor to use for retry of database operations
	PrivateMessagingRetryFactor = rootKey("privatemessaging.retry.factor")
	// PrivateMessagingRetryInitDelay the initial delay to use for retry of data base operations
	PrivateMessagingRetryInitDelay = rootKey("privatemessaging.retry.initDelay")
	// PrivateMessagingRetryMaxDelay the maximum delay to use for retry of data base operations
	PrivateMessagingRetryMaxDelay = rootKey("privatemessaging.retry.maxDelay")
	// CorsAllowCredentials CORS setting to control whether a browser allows credentials to be sent to this API
	CorsAllowCredentials = rootKey("cors.credentials")
	// CorsAllowedHeaders CORS setting to control the allowed headers
	CorsAllowedHeaders = rootKey("cors.headers")
	// CorsAllowedMethods CORS setting to control the allowed methods
	CorsAllowedMethods = rootKey("cors.methods")
	// CorsAllowedOrigins CORS setting to control the allowed origins
	CorsAllowedOrigins = rootKey("cors.origins")
	// CorsDebug is whether debug is enabled for the CORS implementation
	CorsDebug = rootKey("cors.debug")
	// CorsEnabled is whether cors is enabled
	CorsEnabled = rootKey("cors.enabled")
	// CorsMaxAge is the maximum age a browser should rely on CORS checks
	CorsMaxAge = rootKey("cors.maxAge")
	// DataexchangeType is the name of the data exchange plugin being used by this firefly node
	DataexchangeType = rootKey("dataexchange.type")
	// DatabaseType the type of the database interface plugin to use
	DatabaseType = rootKey("database.type")
	// TokensList is the root key containing a list of supported token connectors
	TokensList = rootKey("tokens")
	// DebugPort a HTTP port on which to enable the go debugger
	DebugPort = rootKey("debug.port")
	// EventTransportsDefault the default event transport for new subscriptions
	EventTransportsDefault = rootKey("event.transports.default")
	// EventTransportsEnabled which event interface plugins are enabled
	EventTransportsEnabled = rootKey("event.transports.enabled")
	// EventAggregatorFirstEvent the first event the aggregator should process, if no previous offest is stored in the DB
	EventAggregatorFirstEvent = rootKey("event.aggregator.firstEvent")
	// EventAggregatorBatchSize the maximum number of records to read from the DB before performing an aggregation run
	EventAggregatorBatchSize = rootKey("event.aggregator.batchSize")
	// EventAggregatorBatchTimeout how long to wait for new events to arrive before performing aggregation on a page of events
	EventAggregatorBatchTimeout = rootKey("event.aggregator.batchTimeout")
	// EventAggregatorOpCorrelationRetries how many times to correlate an event for an operation (such as tx submission) back to an operation.
	// Needed because the operation update might come back before we are finished persisting the ID of the request
	EventAggregatorOpCorrelationRetries = rootKey("event.aggregator.opCorrelationRetries")
	// EventAggregatorPollTimeout the time to wait without a notification of new events, before trying a select on the table
	EventAggregatorPollTimeout = rootKey("event.aggregator.pollTimeout")
	// EventAggregatorRetryFactor the backoff factor to use for retry of database operations
	EventAggregatorRetryFactor = rootKey("event.aggregator.retry.factor")
	// EventAggregatorRetryInitDelay the initial delay to use for retry of data base operations
	EventAggregatorRetryInitDelay = rootKey("event.aggregator.retry.initDelay")
	// EventAggregatorRetryMaxDelay the maximum delay to use for retry of data base operations
	EventAggregatorRetryMaxDelay = rootKey("event.aggregator.retry.maxDelay")
	// EventDispatcherPollTimeout the time to wait without a notification of new events, before trying a select on the table
	EventDispatcherPollTimeout = rootKey("event.dispatcher.pollTimeout")
	// EventDispatcherBufferLength the number of events + attachments an individual dispatcher should hold in memory ready for delivery to the subscription
	EventDispatcherBufferLength = rootKey("event.dispatcher.bufferLength")
	// EventDispatcherBatchTimeout a short time to wait for new events to arrive before re-polling for new events
	EventDispatcherBatchTimeout = rootKey("event.dispatcher.batchTimeout")
	// EventDispatcherRetryFactor the backoff factor to use for retry of database operations
	EventDispatcherRetryFactor = rootKey("event.dispatcher.retry.factor")
	// EventDispatcherRetryInitDelay he initial delay to use for retry of data base operations
	EventDispatcherRetryInitDelay = rootKey("event.dispatcher.retry.initDelay")
	// EventDispatcherRetryMaxDelay he maximum delay to use for retry of data base operations
	EventDispatcherRetryMaxDelay = rootKey("event.dispatcher.retry.maxDelay")
	// EventDBEventsBufferSize the size of the buffer of change events
	EventDBEventsBufferSize = rootKey("event.dbevents.bufferSize")
	// GroupCacheSize cache size for private group addresses
	GroupCacheSize = rootKey("group.cache.size")
	// GroupCacheTTL cache time-to-live for private group addresses
	GroupCacheTTL = rootKey("group.cache.ttl")
	// AdminEnabled determines whether the admin interface will be enabled or not
	AdminEnabled = rootKey("admin.enabled")
	// AdminPreinit waits for at least one ConfigREcord to be posted to the server before it starts (the database must be available on startup)
	AdminPreinit = rootKey("admin.preinit")
	// IdentityType the type of the identity plugin in use
	IdentityType = rootKey("identity.type")
	// IdentityManagerCacheTTL the identity manager cache time to live
	IdentityManagerCacheTTL = rootKey("identity.manager.cache.ttl")
	// IdentityManagerCacheLimit the identity manager cache limit in count of items
	IdentityManagerCacheLimit = rootKey("identity.manager.cache.limit")
	// Lang is the language to use for translation
	Lang = rootKey("lang")
	// LogForceColor forces color to be enabled, even if we do not detect a TTY
	LogForceColor = rootKey("log.forceColor")
	// LogLevel is the logging level
	LogLevel = rootKey("log.level")
	// LogNoColor forces color to be disabled, even if we detect a TTY
	LogNoColor = rootKey("log.noColor")
	// LogTimeFormat is a string format for timestamps
	LogTimeFormat = rootKey("log.timeFormat")
	// LogUTC sets log timestamps to the UTC timezone
	LogUTC = rootKey("log.utc")
	// LogFilename sets logging to file
	LogFilename = rootKey("log.filename")
	// LogFilesize sets the size to roll logs at
	LogFilesize = rootKey("log.filesize")
	// LogMaxBackups sets the maximum number of old files to keep
	LogMaxBackups = rootKey("log.maxBackups")
	// LogMaxAge sets the maximum age at which to roll
	LogMaxAge = rootKey("log.maxAge")
	// LogCompress sets whether to compress backups
	LogCompress = rootKey("log.compress")
	// MessageCacheSize
	MessageCacheSize = rootKey("message.cache.size")
	// MessageCacheTTL
	MessageCacheTTL = rootKey("message.cache.ttl")
	// MessageWriterCount
	MessageWriterCount = rootKey("message.writer.count")
	// MessageWriterBatchTimeout
	MessageWriterBatchTimeout = rootKey("message.writer.batchTimeout")
	// MessageWriterBatchMaxInserts
	MessageWriterBatchMaxInserts = rootKey("message.writer.batchMaxInserts")
	// MetricsEnabled determines whether metrics will be instrumented and if the metrics server will be enabled or not
	MetricsEnabled = rootKey("metrics.enabled")
	// MetricsPath determines what path to serve the Prometheus metrics from
	MetricsPath = rootKey("metrics.path")
	// NamespacesDefault is the default namespace - must be in the predefines list
	NamespacesDefault = rootKey("namespaces.default")
	// NamespacesPredefined is a list of namespaces to ensure exists, without requiring a broadcast from the network
	NamespacesPredefined = rootKey("namespaces.predefined")
	// NodeName is a description for the node
	NodeName = rootKey("node.name")
	// NodeDescription is a description for the node
	NodeDescription = rootKey("node.description")
	// OrgName is the short name o the org
	OrgName = rootKey("org.name")
	// OrgIdentityDeprecated deprecated synonym to org.key
	OrgIdentityDeprecated = rootKey("org.identity")
	// OrgKey is the signing identity allocated to the organization (can be the same as the nodes)
	OrgKey = rootKey("org.key")
	// OrgDescription is a description for the org
	OrgDescription = rootKey("org.description")
	// OrchestratorStartupAttempts is how many time to attempt to connect to core infrastructure on startup
	OrchestratorStartupAttempts = rootKey("orchestrator.startupAttempts")
	// SharedStorageType specifies which shared storage interface plugin to use
	SharedStorageType = rootKey("sharedstorage.type")
	// PublicStorageType specifies which shared storage interface plugin to use - deprecated in favor of SharedStorageType
	PublicStorageType = rootKey("publicstorage.type")
	// SubscriptionDefaultsReadAhead default read ahead to enable for subscriptions that do not explicitly configure readahead
	SubscriptionDefaultsReadAhead = rootKey("subscription.defaults.batchSize")
	// SubscriptionMax maximum number of pre-defined subscriptions that can exist (note for high fan-out consider connecting a dedicated pub/sub broker to the dispatcher)
	SubscriptionMax = rootKey("subscription.max")
	// SubscriptionsRetryInitialDelay is the initial retry delay
	SubscriptionsRetryInitialDelay = rootKey("subscription.retry.initDelay")
	// SubscriptionsRetryMaxDelay is the initial retry delay
	SubscriptionsRetryMaxDelay = rootKey("subscription.retry.maxDelay")
	// SubscriptionsRetryFactor the backoff factor to use for retry of database operations
	SubscriptionsRetryFactor = rootKey("subscription.retry.factor")
	// TransactionCacheSize
	TransactionCacheSize = rootKey("transaction.cache.size")
	// TransactionCacheTTL
	TransactionCacheTTL = rootKey("transaction.cache.ttl")
	// AssetManagerKeyNormalization mechanism to normalize keys before using them. Valid options: "blockchain_plugin" - use blockchain plugin (default), "none" - do not attempt normalization
	AssetManagerKeyNormalization = rootKey("asset.manager.keyNormalization")
	// UIEnabled set to false to disable the UI (default is true, so UI will be enabled if ui.path is valid)
	UIEnabled = rootKey("ui.enabled")
	// UIPath the path on which to serve the UI
	UIPath = rootKey("ui.path")
	// ValidatorCacheSize
	ValidatorCacheSize = rootKey("validator.cache.size")
	// ValidatorCacheTTL
	ValidatorCacheTTL = rootKey("validator.cache.ttl")
)

type KeySet interface {
	AddKnownKey(key string, defValue ...interface{})
}

// Prefix represents the global configuration, at a nested point in
// the config hierarchy. This allows plugins to define their
// Note that all values are GLOBAL so this cannot be used for per-instance
// customization. Rather for global initialization of plugins.
type Prefix interface {
	KeySet
	SetDefault(key string, defValue interface{})
	SubPrefix(suffix string) Prefix
	Array() PrefixArray
	Set(key string, value interface{})
	Resolve(key string) string

	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetInt64(key string) int64
	GetByteSize(key string) int64
	GetUint(key string) uint
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	GetObject(key string) fftypes.JSONObject
	GetObjectArray(key string) fftypes.JSONObjectArray
	Get(key string) interface{}
}

// PrefixArray represents an array of options at a particular layer in the config.
// This allows specifying the schema of keys that exist for every entry, and the defaults,
// as well as querying how many entries exist and generating a prefix for each entry
// (so that you can iterate).
type PrefixArray interface {
	KeySet
	ArraySize() int
	ArrayEntry(i int) Prefix
}

// RootKey key are the known configuration keys
type RootKey string

func Reset() {
	keysMutex.Lock() // must only call viper directly here (as we already hold the lock)
	defer keysMutex.Unlock()

	viper.Reset()

	// Set defaults
	viper.SetDefault(string(APIDefaultFilterLimit), 25)
	viper.SetDefault(string(APIRequestTimeout), "120s")
	viper.SetDefault(string(APIRequestMaxTimeout), "10m")
	viper.SetDefault(string(APIMaxFilterLimit), 250)
	viper.SetDefault(string(APIMaxFilterSkip), 1000) // protects database (skip+limit pagination is not for bulk operations)
	viper.SetDefault(string(APIRequestTimeout), "120s")
	viper.SetDefault(string(APIShutdownTimeout), "10s")
	viper.SetDefault(string(AssetManagerKeyNormalization), "blockchain_plugin")
	viper.SetDefault(string(BatchManagerReadPageSize), 100)
	viper.SetDefault(string(BatchManagerReadPollTimeout), "30s")
	viper.SetDefault(string(BatchManagerMinimumPollTime), "50ms")
	viper.SetDefault(string(BatchRetryFactor), 2.0)
	viper.SetDefault(string(BatchRetryFactor), 2.0)
	viper.SetDefault(string(BatchRetryInitDelay), "250ms")
	viper.SetDefault(string(BatchRetryInitDelay), "250ms")
	viper.SetDefault(string(BatchRetryMaxDelay), "30s")
	viper.SetDefault(string(BatchRetryMaxDelay), "30s")
	viper.SetDefault(string(BroadcastBatchAgentTimeout), "2m")
	viper.SetDefault(string(BroadcastBatchSize), 200)
	viper.SetDefault(string(BroadcastBatchPayloadLimit), "800Kb")
	viper.SetDefault(string(BroadcastBatchTimeout), "1s")
	viper.SetDefault(string(CorsAllowCredentials), true)
	viper.SetDefault(string(CorsAllowedHeaders), []string{"*"})
	viper.SetDefault(string(CorsAllowedMethods), []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete})
	viper.SetDefault(string(CorsAllowedOrigins), []string{"*"})
	viper.SetDefault(string(CorsEnabled), true)
	viper.SetDefault(string(CorsMaxAge), 600)
	viper.SetDefault(string(DataexchangeType), "https")
	viper.SetDefault(string(DebugPort), -1)
	viper.SetDefault(string(EventAggregatorFirstEvent), fftypes.SubOptsFirstEventOldest)
	viper.SetDefault(string(EventAggregatorBatchSize), 50)
	viper.SetDefault(string(EventAggregatorBatchTimeout), "250ms")
	viper.SetDefault(string(EventAggregatorPollTimeout), "30s")
	viper.SetDefault(string(EventAggregatorRetryFactor), 2.0)
	viper.SetDefault(string(EventAggregatorRetryInitDelay), "100ms")
	viper.SetDefault(string(EventAggregatorRetryMaxDelay), "30s")
	viper.SetDefault(string(EventAggregatorOpCorrelationRetries), 3)
	viper.SetDefault(string(EventDBEventsBufferSize), 100)
	viper.SetDefault(string(EventDispatcherBufferLength), 5)
	viper.SetDefault(string(EventDispatcherBatchTimeout), "250ms")
	viper.SetDefault(string(EventDispatcherPollTimeout), "30s")
	viper.SetDefault(string(EventTransportsEnabled), []string{"websockets", "webhooks"})
	viper.SetDefault(string(EventTransportsDefault), "websockets")
	viper.SetDefault(string(GroupCacheSize), "1Mb")
	viper.SetDefault(string(GroupCacheTTL), "1h")
	viper.SetDefault(string(AdminEnabled), false)
	viper.SetDefault(string(IdentityType), "onchain")
	viper.SetDefault(string(Lang), "en")
	viper.SetDefault(string(LogLevel), "info")
	viper.SetDefault(string(LogTimeFormat), "2006-01-02T15:04:05.000Z07:00")
	viper.SetDefault(string(LogUTC), false)
	viper.SetDefault(string(LogFilesize), "100m")
	viper.SetDefault(string(LogMaxAge), "24h")
	viper.SetDefault(string(LogMaxBackups), 2)
	viper.SetDefault(string(MessageCacheSize), "50Mb")
	viper.SetDefault(string(MessageCacheTTL), "5m")
	viper.SetDefault(string(MessageWriterBatchMaxInserts), 200)
	viper.SetDefault(string(MessageWriterBatchTimeout), "50ms")
	viper.SetDefault(string(MessageWriterCount), 5)
	viper.SetDefault(string(NamespacesDefault), "default")
	viper.SetDefault(string(NamespacesPredefined), fftypes.JSONObjectArray{{"name": "default", "description": "Default predefined namespace"}})
	viper.SetDefault(string(OrchestratorStartupAttempts), 5)
	viper.SetDefault(string(PrivateMessagingRetryFactor), 2.0)
	viper.SetDefault(string(PrivateMessagingRetryInitDelay), "100ms")
	viper.SetDefault(string(PrivateMessagingRetryMaxDelay), "30s")
	viper.SetDefault(string(PrivateMessagingOpCorrelationRetries), 3)
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

	i18n.SetLang(viper.GetString(string(Lang)))
}

// ReadConfig initializes the config
func ReadConfig(cfgFile string) error {
	keysMutex.Lock() // must only call viper directly here (as we already hold the lock)
	defer keysMutex.Unlock()

	// Set precedence order for reading config location
	viper.SetEnvPrefix("firefly")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")
	if cfgFile != "" {
		f, err := os.Open(cfgFile)
		if err == nil {
			defer f.Close()
			err = viper.ReadConfig(f)
		}
		return err
	}
	viper.SetConfigName("firefly.core")
	viper.AddConfigPath("/etc/firefly/")
	viper.AddConfigPath("$HOME/.firefly")
	viper.AddConfigPath(".")
	return viper.ReadInConfig()
}

func MergeConfig(configRecords []*fftypes.ConfigRecord) error {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	for _, c := range configRecords {
		s := viper.New()
		s.SetConfigType("json")
		var val interface{}
		if c.Value != nil {
			if err := json.Unmarshal([]byte(*c.Value), &val); err != nil {
				return err
			}
		}
		switch v := val.(type) {
		case map[string]interface{}:
			_ = s.ReadConfig(bytes.NewBuffer([]byte(*c.Value)))
			for _, k := range s.AllKeys() {
				value := s.Get(k)
				if reflect.TypeOf(value).Kind() == reflect.Slice {
					configSlice := value.([]interface{})
					for i := range configSlice {
						viper.Set(fmt.Sprintf("%s.%s.%d", c.Key, k, i), configSlice[i])
					}
				} else {
					viper.Set(fmt.Sprintf("%s.%s", c.Key, k), value)
				}
			}
		case []interface{}:
			_ = s.ReadConfig(bytes.NewBuffer([]byte(*c.Value)))
			for i := range v {
				viper.Set(fmt.Sprintf("%s.%d", c.Key, i), v[i])
			}
		default:
			viper.Set(c.Key, v)
		}
	}
	return nil
}

var knownKeys = map[string]bool{} // All keys go here, including those defined in sub prefixies
var keysMutex sync.Mutex
var root = &configPrefix{}

// ark adds a root key, used to define the keys that are used within the core
func rootKey(k string) RootKey {
	root.AddKnownKey(k)
	return RootKey(k)
}

// GetKnownKeys gets the known keys
func GetKnownKeys() []string {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	keys := make([]string, 0, len(knownKeys))
	for k := range knownKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// configPrefix is the main config structure passed to plugins, and used for root to wrap viper
type configPrefix struct {
	prefix string
}

// configPrefixArray is a point in the config that supports an array
type configPrefixArray struct {
	base     string
	defaults map[string][]interface{}
}

// NewPluginConfig creates a new plugin configuration object, at the specified prefix
func NewPluginConfig(prefix string) Prefix {
	if !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	return &configPrefix{
		prefix: prefix,
	}
}

func (c *configPrefix) prefixKey(k string) string {
	// Caller responsible for holding lock when calling
	key := c.prefix + k
	if !knownKeys[key] {
		panic(fmt.Sprintf("Undefined configuration key '%s'", key))
	}
	return key
}

func (c *configPrefix) SubPrefix(suffix string) Prefix {
	return &configPrefix{
		prefix: c.prefix + suffix + ".",
	}
}

func (c *configPrefix) Array() PrefixArray {
	return &configPrefixArray{
		base:     strings.TrimSuffix(c.prefix, "."),
		defaults: make(map[string][]interface{}),
	}
}

func (c *configPrefixArray) ArraySize() int {
	val := viper.Get(c.base)
	vt := reflect.TypeOf(val)
	if vt != nil && (vt.Kind() == reflect.Slice || vt.Kind() == reflect.Map) {
		return reflect.ValueOf(val).Len()
	}
	return 0
}

// ArrayEntry must only be called after the config has been loaded
func (c *configPrefixArray) ArrayEntry(i int) Prefix {
	cp := &configPrefix{
		prefix: c.base + fmt.Sprintf(".%d.", i),
	}
	for knownKey, defValue := range c.defaults {
		cp.AddKnownKey(knownKey, defValue...)
		// Sadly Viper can't handle defaults inside the array, when
		// a value is set. So here we check/set the defaults.
		if defValue != nil && cp.Get(knownKey) == nil {
			if len(defValue) == 1 {
				cp.Set(knownKey, defValue[0])
			} else if len(defValue) > 0 {
				cp.Set(knownKey, defValue)
			}
		}
	}
	return cp
}

func (c *configPrefixArray) AddKnownKey(k string, defValue ...interface{}) {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	// Put a simulated key in the known keys array, to pop into the help info.
	knownKeys[fmt.Sprintf("%s[].%s", c.base, k)] = true
	c.defaults[k] = defValue
}

func (c *configPrefix) AddKnownKey(k string, defValue ...interface{}) {
	key := c.prefix + k
	if len(defValue) == 1 {
		c.SetDefault(k, defValue[0])
	} else if len(defValue) > 0 {
		c.SetDefault(k, defValue)
	}
	keysMutex.Lock()
	defer keysMutex.Unlock()
	knownKeys[key] = true
}

func (c *configPrefix) SetDefault(k string, defValue interface{}) {
	key := c.prefix + k
	viper.SetDefault(key, defValue)
}

func GetConfig() fftypes.JSONObject {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	conf := fftypes.JSONObject{}
	_ = viper.Unmarshal(&conf)
	return conf
}

// GetString gets a configuration string
func GetString(key RootKey) string {
	return root.GetString(string(key))
}
func (c *configPrefix) GetString(key string) string {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetString(c.prefixKey(key))
}

// GetStringSlice gets a configuration string array
func GetStringSlice(key RootKey) []string {
	return root.GetStringSlice(string(key))
}
func (c *configPrefix) GetStringSlice(key string) []string {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetStringSlice(c.prefixKey(key))
}

// GetBool gets a configuration bool
func GetBool(key RootKey) bool {
	return root.GetBool(string(key))
}
func (c *configPrefix) GetBool(key string) bool {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetBool(c.prefixKey(key))
}

// GetDuration gets a configuration time duration with consistent semantics
func GetDuration(key RootKey) time.Duration {
	return root.GetDuration(string(key))
}
func (c *configPrefix) GetDuration(key string) time.Duration {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return fftypes.ParseToDuration(viper.GetString(c.prefixKey(key)))
}

// GetByteSize get a size in bytes
func GetByteSize(key RootKey) int64 {
	return root.GetByteSize(string(key))
}
func (c *configPrefix) GetByteSize(key string) int64 {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return fftypes.ParseToByteSize(viper.GetString(c.prefixKey(key)))
}

// GetUint gets a configuration uint
func GetUint(key RootKey) uint {
	return root.GetUint(string(key))
}
func (c *configPrefix) GetUint(key string) uint {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetUint(c.prefixKey(key))
}

// GetInt gets a configuration uint
func GetInt(key RootKey) int {
	return root.GetInt(string(key))
}
func (c *configPrefix) GetInt(key string) int {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetInt(c.prefixKey(key))
}

// GetInt64 gets a configuration uint
func GetInt64(key RootKey) int64 {
	return root.GetInt64(string(key))
}
func (c *configPrefix) GetInt64(key string) int64 {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetInt64(c.prefixKey(key))
}

// GetFloat64 gets a configuration uint
func GetFloat64(key RootKey) float64 {
	return root.GetFloat64(string(key))
}
func (c *configPrefix) GetFloat64(key string) float64 {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.GetFloat64(c.prefixKey(key))
}

// GetObject gets a configuration map
func GetObject(key RootKey) fftypes.JSONObject {
	return root.GetObject(string(key))
}
func (c *configPrefix) GetObject(key string) fftypes.JSONObject {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return fftypes.JSONObject(viper.GetStringMap(c.prefixKey(key)))
}

// GetObjectArray gets an array of configuration maps
func GetObjectArray(key RootKey) fftypes.JSONObjectArray {
	return root.GetObjectArray(string(key))
}
func (c *configPrefix) GetObjectArray(key string) fftypes.JSONObjectArray {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	v, _ := fftypes.ToJSONObjectArray(viper.Get(c.prefixKey(key)))
	return v
}

// Get gets a configuration in raw form
func Get(key RootKey) interface{} {
	return root.Get(string(key))
}
func (c *configPrefix) Get(key string) interface{} {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return viper.Get(c.prefixKey(key))
}

// Set allows runtime setting of config (used in unit tests)
func Set(key RootKey, value interface{}) {
	root.Set(string(key), value)
}
func (c *configPrefix) Set(key string, value interface{}) {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	viper.Set(c.prefixKey(key), value)
}

// Resolve gives the fully qualified path of a key
func (c *configPrefix) Resolve(key string) string {
	keysMutex.Lock()
	defer keysMutex.Unlock()

	return c.prefixKey(key)
}

// SetupLogging initializes logging
func SetupLogging(ctx context.Context) {
	log.SetFormatting(log.Formatting{
		DisableColor:    GetBool(LogNoColor),
		ForceColor:      GetBool(LogForceColor),
		TimestampFormat: GetString(LogTimeFormat),
		UTC:             GetBool(LogUTC),
	})
	logFilename := GetString(LogFilename)
	if logFilename != "" {
		lumberjack := &lumberjack.Logger{
			Filename:   logFilename,
			MaxSize:    int(math.Ceil(float64(GetByteSize(LogFilesize)) / 1024 / 1024)), /* round up in megabytes */
			MaxBackups: GetInt(LogMaxBackups),
			MaxAge:     int(math.Ceil(float64(GetDuration(LogMaxAge)) / float64(time.Hour) / 24)), /* round up in days */
			Compress:   GetBool(LogCompress),
		}
		logrus.SetOutput(lumberjack)
	}
	log.SetLevel(GetString(LogLevel))
	log.L(ctx).Debugf("Log level: %s", logrus.GetLevel())
}
