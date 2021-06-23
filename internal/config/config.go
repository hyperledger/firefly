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

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// The following keys can be access from the root configuration.
// Plugins are resonsible for defining their own keys using the Config interface
var (
	// APIDefaultFilterLimit is the default limit that will be applied to filtered queries on the API
	APIDefaultFilterLimit = rootKey("api.defaultFilterLimit")
	// APIMaxFilterLimit is the maximum limit that can be specified by an API call
	APIMaxFilterLimit = rootKey("api.maxFilterLimit")
	// APIMaxFilterSkip is the maximum skip value that can be specified on the API
	APIMaxFilterSkip = rootKey("api.maxFilterLimit")
	// APIRequestTimeout is the server side timeout for API calls (context timeout), to avoid the server continuing processing when the client gives up
	APIRequestTimeout = rootKey("api.requestTimeout")
	// BatchManagerReadPageSize is the size of each page of messages read from the database into memory when assembling batches
	BatchManagerReadPageSize = rootKey("batch.manager.readPageSize")
	// BatchManagerReadPollTimeout is how long without any notifications of new messages to wait, before doing a page query
	BatchManagerReadPollTimeout = rootKey("batch.manager.pollTimeout")
	// BatchRetryFactor is the retry backoff factor for database operations performed by the batch manager
	BatchRetryFactor = rootKey("batch.retry.factor")
	// BatchRetryInitDelay is the retry initial delay for database operations
	BatchRetryInitDelay = rootKey("batch.retry.initDelay")
	// BatchRetryMaxDelay is the maximum delay between retry attempts
	BatchRetryMaxDelay = rootKey("batch.retry.maxDelay")
	// BlockchainType is the name of the blockchain interface plugin being used by this firefly name
	BlockchainType = rootKey("blockchain.type")
	// BroadcastBatchAgentTimeout how long to keep around a batching agent for a sending identity before disposal
	BroadcastBatchAgentTimeout = rootKey("broadcast.batch.agentTimeout")
	// BroadcastBatchSize is the maximum size of a batch for broadcast messages
	BroadcastBatchSize = rootKey("broadcast.batch.size")
	// BroadcastBatchTimeout is the timeout to wait for a batch to fill, before sending
	BroadcastBatchTimeout = rootKey("broadcast.batch.timeout")
	// PrivateMessagingBatchAgentTimeout how long to keep around a batching agent for a sending identity before disposal
	PrivateMessagingBatchAgentTimeout = rootKey("privatemessaging.batch.agentTimeout")
	// PrivateMessagingBatchSize is the maximum size of a batch for broadcast messages
	PrivateMessagingBatchSize = rootKey("privatemessaging.batch.size")
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
	// DataexchangeType is the name of the data exchange plugin being used by this firefly name
	DataexchangeType = rootKey("dataexchange.type")
	// DatabaseType the type of the database interface plugin to use
	DatabaseType = rootKey("database.type")
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
	// GroupCacheSize cache size for private group addresses
	GroupCacheSize = rootKey("group.cache.size")
	// GroupCacheTTL cache time-to-live for private group addresses
	GroupCacheTTL = rootKey("group.cache.ttl")
	// AdminHTTPEnabled determines whether the admin interface will be enabled or not
	AdminEnabled = rootKey("admin.enabled")
	// AdminPreinit waits for at least one ConfigREcord to be posted to the server before it starts (the database must be available on startup)
	AdminPreinit = rootKey("admin.preinit")
	// IdentityType the type of the identity plugin in use
	IdentityType = rootKey("identity.type")
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
	// OrgIdentity is the signing identity allocated to the organization (can be the same as the nodes)
	OrgIdentity = rootKey("org.identity")
	// OrgDescription is a description for the org
	OrgDescription = rootKey("org.description")
	// OrchestratorStartupAttempts is how many time to attempt to connect to core infrastructure on startup
	OrchestratorStartupAttempts = rootKey("orchestrator.startupAttempts")
	// PublicStorageType specifies which public storage interface plugin to use
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
	SubscriptionsRetryFactor = rootKey("event.dispatcher.retry.factor")
	// UIEnabled set to false to disable the UI (default is true, so UI will be enabled if ui.path is valid)
	UIEnabled = rootKey("ui.enabled")
	// UIPath the path on which to serve the UI
	UIPath = rootKey("ui.path")
	// ValidatorCacheSize
	ValidatorCacheSize = rootKey("validator.cache.size")
	// ValidatorCacheTTL
	ValidatorCacheTTL = rootKey("validator.cache.ttl")
)

