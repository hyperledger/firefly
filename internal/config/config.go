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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/spf13/viper"
)

// The following keys can be access from the root configuration.
// Plugins are resonsible for defining their own keys using the Config interface
var (
	Lang                       RootKey = ark("lang")
	LogLevel                   RootKey = ark("log.level")
	LogColor                   RootKey = ark("log.color")
	DebugPort                  RootKey = ark("debug.port")
	HttpAddress                RootKey = ark("http.address")
	HttpPort                   RootKey = ark("http.port")
	HttpReadTimeout            RootKey = ark("http.readTimeout")
	HttpWriteTimeout           RootKey = ark("http.writeTimeout")
	HttpTLSEnabled             RootKey = ark("http.tls.enabled")
	HttpTLSClientAuth          RootKey = ark("http.tls.clientAuth")
	HttpTLSCAFile              RootKey = ark("http.tls.caFile")
	HttpTLSCertFile            RootKey = ark("http.tls.certFile")
	HttpTLSKeyFile             RootKey = ark("http.tls.keyFile")
	CorsEnabled                RootKey = ark("cors.enabled")
	CorsAllowedOrigins         RootKey = ark("cors.origins")
	CorsAllowedMethods         RootKey = ark("cors.methods")
	CorsAllowedHeaders         RootKey = ark("cors.headers")
	CorsAllowCredentials       RootKey = ark("cors.credentials")
	CorsMaxAge                 RootKey = ark("cors.maxAge")
	CorsDebug                  RootKey = ark("cors.debug")
	NodeIdentity               RootKey = ark("node.identity")
	APIRequestTimeout          RootKey = ark("api.requestTimeout")
	APIDefaultFilterLimit      RootKey = ark("api.defaultFilterLimit")
	Database                   RootKey = ark("database")
	DatabaseType               RootKey = ark("database.type")
	BlockchainType             RootKey = ark("blockchain.type")
	Blockchain                 RootKey = ark("blockchain")
	P2PFSType                  RootKey = ark("p2pfs.type")
	P2PFS                      RootKey = ark("p2pfs")
	BroadcastBatchSize         RootKey = ark("broadcast.batch.size")
	BroadcastBatchTimeout      RootKey = ark("broadcast.batch.timeout")
	BroadcastBatchAgentTimeout RootKey = ark("broadcast.batch.agentTimeout")
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

	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetUint(key string) uint
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	UnmarshalKey(ctx context.Context, key string, rawVal interface{}) error
	Get(key string) interface{}
}

// Key are the known configuration keys
type RootKey string

func Reset() {
	viper.Reset()

	// Set defaults
	viper.SetDefault(string(Lang), "en")
	viper.SetDefault(string(LogLevel), "info")
	viper.SetDefault(string(LogColor), true)
	viper.SetDefault(string(DebugPort), -1)
	viper.SetDefault(string(HttpAddress), "127.0.0.1")
	viper.SetDefault(string(HttpPort), 5000)
	viper.SetDefault(string(HttpReadTimeout), 15000)
	viper.SetDefault(string(HttpWriteTimeout), 15000)
	viper.SetDefault(string(CorsEnabled), true)
	viper.SetDefault(string(CorsAllowedOrigins), []string{"*"})
	viper.SetDefault(string(CorsAllowedMethods), []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete})
	viper.SetDefault(string(CorsAllowedHeaders), []string{"*"})
	viper.SetDefault(string(CorsAllowCredentials), true)
	viper.SetDefault(string(CorsMaxAge), 600)
	viper.SetDefault(string(APIRequestTimeout), 12000)
	viper.SetDefault(string(APIDefaultFilterLimit), 25)
	viper.SetDefault(string(BroadcastBatchSize), 200)
	viper.SetDefault(string(BroadcastBatchTimeout), 500)
	viper.SetDefault(string(BroadcastBatchAgentTimeout), 120000)

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

// GetStringMap gets a configuration map
func GetStringMap(key RootKey) map[string]interface{} {
	return root.GetStringMap(string(key))
}
func (c *configPrefix) GetStringMap(key string) map[string]interface{} {
	return viper.GetStringMap(c.prefixKey(key))
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

// Unmarshal gets a configuration section into a struct
func UnmarshalKey(ctx context.Context, key RootKey, rawVal interface{}) error {
	return root.UnmarshalKey(ctx, string(key), rawVal)
}
func (c *configPrefix) UnmarshalKey(ctx context.Context, key string, rawVal interface{}) error {
	// Viper's unmarshal does not work with our json annotated config
	// structures, so we have to go from map to JSON, then to unmarshal
	var intermediate map[string]interface{}
	err := viper.UnmarshalKey(c.prefixKey(key), &intermediate)
	if err == nil {
		b, _ := json.Marshal(intermediate)
		err = json.Unmarshal(b, rawVal)
	}
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed, key)
	}
	return nil
}
