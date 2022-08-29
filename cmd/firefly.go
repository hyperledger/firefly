// Copyright © 2022 Kaleido, Inc.
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
	"time"

	"github.com/gorilla/mux"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/apiserver"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/namespace"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const configSuffix = "core"

var sigs = make(chan os.Signal, 1)

var rootCmd = &cobra.Command{
	Use:   "firefly",
	Short: "FireFly is a complete stack for enterprises to build and scale secure Web3 applications",
	Long: `Hyperledger FireFly is the first open source Supernode: a complete stack for
enterprises to build and scale secure Web3 applications. The FireFly API for digital
assets, data flows, and blockchain transactions makes it radically faster to build
production-ready apps on popular chains and protocols.`,
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
		resetConfig()
		getRootManager()
		_ = config.ReadConfig(configSuffix, cfgFile)

		// Print it all out
		fmt.Printf("%-64s %v\n", "Key", "Value")
		fmt.Print("-----------------------------------------------------------------------------------\n")
		for _, k := range config.GetKnownKeys() {
			fmt.Printf("%-64s %v\n", k, config.Get(config.RootKey(k)))
		}
	},
}

var cfgFile string

var _utManager namespace.Manager

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file")
	rootCmd.AddCommand(showConfigCommand)
}

func resetConfig() {
	coreconfig.Reset()
	apiserver.InitConfig()
}

func getRootManager() namespace.Manager {
	if _utManager != nil {
		return _utManager
	}
	return namespace.NewNamespaceManager(true)
}

// Execute is called by the main method of the package
func Execute() error {
	return rootCmd.Execute()
}

func run() error {

	// Read the configuration
	resetConfig()
	err := config.ReadConfig(configSuffix, cfgFile)

	// Setup logging after reading config (even if failed), to output header correctly
	ctx, cancelCtx := context.WithCancel(context.Background())
	ctx = log.WithLogger(ctx, logrus.WithField("pid", fmt.Sprintf("%d", os.Getpid())))

	config.SetupLogging(ctx)
	log.L(ctx).Infof("Hyperledger FireFly")
	log.L(ctx).Infof("© Copyright 2022 Kaleido, Inc.")

	// Deferred error return from reading config
	if err != nil {
		cancelCtx()
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed)
	}

	// Setup signal handling to cancel the context, which shuts down the API Server
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		log.L(ctx).Infof("Starting up")
		rootCtx, rootCancelCtx := context.WithCancel(ctx)
		mgr := getRootManager()
		as := apiserver.NewAPIServer()
		errChan := make(chan error, 1)
		ffDone := make(chan struct{})
		go startFirefly(rootCtx, rootCancelCtx, mgr, as, errChan, ffDone)
		select {
		case sig := <-sigs:
			log.L(ctx).Infof("Shutting down due to %s", sig.String())
			cancelCtx()
			mgr.WaitStop()
			return nil
		case <-rootCtx.Done():
			log.L(ctx).Infof("Restarting due to configuration change")
			mgr.WaitStop()
			// Must wait for the server to close before we restart
			<-ffDone
			// Re-read the configuration
			resetConfig()
			if err := config.ReadConfig(configSuffix, cfgFile); err != nil {
				cancelCtx()
				return err
			}
		case err := <-errChan:
			cancelCtx()
			return err
		}
	}
}

func startFirefly(ctx context.Context, cancelCtx context.CancelFunc, mgr namespace.Manager, as apiserver.Server, errChan chan error, ffDone chan struct{}) {
	var err error
	// Start debug listener
	var debugServer *http.Server
	debugPort := config.GetInt(coreconfig.DebugPort)
	if debugPort >= 0 {
		r := mux.NewRouter()
		r.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
		r.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
		r.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
		r.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		debugServer = &http.Server{Addr: fmt.Sprintf("localhost:%d", debugPort), Handler: r, ReadHeaderTimeout: 30 * time.Second}
		go func() {
			_ = debugServer.ListenAndServe()
		}()
		log.L(ctx).Debugf("Debug HTTP endpoint listening on localhost:%d", debugPort)
	}

	defer func() {
		if debugServer != nil {
			_ = debugServer.Close()
		}
		close(ffDone)
	}()

	if err = mgr.Init(ctx, cancelCtx); err != nil {
		errChan <- err
		return
	}
	if err = mgr.Start(); err != nil {
		errChan <- err
		return
	}

	// Run the API Server

	if err = as.Serve(ctx, mgr); err != nil {
		errChan <- err
	}
}
