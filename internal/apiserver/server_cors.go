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

package apiserver

import (
	"context"
	"net/http"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/rs/cors"
)

func wrapCorsIfEnabled(ctx context.Context, chain http.Handler) http.Handler {
	if !config.GetBool(config.CorsEnabled) {
		return chain
	}
	corsOptions := cors.Options{
		AllowedOrigins:   config.GetStringSlice(config.CorsAllowedOrigins),
		AllowedMethods:   config.GetStringSlice(config.CorsAllowedMethods),
		AllowedHeaders:   config.GetStringSlice(config.CorsAllowedHeaders),
		AllowCredentials: config.GetBool(config.CorsAllowCredentials),
		MaxAge:           config.GetInt(config.CorsMaxAge),
		Debug:            config.GetBool(config.CorsDebug),
	}
	log.L(ctx).Debugf("CORS origins=%v methods=%v headers=%v creds=%t maxAge=%d",
		corsOptions.AllowedOrigins,
		corsOptions.AllowedMethods,
		corsOptions.AllowedHeaders,
		corsOptions.AllowCredentials,
		corsOptions.MaxAge,
	)
	c := cors.New(corsOptions)
	return c.Handler(chain)
}
