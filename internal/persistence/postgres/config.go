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

package postgres

import (
	"github.com/kaleido-io/firefly/internal/config"
)

const (
	prefix              = "database"
	URL                 = "url"
	AutoMigrate         = "autoMigrate"
	MigrationsDirectory = "migrationsDirectory"
)

var defaults = map[string]interface{}{
	AutoMigrate:         false,
	MigrationsDirectory: "./db/migrations/postgres",
}

type Config struct {
	URL                 string
	AutoMigrate         bool
	MigrationsDirectory string
}

func NewConfig(config config.PluginConfig) *Config {
	for k, v := range defaults {
		config.SetDefault(prefix+"."+k, v)
	}

	return &Config{
		URL:                 config.GetString(prefix + "." + URL),
		AutoMigrate:         config.GetBool(prefix + "." + AutoMigrate),
		MigrationsDirectory: config.GetString(prefix + "." + MigrationsDirectory),
	}
}
