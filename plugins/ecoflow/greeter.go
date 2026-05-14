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
	for _, ecoDevice := range ecoflowDeviceMap {
		services.ServerMessage("Reset power of %s to %02f", ecoDevice.converter, float64(adapter.DefaultConfig.BaseRequest))
		if ecoDevice.serialNumber != "" {
			ecoDevice.SetEcoflowPowerConsumption(float64(adapter.DefaultConfig.BaseRequest))
		}
	}
	adapter.Close()
}

func (g greeting) GetPower(converter string) ([]float64, error) {
	if converter == "" {
		return []float64{0, 0}, nil
	}
	if ecoDevice, ok := ecoflowDeviceMap[converter]; ok {
		return ecoDevice.ecoflowCurrentPowerRequest(), nil
	}
	return []float64{0, 0}, nil
}

// SetPower set power to device
func (g greeting) SetPower(converter string, power float64) (float64, error) {
	if converter == "" {
		return 0, nil
	}
	if ecoDevice, ok := ecoflowDeviceMap[converter]; ok {
		services.ServerMessage("Set power of %s to %02f", converter, power)
		return ecoDevice.SetEcoflowPowerConsumption(power)
	}
	return 0, nil
}

func (g greeting) Converter() []string {
	converters := make([]string, 0)
	for _, device := range adapter.DevicesConfig.EnergySources {
		if device.Type == "ecoflow" {
			if device.MicroConverter != "" {
				converters = append(converters, device.MicroConverter)
			}
		}
	}
	return converters
}

func (g greeting) Register(config *energymonitor.AdapterConfig) {
	adapter = config
	InitEcoflow()
}

// exported

// Loader loader for initialize plugin
var Loader greeting

// EntryPoint entry point for main structure
var EntryPoint greeting
