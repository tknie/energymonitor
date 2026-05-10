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

package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/tknie/energymonitor"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

func init() {
	services.ServerMessage("Start energymonitor application %s (build at %v)", energymonitor.Version, energymonitor.BuildDate)
	log.InitZapLogWithFilename("energymonitor.log")
}

func main() {
	create := false
	energymonitor.LoopSeconds = energymonitor.DefaultSeconds
	seconds := os.Getenv("energymonitor_WAIT_SECONDS")
	if seconds != "" {
		sec, err := strconv.Atoi(seconds)
		if err != nil {
			services.ServerMessage("Invalid wait seconds given, use default %d seconds", energymonitor.LoopSeconds)
		} else {
			energymonitor.LoopSeconds = sec
		}
	}
	statSecs := 0
	powervalue := float64(0)
	readFlow := false
	flowControlFile := ""
	flow := false
	caracon := false
	serialNumber := ""
	listDevices := false

	flag.IntVar(&energymonitor.LoopSeconds, "t", energymonitor.LoopSeconds, "The seconds wating between REST API queries")
	flag.IntVar(&statSecs, "s", int(energymonitor.StatLoopMinutes), "The minutes waiting between statistics output")
	flag.BoolVar(&energymonitor.MqttDisable, "m", false, "Disable MQTT listener")
	flag.BoolVar(&readFlow, "r", false, "Read current flow parameter")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.BoolVar(&flow, "a", false, "Start energy analyze")
	flag.BoolVar(&caracon, "c", false, "Power car AC on")
	flag.BoolVar(&energymonitor.Test, "T", false, "Do tests and output only")
	flag.BoolVar(&listDevices, "l", false, "List of Ecoflow devices")
	flag.StringVar(&serialNumber, "S", "", "Use serial number")
	flag.StringVar(&flowControlFile, "f", "", "Load YAML control file")
	flag.Float64Var(&powervalue, "p", 0, "Set new power value for the power powerstream")

	flag.Parse()

	// Go into server mode
	energymonitor.StatLoopMinutes = time.Duration(statSecs)

	if flowControlFile == "" {
		flowControlFile = os.Getenv("energymonitor_CONFIG")
	}

	if flowControlFile != "" {
		energymonitor.LoadConfig(flowControlFile)
	}

	energymonitor.InitDatabase()
	energymonitor.InitDevices()
}
