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

package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// The following keys can be access from the root configuration.
// Plugins are resonsible for defining their own keys using the Config interface
var (
	APIDefaultFilterLimit            RootKey = ark("api.defaultFilterLimit")
	APIRequestTimeout                RootKey = ark("api.requestTimeout")
	BatchManagerReadPageSize         RootKey = ark("batch.manager.readPageSize")
	BatchManagerReadPollTimeout      RootKey = ark("batch.manager.pollTimeout")
	BatchRetryFactor                 RootKey = ark("batch.retry.factor")
	BatchRetryInitDelay              RootKey = ark("batch.retry.initDelay")
	BatchRetryMaxDelay               RootKey = ark("batch.retry.maxDelay")
	BlockchainType                   RootKey = ark("blockchain.type")
	BroadcastBatchAgentTimeout       RootKey = ark("broadcast.batch.agentTimeout")
	BroadcastBatchSize               RootKey = ark("broadcast.batch.size")
	BroadcastBatchTimeout            RootKey = ark("broadcast.batch.timeout")
	CorsAllowCredentials             RootKey = ark("cors.credentials")
	CorsAllowedHeaders               RootKey = ark("cors.headers")
	CorsAllowedMethods               RootKey = ark("cors.methods")
	CorsAllowedOrigins               RootKey = ark("cors.origins")
	CorsDebug                        RootKey = ark("cors.debug")
	CorsEnabled                      RootKey = ark("cors.enabled")
	CorsMaxAge                       RootKey = ark("cors.maxAge")
	DatabaseType                     RootKey = ark("database.type")
	DebugPort                        RootKey = ark("debug.port")
	EventTransportsEnabled           RootKey = ark("event.transports.enabled")
	EventAggregatorBatchSize         RootKey = ark("event.aggregator.batchSize")
	EventAggregatorBatchTimeout      RootKey = ark("event.aggregator.batchTimeout")
	EventAggregatorPollTimeout       RootKey = ark("event.aggregator.pollTimeout")
	EventAggregatorRetryFactor       RootKey = ark("event.aggregator.retry.factor")
	EventAggregatorRetryInitDelay    RootKey = ark("event.aggregator.retry.initDelay")
	EventAggregatorRetryMaxDelay     RootKey = ark("event.aggregator.retry.maxDelay")
	EventDispatcherPollTimeout       RootKey = ark("event.dispatcher.pollTimeout")
	EventDispatcherRetryFactor       RootKey = ark("event.dispatcher.retry.factor")
	EventDispatcherRetryInitDelay    RootKey = ark("event.dispatcher.retry.initDelay")
	EventDispatcherRetryMaxDelay     RootKey = ark("event.dispatcher.retry.maxDelay")
	HttpAddress                      RootKey = ark("http.address")
	HttpPort                         RootKey = ark("http.port")
	HttpReadTimeout                  RootKey = ark("http.readTimeout")
	HttpTLSCAFile                    RootKey = ark("http.tls.caFile")
	HttpTLSCertFile                  RootKey = ark("http.tls.certFile")
	HttpTLSClientAuth                RootKey = ark("http.tls.clientAuth")
	HttpTLSEnabled                   RootKey = ark("http.tls.enabled")
	HttpTLSKeyFile                   RootKey = ark("http.tls.keyFile")
	HttpWriteTimeout                 RootKey = ark("http.writeTimeout")
	Lang                             RootKey = ark("lang")
	LogForceColor                    RootKey = ark("log.forceColor")
	LogLevel                         RootKey = ark("log.level")
	LogNoColor                       RootKey = ark("log.noColor")
	LogTimeFormat                    RootKey = ark("log.timeFormat")
	LogUTC                           RootKey = ark("log.utc")
	NamespacesDefault                RootKey = ark("namespaces.default")
	NamespacesPredefined             RootKey = ark("namespaces.predefined")
	NodeIdentity                     RootKey = ark("node.identity")
	OrchestratorStartupAttempts      RootKey = ark("orchestrator.startupAttempts")
	PublicStorageType                RootKey = ark("publicstorage.type")
	SubscriptionDefaultsBatchSize    RootKey = ark("subscription.defaults.batchSize")
	SubscriptionDefaultsBatchTimeout RootKey = ark("subscription.defaults.batchTimeout")
	SubscriptionMaxPerTransport      RootKey = ark("subscription.maxPerTransport")
)

