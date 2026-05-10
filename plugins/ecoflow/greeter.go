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
	"github.com/tknie/energymonitor"
	"github.com/tknie/services"
)

type greeting string

var adapter *energymonitor.AdapterConfig

const (
	checkMediaNr byte = iota
)

func init() {
}

// Types type of plugin working with
func (g greeting) Info() *energymonitor.PluginInfo {
	return &energymonitor.PluginInfo{
		Name:         "Ecoflow",
		Version:      "1.0",
		Types:        []energymonitor.PluginTypes{energymonitor.DevicePlugin},
		Identifier:   []string{"ecoflow"},
		AbortOnError: false,
	}
}

// Name name of the plugin
func (g greeting) Name() string {
	return "Ecoflow"
}

// Version version of the number
func (g greeting) Version() string {
	return "1.0"
}

// Stop stop plugin
func (g greeting) Stop() {
}

func (g greeting) GetPower(converter string) (float64, error) {
	return ecoflowCurrentPowerRequest(), nil
}

// SetPower set power to device
func (g greeting) SetPower(converter string, power float64) error {
	return nil
}

func (g greeting) Register(config *energymonitor.AdapterConfig) {
	adapter = config
	for _, device := range config.DevicesConfig.EnergySources {
		if device.Type == "ecoflow" {
			services.ServerMessage("Register Ecoflow device: %s", device.MicroConverter)
		}
	}
	InitEcoflow()
	services.ServerMessage("Ecoflow devices registered")
}

// exported

// Loader loader for initialize plugin
var Loader greeting

// EntryPoint entry point for main structure
var EntryPoint greeting
