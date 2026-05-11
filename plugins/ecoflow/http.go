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
	"sort"
	"strings"
	"time"

	"github.com/tknie/ecoflow"
	"github.com/tknie/energymonitor"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

// readEcoflowAndStoreDB main thread reading information with HTTP request
// and store them in the database
func readEcoflowAndStoreDB() {
	services.ServerMessage("Init HTTP Ecoflow parameter store to database loop")
	client = prepareEcoflow()
	if client == nil {
		services.ServerMessage("Ecoflow client is not initialized, cannot start HTTP parameter store")
		return
	}
	id := energymonitor.ConnnectDatabase()
	devices := client.GetDevices()
	if devices == nil {
		log.Log.Fatal("Devices empty")
	}

	table := getEcoflowTable()
	if table == "" {
		services.ServerMessage("No table defined for ecoflow plugin, check configuration")
		return
	}
	services.ServerMessage("Go through %d devices to get parameters and store to %s", len(devices.Devices), table)
	for _, l := range devices.Devices {
		// get all parameters for device
		// services.ServerMessage("Get Parameter for : %s", l.SN)
		resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
		if err != nil {
			services.ServerMessage("Error getting device parameter sn=%s: %v", l.SN, err)
			log.Log.Errorf("Error getting device parameter sn=%s: %v", l.SN, err)
			continue
		}
		// Check, create and write into table
		energymonitor.CheckTableExists(id, table, func() []*common.Column {
			keys := make([]string, 0, len(resp))
			for k := range resp {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			columns := make([]*common.Column, 0)
			// prefix := ""
			for _, k := range keys {
				v := resp[k]
				if k == "timestamp" {
					column := energymonitor.CreateValueColumn(k, v)
					columns = append(columns, column)
				}
				// prefix = strings.Split(k, ".")[0]
				// name := "eco_" + strings.ReplaceAll(k[len(prefix)+1:], ".", "_")
				name := "eco_" + strings.ReplaceAll(k, ".", "_")
				log.Log.Debugf("Add column %s=%v %T -> %s", k, v, v, name)
				column := energymonitor.CreateValueColumn(name, v)
				columns = append(columns, column)
			}
			return columns
		})
	}

	loopEcoflowStoreToDb(id, devices, table)
}

func loopEcoflowStoreToDb(id common.RegDbID, devices *ecoflow.DeviceListResponse, table string) {

	// Loop reading and writing data into table
	counter := uint64(0)
	needRefresh := false

	services.ServerMessage("Ecoflow API loop is started %d seconds interval", energymonitor.LoopSeconds)

	for {
		counter++
		select {
		case <-httpDone:
			services.ServerMessage("Ecoflow API loop is stopped")

			return
		case <-time.After(time.Second * time.Duration(energymonitor.LoopSeconds)):

			for _, l := range devices.Devices {
				if counter%350 == 0 {
					services.ServerMessage("Send HTTP requests: %04d", counter)
				}
				resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
				if err != nil {
					log.Log.Errorf("Error getting device list %s: %v", l.SN, err)
					services.ServerMessage("Error getting device list %s: %v", l.SN, err)
				} else {
					//services.ServerMessage("Get Parameter for : %s", l.SN)
					if _, ok := resp["serial_number"]; !ok {
						resp["serial_number"] = l.SN
					}
					if _, ok := resp["timestamp"]; !ok {
						resp["timestamp"] = time.Now()
					}
					energymonitor.AddTableColumns(id, table, "eco", resp)
					err = energymonitor.InsertTable(id, table, "eco", resp, energymonitor.InsertHttpData)
					if err != nil && strings.Contains(err.Error(), "conn closed") {
						id.Close()
						id = energymonitor.ConnnectDatabase()
					}
					httpCounter++
					status, ok := statusChange[l.SN]
					if !ok {
						statusChange[l.SN] = l.Online == 1
						status = false
					}
					if l.Online != 1 {
						if status && !ok {
							services.ServerMessage("'%s' device is getting offline", l.SN)
						}
						statusChange[l.SN] = false
						needRefresh = true
					} else {
						statusChange[l.SN] = true
						if !status {
							services.ServerMessage("'%s' device is getting online", l.SN)
						}
					}
				}
			}
		}
		log.Log.Infof("Triggered %d. HTTP query at %s", counter, time.Now().Format(energymonitor.Layout))
		if needRefresh {
			client.RefreshDeviceList()
		}
	}
}

const DefaultSeconds = 60

var httpDone = make(chan bool, 1)
var httpCounter = uint64(0)

var statusChange = make(map[string]bool)

func init() {
	ecoflow.Callback = Callback
}

func checkDeviceOnline(sn string) bool {
	if b, ok := statusChange[sn]; ok {
		return b
	}
	return false
}

func Callback(serialNumber string, data map[string]interface{}) {
	tn := fmt.Sprintf("%s_mqtt", serialNumber)
	if !energymonitor.CheckTableExists(mqttid, tn, func() []*common.Column {
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		columns := make([]*common.Column, 0)
		// prefix := ""
		for _, k := range keys {
			v := data[k]
			name := "eco_" + strings.ReplaceAll(k, ".", "_")
			log.Log.Debugf("Add column %s=%v %T -> %s", k, v, v, name)
			column := energymonitor.CreateValueColumn(name, v)
			columns = append(columns, column)
		}
		return columns
	}) {
		energymonitor.AddTableColumns(mqttid, tn, "eco", data)
	}
	err := energymonitor.InsertTable(mqttid, tn, "eco", data, insertMqttData)
	if err != nil && strings.Contains(err.Error(), "conn closed") {
		// Connection is closed reconnect
		mqttid.Close()
		mqttid = energymonitor.ConnnectDatabase()
	}
}

func getEcoflowTable() string {
	for _, x := range adapter.DevicesConfig.EnergySources {
		if x.Type == "ecoflow" {
			return x.Tables[1]
		}
	}
	return ""
}
