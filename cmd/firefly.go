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

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/sirupsen/logrus"
)

// Execute is called by the main method of the package
func Execute() error {

	// Read the configuration first of all
	err := config.ReadConfig()

	// Setup logging after reading config (even if failed), to output header correctly
	ctx := log.WithLogger(context.Background(), logrus.WithField("pid", os.Getpid()))
	log.SetupLogging(ctx)
	log.L(ctx).Infof("Project Firefly")
	log.L(ctx).Infof("© Copyright 2021 Kaleido, Inc.")

	// Deferred error return from reading config
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed)
	}

	debugPort := config.GetUint(config.DebugPort)
	if debugPort > 0 {
		go func() {
			log.L(ctx).Debugf("Debug HTTP endpoint listening on localhost:%d: %s", debugPort, http.ListenAndServe(fmt.Sprintf("localhost:%d", debugPort), nil))
		}()
	}

	return nil

}
