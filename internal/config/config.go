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
	"net/http"
	"os"
	"strings"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/spf13/viper"
)

type PluginConfig interface {
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	SetDefault(key string, value interface{})
}

// Key are the known configuration keys
type Key string

const (
	Lang                       Key = "lang"
	LogLevel                   Key = "log.level"
	LogColor                   Key = "log.color"
	DebugPort                  Key = "debug.port"
	HttpAddress                Key = "http.address"
	HttpPort                   Key = "http.port"
	HttpReadTimeout            Key = "http.readTimeout"
	HttpWriteTimeout           Key = "http.writeTimeout"
	HttpTLSEnabled             Key = "http.tls.enabled"
	HttpTLSClientAuth          Key = "http.tls.clientAuth"
	HttpTLSCAFile              Key = "http.tls.caFile"
	HttpTLSCertFile            Key = "http.tls.certFile"
	HttpTLSKeyFile             Key = "http.tls.keyFile"
	CorsEnabled                Key = "cors.enabled"
	CorsAllowedOrigins         Key = "cors.origins"
	CorsAllowedMethods         Key = "cors.methods"
	CorsAllowedHeaders         Key = "cors.headers"
	CorsAllowCredentials       Key = "cors.credentials"
	CorsMaxAge                 Key = "cors.maxAge"
	CorsDebug                  Key = "cors.debug"
	NodeIdentity               Key = "node.identity"
	APIRequestTimeout          Key = "api.requestTimeout"
	APIDefaultFilterLimit      Key = "api.defaultFilterLimit"
	Database                   Key = "database"
	DatabaseType               Key = "database.type"
	BlockchainType             Key = "blockchain.type"
	Blockchain                 Key = "blockchain"
	P2PFSType                  Key = "p2pfs.type"
	P2PFS                      Key = "p2pfs"
	BroadcastBatchSize         Key = "broadcast.batch.size"
	BroadcastBatchTimeout      Key = "broadcast.batch.timeout"
	BroadcastBatchAgentTimeout Key = "broadcast.batch.agentTimeout"
)

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

// GetString gets a configuration string
func GetString(key Key) string {
	return viper.GetString(string(key))
}

// GetStringSlice gets a configuration string array
func GetStringSlice(key Key) []string {
	return viper.GetStringSlice(string(key))
}

// GetBool gets a configuration bool
func GetBool(key Key) bool {
	return viper.GetBool(string(key))
}

// GetUInt gets a configuration uint
func GetUint(key Key) uint {
	return viper.GetUint(string(key))
}

// GetInt gets a configuration uint
func GetInt(key Key) int {
	return viper.GetInt(string(key))
}

// Set allows runtime setting of config (used in unit tests)
func Set(key Key, value interface{}) {
	viper.Set(string(key), value)
}

// Unmarshal gets a configuration section into a struct
func UnmarshalKey(ctx context.Context, key Key, rawVal interface{}) error {
	// Viper's unmarshal does not work with our json annotated config
	// structures, so we have to go from map to JSON, then to unmarshal
	var intermediate map[string]interface{}
	err := viper.UnmarshalKey(string(key), &intermediate)
	if err == nil {
		b, _ := json.Marshal(intermediate)
		err = json.Unmarshal(b, rawVal)
	}
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed, key)
	}
	return nil
}

// UintWithDefault is a helper for addressing optional fields with a default in unmarshalled JSON structs
func UintWithDefault(val *uint, def uint) uint {
	if val == nil {
		return def
	}
	return *val
}

// BoolWithDefault is a helper for addressing optional fields with a default in unmarshalled JSON structs
func BoolWithDefault(val *bool, def bool) bool {
	if val == nil {
		return def
	}
	return *val
}

// StringWithDefault is a helper for addressing optional fields with a default in unmarshalled JSON structs
func StringWithDefault(val *string, def string) string {
	if val == nil {
		return def
	}
	return *val
}

func GetConfig() PluginConfig {
	return viper.GetViper()
}
