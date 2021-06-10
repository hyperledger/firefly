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

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/hyperledger-labs/firefly/internal/apiserver"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/orchestrator"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

var showConfigCommand = &cobra.Command{
	Use:     "showconfig",
	Aliases: []string{"showconf"},
	Short:   "List out the configuration options",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize config of all plugins
		getOrchestrator()
		_ = config.ReadConfig(cfgFile)

		// Print it all out
		for _, k := range config.GetKnownKeys() {
			fmt.Printf("%-64s %v\n", k, config.Get(config.RootKey(k)))
		}
	},
}

var cfgFile string

var _utOrchestrator orchestrator.Orchestrator

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file")
	rootCmd.AddCommand(showConfigCommand)
}

func getOrchestrator() orchestrator.Orchestrator {
	if _utOrchestrator != nil {
		return _utOrchestrator
	}
	return orchestrator.NewOrchestrator()
}

// Execute is called by the main method of the package
func Execute() error {
	return rootCmd.Execute()
}

func run() error {

	// Read the configuration
	err := config.ReadConfig(cfgFile)

	// Setup logging after reading config (even if failed), to output header correctly
	ctx, cancelCtx := context.WithCancel(context.Background())
	ctx = log.WithLogger(ctx, logrus.WithField("pid", os.Getpid()))

	config.SetupLogging(ctx)
	log.L(ctx).Infof("Project Firefly")
	log.L(ctx).Infof("© Copyright 2021 Kaleido, Inc.")

	// Deferred error return from reading config
	if err != nil {
		cancelCtx()
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed)
	}

	// Setup signal handling to cancel the context, which shuts down the API Server
	errChan := make(chan error)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		orchestratorCtx, cancelOrchestratorCtx := context.WithCancel(ctx)
		o := getOrchestrator()
		go startFirefly(orchestratorCtx, cancelOrchestratorCtx, o, errChan)
		select {
		case sig := <-sigs:
			log.L(ctx).Infof("Shutting down due to %s", sig.String())
			cancelCtx()
			o.WaitStop()
			return nil
		case <-orchestratorCtx.Done():
			log.L(ctx).Infof("Restarting due to configuration change")
			o.WaitStop()
		case err := <-errChan:
			cancelCtx()
			return err
		}
	}
}

func startFirefly(ctx context.Context, cancelCtx context.CancelFunc, o orchestrator.Orchestrator, errChan chan error) {
	var err error
	// Start debug listener
	debugPort := config.GetInt(config.DebugPort)
	if debugPort >= 0 {
		r := mux.NewRouter()
		r.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
		r.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
		r.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
		r.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		go func() {
			_ = http.ListenAndServe(fmt.Sprintf("localhost:%d", debugPort), r)
		}()
		log.L(ctx).Debugf("Debug HTTP endpoint listening on localhost:%d", debugPort)
	}

	if err = o.Init(ctx, cancelCtx); err != nil {
		errChan <- err
		return
	}
	if err = o.Start(); err != nil {
		errChan <- err
		return
	}

	// Run the API Server
	if err = apiserver.Serve(ctx, o); err != nil {
		errChan <- err
	}
}