// Config prefix represents the global configuration, at a nested point in
// the config heirarchy. This allows plugins to define their
//
// Note that all values are GLOBAL so this cannot be used for per-instance
// customization. Rather for global initialization of plugins.
type ConfigPrefix interface {
	AddKnownKey(key string, defValue ...interface{})
	SubPrefix(suffix string) ConfigPrefix
	Set(key string, value interface{})
	Resolve(key string) string

	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetUint(key string) uint
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	GetObject(key string) fftypes.JSONObject
	GetObjectArray(key string) fftypes.JSONObjectArray
	Get(key string) interface{}
}

// Key are the known configuration keys
type RootKey string

func Reset() {
	viper.Reset()

	// Set defaults
	viper.SetDefault(string(APIDefaultFilterLimit), 25)
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
	viper.SetDefault(string(BroadcastBatchTimeout), "500ms")
	viper.SetDefault(string(CorsAllowCredentials), true)
	viper.SetDefault(string(CorsAllowedHeaders), []string{"*"})
	viper.SetDefault(string(CorsAllowedMethods), []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete})
	viper.SetDefault(string(CorsAllowedOrigins), []string{"*"})
	viper.SetDefault(string(CorsEnabled), true)
	viper.SetDefault(string(CorsMaxAge), 600)
	viper.SetDefault(string(DebugPort), -1)
	viper.SetDefault(string(EventAggregatorBatchSize), 100)
	viper.SetDefault(string(EventAggregatorBatchTimeout), "250ms")
	viper.SetDefault(string(EventAggregatorPollTimeout), "30s")
	viper.SetDefault(string(EventAggregatorRetryFactor), 2.0)
	viper.SetDefault(string(EventAggregatorRetryInitDelay), "250ms")
	viper.SetDefault(string(EventAggregatorRetryMaxDelay), "30s")
	viper.SetDefault(string(EventTransportsEnabled), []string{"websocket"})
	viper.SetDefault(string(HttpAddress), "127.0.0.1")
	viper.SetDefault(string(HttpPort), 5000)
	viper.SetDefault(string(HttpReadTimeout), "15s")
	viper.SetDefault(string(HttpWriteTimeout), "15s")
	viper.SetDefault(string(Lang), "en")
	viper.SetDefault(string(LogLevel), "info")
	viper.SetDefault(string(LogTimeFormat), "2006-01-02T15:04:05.000Z07:00")
	viper.SetDefault(string(LogUTC), false)
	viper.SetDefault(string(NamespacesDefault), "default")
	viper.SetDefault(string(NamespacesPredefined), fftypes.JSONObjectArray{{"name": "default", "description": "Default predefined namespace"}})
	viper.SetDefault(string(OrchestratorStartupAttempts), 5)
	viper.SetDefault(string(SubscriptionMaxPerTransport), 500)

	i18n.SetLang(GetString(Lang))
}

// ReadConfig initializes the config
func ReadConfig(cfgFile string) error {
	Reset()

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
	} else {
		viper.SetConfigName("firefly.core")
		viper.AddConfigPath("/etc/firefly/")
		viper.AddConfigPath("$HOME/.firefly")
		viper.AddConfigPath(".")
		return viper.ReadInConfig()
	}
}

var root *configPrefix = &configPrefix{
	keys: map[string]bool{}, // All keys go here, including those defined in sub prefixies
}

// ark adds a root key, used to define the keys that are used within the core
func ark(k string) RootKey {
	root.AddKnownKey(k)
	return RootKey(k)
}

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
func NewPluginConfig(prefix string) ConfigPrefix {
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
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

func (c *configPrefix) SubPrefix(suffix string) ConfigPrefix {
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

// GetUInt gets a configuration uint
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

// GetFloat gets a configuration uint
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
