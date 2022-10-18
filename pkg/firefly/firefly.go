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

package firefly

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
)

const configSuffix = "core"

var sigs = make(chan os.Signal, 1)

var _utManager namespace.Manager

func ResetConfig() {
	coreconfig.Reset()
	apiserver.InitConfig()
}

func ShowConfig(cfgFile string) string {
	// Initialize config of all plugins
	ResetConfig()
	GetRootManager()
	_ = config.ReadConfig(configSuffix, cfgFile)

	// Print it all out
	result := fmt.Sprintf("%-64s %v\n", "Key", "Value")
	result = fmt.Sprintf("%s-----------------------------------------------------------------------------------\n", result)
	for _, k := range config.GetKnownKeys() {
		result = fmt.Sprintf("%s%-64s %v\n", result, k, config.Get(config.RootKey(k)))
	}
	return result
}

func GetRootManager() namespace.Manager {
	if _utManager != nil {
		return _utManager
	}
	return namespace.NewNamespaceManager(true)
}

func Run(cfgFile string) error {

	// Read the configuration
	ResetConfig()
	err := config.ReadConfig(configSuffix, cfgFile)

	// Setup logging after reading config (even if failed), to output header correctly
	rootCtx, cancelRootCtx := context.WithCancel(context.Background())
	rootCtx = log.WithLogger(rootCtx, logrus.WithField("pid", fmt.Sprintf("%d", os.Getpid())))

	config.SetupLogging(rootCtx)
	log.L(rootCtx).Infof("Hyperledger FireFly")
	log.L(rootCtx).Infof("© Copyright 2022 Kaleido, Inc.")

	// Deferred error return from reading config
	if err != nil {
		cancelRootCtx()
		return i18n.WrapError(rootCtx, err, i18n.MsgConfigFailed)
	}

	// Setup signal handling to cancel the context, which shuts down the API Server
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		log.L(rootCtx).Infof("Starting up")
		runCtx, cancelRunCtx := context.WithCancel(rootCtx)
		mgr := GetRootManager()
		as := apiserver.NewAPIServer()
		errChan := make(chan error, 1)
		resetChan := make(chan bool, 1)
		ffDone := make(chan struct{})
		go startFirefly(runCtx, cancelRootCtx, mgr, as, errChan, resetChan, ffDone)
		select {
		case sig := <-sigs:
			log.L(rootCtx).Infof("Shutting down due to %s", sig.String())
			cancelRunCtx()
			mgr.WaitStop()
			return nil
		case <-rootCtx.Done():
			log.L(rootCtx).Infof("Shutting down due to cancelled context")
			cancelRunCtx()
			mgr.WaitStop()
			return nil
		case <-resetChan:
			log.L(rootCtx).Infof("Restarting due to configuration change")
			cancelRunCtx()
			mgr.WaitStop()
			// Must wait for the server to close before we restart
			<-ffDone
			// Re-read the configuration
			ResetConfig()
			if err := config.ReadConfig(configSuffix, cfgFile); err != nil {
				return err
			}
		case err := <-errChan:
			cancelRunCtx()
			return err
		}
	}
}

func startFirefly(ctx context.Context, cancelCtx context.CancelFunc, mgr namespace.Manager, as apiserver.Server, errChan chan error, resetChan chan bool, ffDone chan struct{}) {
	var err error
	// Start debug listener
	var debugServer *http.Server
	debugPort := config.GetInt(coreconfig.DebugPort)
	debugAddress := config.GetString(coreconfig.DebugAddress)

	if debugPort >= 0 {
		r := mux.NewRouter()
		r.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
		r.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
		r.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
		r.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		debugServer = &http.Server{Addr: fmt.Sprintf("%s:%d", debugAddress, debugPort), Handler: r, ReadHeaderTimeout: 30 * time.Second}
		go func() {
			_ = debugServer.ListenAndServe()
		}()
		log.L(ctx).Debugf("Debug HTTP endpoint listening on %s:%d", debugAddress, debugPort)
	}

	defer func() {
		if debugServer != nil {
			_ = debugServer.Close()
		}
		close(ffDone)
	}()

	if err = mgr.Init(ctx, cancelCtx, resetChan); err != nil {
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
