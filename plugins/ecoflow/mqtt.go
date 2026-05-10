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
	"github.com/tknie/ecoflow"
	"github.com/tknie/energymonitor"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var mqttid common.RegDbID
var client *ecoflow.Client

// InitMqtt initialize Ecoflow MQTT listener
func InitMqtt(user, password string) {
	mqttid = energymonitor.ConnnectDatabase()
	services.ServerMessage("Connecting MQTT client")
	ecoflow.InitMqtt(user, password)
	log.Log.Debugf("Wait for Ecoflow disconnect")
}
