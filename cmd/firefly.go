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
	"os/signal"
	"syscall"

	"github.com/kaleido-io/firefly/internal/apiserver"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var sigs = make(chan os.Signal, 1)

var rootCmd = &cobra.Command{
	Use:   "firefly",
	Short: "Firefly is an API toolkit for building enterprise grade multi-party systems",
	Long: `You build great user experiences and business logic in your favorite language,
and let Firefly take care of the REST. The event-driven programming model gives you the
building blocks needed for high performance, scalable multi-party systems, and the power
to digital transformation your business ecosystem.`,
	Run: func(cmd *cobra.Command, args []string) {},
}

var cfgFile string

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file")
}

// Execute is called by the main method of the package
func Execute() error {
	err := rootCmd.Execute()
	if err == nil {
		err = run()
	}
	return err
}

func run() error {

	// Read the configuration
	err := config.ReadConfig(cfgFile)

	// Setup logging after reading config (even if failed), to output header correctly
	ctx, cancelCtx := context.WithCancel(context.Background())
	ctx = log.WithLogger(ctx, logrus.WithField("pid", os.Getpid()))
	log.SetupLogging(ctx)
	log.L(ctx).Infof("Project Firefly")
	log.L(ctx).Infof("© Copyright 2021 Kaleido, Inc.")

	// Setup signal handling to cancel the context, which shuts down the API Server
	done := make(chan struct{})
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigs:
			log.L(ctx).Infof("Shutting down due to %s", sig.String())
			cancelCtx()
		case <-done:
		}
	}()
	defer close(done)

	// Deferred error return from reading config
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed)
	}

	// Start debug listener
	debugPort := config.GetInt(config.DebugPort)
	if debugPort >= 0 {
		go func() {
			log.L(ctx).Debugf("Debug HTTP endpoint listening on localhost:%d: %s", debugPort, http.ListenAndServe(fmt.Sprintf("localhost:%d", debugPort), nil))
		}()
	}

	// Run the API Server
	return apiserver.Serve(ctx)

}
