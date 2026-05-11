/*
* Copyright 2025-2026 Thorsten A. Knieling
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
 */

package energymonitor

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/tknie/log"
	"github.com/tknie/services"
)

var quit = make(chan struct{})
var httpDone = make(chan bool, 1)

func setupGracefulShutdown(done chan bool) {
	// Create a channel to listen for OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle shutdown
	go func() {
		s := <-signalChan
		log.Log.Errorf("Received shutdown signal: %s", s)
		services.ServerMessage("Energy Monitor shutdown signal received: %s", s)

		endHttp()
		close(quit)
		done <- true
	}()
}

// endHttp end Database store of HTTP data
func endHttp() {
	httpDone <- true
}

func InitDevices() {
	done := make(chan bool, 1)
	setupGracefulShutdown(done)
	services.ServerMessage("Initialize devices")
	startStatLoop()
	InitPlugins()
	services.ServerMessage("Devices up and running, waiting...")

	<-done
	log.Log.Debugf("Shutting down plugins")
	ShutdownPlugins()
}
