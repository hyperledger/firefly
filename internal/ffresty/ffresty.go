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

package ffresty

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

type retryCtxKey struct{}

type retryCtx struct {
	id       string
	start    time.Time
	attempts uint
}

// When using SetDoNotParseResponse(true) for streming binary replies,
// the caller should invoke ffrest.OnAfterResponse on the response manually.
// The middleware is disabled on this path :-(
// See: https://github.com/go-resty/resty/blob/d01e8d1bac5ba1fed0d9e03c4c47ca21e94a7e8e/client.go#L912-L948
func OnAfterResponse(c *resty.Client, resp *resty.Response) {
	if c == nil || resp == nil {
		return
	}
	rctx := resp.Request.Context()
	rc := rctx.Value(retryCtxKey{}).(*retryCtx)
	elapsed := float64(time.Since(rc.start)) / float64(time.Millisecond)
	log.L(rctx).Infof("<== %s %s [%d] (%.2fms)", resp.Request.Method, resp.Request.URL, resp.StatusCode(), elapsed)
}

// New creates a new Resty client, using static configuration (from the config file)
// from a given nested prefix in the static configuration
//
// You can use the normal Resty builder pattern, to set per-instance configuration
// as required.
func New(ctx context.Context, staticConfig config.ConfigPrefix) *resty.Client {

	var client *resty.Client

	iHttpClient := staticConfig.Get(HTTPCustomClient)
	if iHttpClient != nil {
		if httpClient, ok := iHttpClient.(*http.Client); ok {
			client = resty.NewWithClient(httpClient)
		}
	}
	if client == nil {
		client = resty.New()
	}

	url := strings.TrimSuffix(staticConfig.GetString(HTTPConfigURL), "/")
	if url != "" {
		client.SetHostURL(url)
		log.L(ctx).Debugf("Created REST client to %s", url)
	}

	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		rctx := req.Context()
		rc := rctx.Value(retryCtxKey{})
		if rc == nil {
			// First attempt
			r := &retryCtx{
				id:    fftypes.ShortID(),
				start: time.Now(),
			}
			rctx = context.WithValue(rctx, retryCtxKey{}, r)
			// Create a request logger from the root logger passed into the client
			l := log.L(ctx).WithField("breq", r.id)
			rctx = log.WithLogger(rctx, l)
			req.SetContext(rctx)
		}
		log.L(rctx).Infof("==> %s %s%s", req.Method, url, req.URL)
		return nil
	})

	// Note that callers using SetNotParseResponse will need to invoke this themselves

	client.OnAfterResponse(func(c *resty.Client, r *resty.Response) error { OnAfterResponse(c, r); return nil })

	headers := staticConfig.GetStringMap(HTTPConfigHeaders)
	for k, v := range headers {
		if vs, ok := v.(string); ok {
			client.SetHeader(k, vs)
		}
	}
	authUsername := staticConfig.GetString((HTTPConfigAuthUsername))
	authPassword := staticConfig.GetString((HTTPConfigAuthPassword))
	if authUsername != "" && authPassword != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", authUsername, authPassword)))))
	}

	if staticConfig.GetBool(HTTPConfigRetryEnabled) {
		retryCount := staticConfig.GetInt(HTTPConfigRetryCount)
		minTimeout := staticConfig.GetUint(HTTPConfigRetryWaitTimeMS)
		maxTimeout := staticConfig.GetUint(HTTPConfigRetryMaxWaitTimeMS)
		client.
			SetRetryCount(retryCount).
			SetRetryWaitTime(time.Duration(minTimeout) * time.Millisecond).
			SetRetryMaxWaitTime(time.Duration(maxTimeout) * time.Millisecond).
			AddRetryCondition(func(r *resty.Response, err error) bool {
				if r == nil || r.IsSuccess() {
					return false
				}
				rctx := r.Request.Context()
				rc := rctx.Value(retryCtxKey{}).(*retryCtx)
				log.L(rctx).Infof("retry %d/%d (min=%dms/max=%dms) status=%d", rc.attempts, retryCount, minTimeout, maxTimeout, r.StatusCode())
				rc.attempts++
				return true
			})
	}

	return client
}

func WrapRestErr(ctx context.Context, res *resty.Response, err error, key i18n.MessageKey) error {
	var respData string
	if res != nil {
		if res.RawBody() != nil {
			defer func() { _ = res.RawBody().Close() }()
			if r, err := ioutil.ReadAll(res.RawBody()); err == nil {
				respData = string(r)
			}
		}
		if respData == "" {
			respData = res.String()
		}
		if len(respData) > 256 {
			respData = respData[0:256] + "..."
		}
	}
	if err != nil {
		return i18n.WrapError(ctx, err, key, respData)
	}
	return i18n.NewError(ctx, key, respData)
}