// Prefix represents the global configuration, at a nested point in
// the config hierarchy. This allows plugins to define their
// Note that all values are GLOBAL so this cannot be used for per-instance
// customization. Rather for global initialization of plugins.
type Prefix interface {
	AddKnownKey(key string, defValue ...interface{})
	SubPrefix(suffix string) Prefix
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

// RootKey key are the known configuration keys
type RootKey string

func Reset() {
	viper.Reset()

	// Set defaults
	viper.SetDefault(string(APIDefaultFilterLimit), 25)
	viper.SetDefault(string(APIRequestTimeout), "120s")
	viper.SetDefault(string(APIMaxFilterLimit), 250)
	viper.SetDefault(string(APIMaxFilterSkip), 1000) // protects database (skip+limit pagination is not for bulk operations)
	viper.SetDefault(string(APIRequestTimeout), "120s")
	viper.SetDefault(string(BatchManagerReadPageSize), 100)
	viper.SetDefault(string(BatchManagerReadPollTimeout), "30s")
	viper.SetDefault(string(BatchRetryFactor), 2.0)
	viper.SetDefault(string(BatchRetryFactor), 2.0)
	viper.SetDefault(string(BatchRetryInitDelay), "250ms")
	viper.SetDefault(string(BatchRetryInitDelay), "250ms")
	viper.SetDefault(string(BatchRetryMaxDelay), "30s")
	viper.SetDefault(string(BatchRetryMaxDelay), "30s")
	viper.SetDefault(string(BroadcastBatchAgentTimeout), "2m")
	viper.SetDefault(string(BroadcastBatchSize), 200)
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
	viper.SetDefault(string(SubscriptionDefaultsReadAhead), 0)
	viper.SetDefault(string(SubscriptionMax), 500)
	viper.SetDefault(string(SubscriptionsRetryInitialDelay), "250ms")
	viper.SetDefault(string(SubscriptionsRetryMaxDelay), "30s")
	viper.SetDefault(string(SubscriptionsRetryFactor), 2.0)
	viper.SetDefault(string(UIEnabled), true)
	viper.SetDefault(string(ValidatorCacheSize), "1Mb")
	viper.SetDefault(string(ValidatorCacheTTL), "1h")

	i18n.SetLang(GetString(Lang))
}

// ReadConfig initializes the config
func ReadConfig(cfgFile string) error {
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
	for _, c := range configRecords {
		s := viper.New()
		s.SetConfigType("json")
		var val interface{}
		if err := json.Unmarshal(c.Value, &val); err != nil {
			return err
		}
		switch val.(type) {
		case map[string]interface{}:
			_ = s.ReadConfig(bytes.NewBuffer(c.Value))
			for _, k := range s.AllKeys() {
				viper.Set(fmt.Sprintf("%s.%s", c.Key, k), s.Get(k))
			}
		default:
			viper.Set(c.Key, val)
		}
	}
	return nil
}

var root = &configPrefix{
	keys: map[string]bool{}, // All keys go here, including those defined in sub prefixies
}

// ark adds a root key, used to define the keys that are used within the core
func rootKey(k string) RootKey {
	root.AddKnownKey(k)
	return RootKey(k)
}

// GetKnownKeys gets the known keys
func GetKnownKeys() []string {
	var keys []string
	for k := range root.keys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// configPrefix is the main config structure passed to plugins, and used for root to wrap viper
type configPrefix struct {
	prefix string
	keys   map[string]bool
}

// NewPluginConfig creates a new plugin configuration object, at the specified prefix
func NewPluginConfig(prefix string) Prefix {
	if !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	return &configPrefix{
		prefix: prefix,
		keys:   root.keys,
	}
}

func (c *configPrefix) prefixKey(k string) string {
	key := c.prefix + k
	if !c.keys[key] {
		panic(fmt.Sprintf("Undefined configuration key '%s'", key))
	}
	return key
}

func (c *configPrefix) SubPrefix(suffix string) Prefix {
	return &configPrefix{
		prefix: c.prefix + suffix + ".",
		keys:   root.keys,
	}
}

func (c *configPrefix) AddKnownKey(k string, defValue ...interface{}) {
	key := c.prefix + k
	if len(defValue) == 1 {
		viper.SetDefault(key, defValue[0])
	} else if len(defValue) > 0 {
		viper.SetDefault(key, defValue)
	}
	c.keys[key] = true
}

func GetConfig() fftypes.JSONObject {
	conf := fftypes.JSONObject{}
	_ = viper.Unmarshal(&conf)
	return conf
}

// GetString gets a configuration string
func GetString(key RootKey) string {
	return root.GetString(string(key))
}
func (c *configPrefix) GetString(key string) string {
	return viper.GetString(c.prefixKey(key))
}

// GetStringSlice gets a configuration string array
func GetStringSlice(key RootKey) []string {
	return root.GetStringSlice(string(key))
}
func (c *configPrefix) GetStringSlice(key string) []string {
	return viper.GetStringSlice(c.prefixKey(key))
}

// GetBool gets a configuration bool
func GetBool(key RootKey) bool {
	return root.GetBool(string(key))
}
func (c *configPrefix) GetBool(key string) bool {
	return viper.GetBool(c.prefixKey(key))
}

// GetDuration gets a configuration time duration with consistent semantics
func GetDuration(key RootKey) time.Duration {
	return root.GetDuration(string(key))
}
func (c *configPrefix) GetDuration(key string) time.Duration {
	return fftypes.ParseToDuration(viper.GetString(c.prefixKey(key)))
}

// GetByteSize get a size in bytes
func GetByteSize(key RootKey) int64 {
	return root.GetByteSize(string(key))
}
func (c *configPrefix) GetByteSize(key string) int64 {
	return fftypes.ParseToByteSize(c.GetString(key))
}

// GetUint gets a configuration uint
func GetUint(key RootKey) uint {
	return root.GetUint(string(key))
}
func (c *configPrefix) GetUint(key string) uint {
	return viper.GetUint(c.prefixKey(key))
}

// GetInt gets a configuration uint
func GetInt(key RootKey) int {
	return root.GetInt(string(key))
}
func (c *configPrefix) GetInt(key string) int {
	return viper.GetInt(c.prefixKey(key))
}

// GetInt64 gets a configuration uint
func GetInt64(key RootKey) int64 {
	return root.GetInt64(string(key))
}
func (c *configPrefix) GetInt64(key string) int64 {
	return viper.GetInt64(c.prefixKey(key))
}

// GetFloat64 gets a configuration uint
func GetFloat64(key RootKey) float64 {
	return root.GetFloat64(string(key))
}
func (c *configPrefix) GetFloat64(key string) float64 {
	return viper.GetFloat64(c.prefixKey(key))
}

// GetObject gets a configuration map
func GetObject(key RootKey) fftypes.JSONObject {
	return root.GetObject(string(key))
}
func (c *configPrefix) GetObject(key string) fftypes.JSONObject {
	return fftypes.JSONObject(viper.GetStringMap(c.prefixKey(key)))
}

// GetObjectArray gets an array of configuration maps
func GetObjectArray(key RootKey) fftypes.JSONObjectArray {
	return root.GetObjectArray(string(key))
}
func (c *configPrefix) GetObjectArray(key string) fftypes.JSONObjectArray {
	v, _ := fftypes.ToJSONObjectArray(viper.Get(c.prefixKey(key)))
	return v
}

// Get gets a configuration in raw form
func Get(key RootKey) interface{} {
	return root.Get(string(key))
}
func (c *configPrefix) Get(key string) interface{} {
	return viper.Get(c.prefixKey(key))
}

// Set allows runtime setting of config (used in unit tests)
func Set(key RootKey, value interface{}) {
	root.Set(string(key), value)
}
func (c *configPrefix) Set(key string, value interface{}) {
	viper.Set(c.prefixKey(key), value)
}

// Resolve gives the fully qualified path of a key
func (c *configPrefix) Resolve(key string) string {
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
	log.SetLevel(GetString(LogLevel))
	log.L(ctx).Debugf("Log level: %s", logrus.GetLevel())
}
