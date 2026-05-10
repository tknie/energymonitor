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
	"fmt"

	"github.com/tknie/energymonitor"
	"github.com/tknie/services"
)

type greeting string

const (
	checkMediaNr byte = iota
)

var adapter *energymonitor.AdapterConfig

func init() {
}

// Types type of plugin working with
func (g greeting) Info() *energymonitor.PluginInfo {
	return &energymonitor.PluginInfo{
		Name:         "Marstek",
		Version:      "1.0",
		Types:        []energymonitor.PluginTypes{energymonitor.DevicePlugin},
		Identifier:   []string{"marstek"},
		AbortOnError: false,
	}
}

// Name name of the plugin
func (g greeting) Name() string {
	return "Marstek"
}

// Version version of the number
func (g greeting) Version() string {
	return "1.0"
}

func (g greeting) GetPower(converter string) ([]float64, error) {
	return []float64{0, 0}, fmt.Errorf("Not implemented yet")
}

func (g greeting) Converter() []string {
	converters := make([]string, 0)
	for _, device := range adapter.DevicesConfig.EnergySources {
		if device.Type == "marstek" {
			converters = append(converters, device.MicroConverter)
		}
	}
	return converters
}

// SetPower set power to device
func (g greeting) SetPower(converter string, power float64) (float64, error) {
	return 0, nil
}

// Stop stop plugin
func (g greeting) Stop() {
}

func (g greeting) Register(config *energymonitor.AdapterConfig) {
	adapter = config
	for _, device := range config.DevicesConfig.EnergySources {
		if device.Type == "marstek" {
			services.ServerMessage("Register Marstek device: %s", device.MicroConverter)
			InitMastek()
		}
	}
}

// exported

// Loader loader for initialize plugin
var Loader greeting

// EntryPoint entry point for main structure
var EntryPoint greeting
