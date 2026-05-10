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
	"context"
	"fmt"
	"os"

	"github.com/tknie/ecoflow"
	"github.com/tknie/energymonitor"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var currentRequested float64 = 0

const ECOFLOW_SELECT_GET_ALL_PARAMETER = `
with battery as (
select
	to_char(dq.eco_timestamp at TIME zone 'GMT', 'YYYYMMDD HH24MI') as timest,
	row_number() over (partition by to_char(dq.eco_timestamp at TIME zone 'GMT' , 'YYYYMMDD HH24MI')) as rn,
	eco_bms_bmsstatus_actsoc as batfill
from
	{{ .EcoflowTable }} dq
where
	dq.eco_serial_number = upper('{{ .BatteryConverterSerialNumber }}')
	and eco_timestamp at TIME zone 'GMT' >= NOW() - '30 minute'::interval
order by
	dq.eco_timestamp desc
),
adapter as (
select
	dq.eco_timestamp at TIME zone 'GMT',
	to_char(dq.eco_timestamp at TIME zone 'GMT', 'YYYYMMDD HH24MI') as timest,
	row_number() over (partition by to_char(dq.eco_timestamp at TIME zone 'GMT', 'YYYYMMDD HH24MI')) as rn,
		(eco_20_1_pv1inputwatts + eco_20_1_pv2inputwatts)/ 10 as "solargen" ,
	abs(least( eco_20_1_batinputwatts::float / 10, 0)) as "batinput",
	abs(greatest( eco_20_1_batinputwatts::float / 10, 0)) as "batout",
	eco_20_1_genewatt / 10 as "housein",
	abs(eco_20_1_gridconswatts::float / 10) as "gridwatts",
	eco_20_1_invdemandwatts / 10 as "requested",
	eco_20_1_bmsreqchgamp as "batreqfill"
from
	{{ .EcoflowTable }} dq
where
	dq.eco_serial_number = upper('{{ .ConverterSerialNumber }}')
		and dq.eco_timestamp at TIME zone 'GMT' >= NOW() - '30 minute'::interval
	order by
		dq.eco_timestamp desc
)
select
		h.inserted_on ,
	b.batfill,
	h.powercurr,
	h.powerout,
	a.solargen,
	a.batinput ,
	a.batout ,
	a.housein ,
	a.gridwatts,
	a.requested ,
	a.batreqfill
from
	battery b
inner join {{ .EnergyTable }} h on
	b.timest = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
inner join adapter a on
	a.timest = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
where
	b.rn = 1
	and a.rn = 1
order by
	b.timest desc,
	h.inserted_on desc
`

// InitEcoflow init ecoflow MQTT
func InitEcoflow() {
	if prepareEcoflow() == nil {
		return
	}
	adapter.ConnectMQTT(energymonitor.LoopCounterAndCancelOutput)
	user := adapter.EcoflowConfig.User
	password := adapter.EcoflowConfig.Password
	// Start statistics output
	go httpParameterStore()

	//done := make(chan bool, 1)
	if !energymonitor.MqttDisable {
		InitMqtt(user, password)
	}
	services.ServerMessage("Ecoflow plugin initialized")
	//<-done
}

func prepareEcoflow() *ecoflow.Client {

	accessToken := os.ExpandEnv(adapter.EcoflowConfig.AccessKey)
	secretToken := os.ExpandEnv(adapter.EcoflowConfig.SecretKey)
	if accessToken == "" || secretToken == "" {
		log.Log.Errorf("No access key or secret key given for ecoflow plugin")
		return nil
	}

	log.Log.Debugf("AccessKey: %v", accessToken)
	log.Log.Debugf("SecretKey: %v", secretToken)
	client := ecoflow.NewClient(accessToken, secretToken)
	client.RefreshDeviceList()
	return client
}

func ecoflowCurrentPowerRequest(converter string) []float64 {
	accessKey := os.ExpandEnv(adapter.EcoflowConfig.AccessKey)
	secretKey := os.ExpandEnv(adapter.EcoflowConfig.SecretKey)
	if accessKey == "" {
		accessKey = os.ExpandEnv("${ECOFLOW_ACCESS_KEY}")
	}
	if secretKey == "" {
		secretKey = os.ExpandEnv("${ECOFLOW_SECRET_KEY}")
	}
	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client := ecoflow.NewClient(accessKey, secretKey)
	ctx := context.Background()
	dsn, err := client.GetDeviceInfo(ctx, converter, "")
	if err != nil {
		fmt.Println("Error getting device info for converter: ", converter, " error: ", err)
		return []float64{0, 0}
	}
	converterRequested := dsn["20_1.invDemandWatts"].(float64) / 10
	energyProviding := dsn["20_1.invToOtherWatts"].(float64) / 10
	if converterRequested != currentRequested {
		services.ServerMessage("Update accu energy requested: %.1f before was %.1f", converterRequested, currentRequested)
		currentRequested = converterRequested
	}
	return []float64{currentRequested, energyProviding}
}

func EcoflowMicroConverter() string {
	for _, energySource := range adapter.DevicesConfig.EnergySources {
		if energySource.Type == "ecoflow" {
			return os.ExpandEnv(energySource.MicroConverter)
		}
	}
	return ""
}

func SetEcoflowPowerConsumption(microConverter string, value float64) (float64, error) {
	client := prepareEcoflow()
	client.SetEnvironmentPowerConsumption(microConverter, value)
	return value, nil
}
