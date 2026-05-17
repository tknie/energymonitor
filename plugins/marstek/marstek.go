package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tknie/energymonitor"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/marstek"
	"github.com/tknie/services"
)

var httpDone = make(chan bool, 1)
var httpCounter = uint64(0)

var statusChange = make(map[string]bool)

func InitMastek() {
	go deviceValueStore()
}

// deviceValueStore main thread reading Marstek parameters and store it into database
func deviceValueStore() {

	clients := make([]struct {
		table     string
		converter string
		client    *marstek.Marstek
		id        common.RegDbID
	}, 0)

	for _, l := range adapter.DevicesConfig.EnergySources {
		if l.Type != "marstek" {
			continue
		}
		if len(l.Tables) == 0 {
			services.ServerMessage("No table defined for marstek plugin, check configuration")
			continue
		}
		// get all parameters for device
		services.ServerMessage("Query Matrix device values at %s", l.MicroConverter)
		client := marstek.New(l.MicroConverter)
		marstekMap, err := client.Summary()
		if err != nil {
			services.ServerMessage("Error getting device summary sn=%s: %v", l.MicroConverter, err)
			log.Log.Errorf("Error getting device summary sn=%s: %v", l.MicroConverter, err)
			continue
		}
		marstekMap["serial_number"] = l.MicroConverter
		marstekMap["timestamp"] = time.Now()
		table := l.Tables[0]
		id := energymonitor.ConnnectDatabase()
		clients = append(clients, struct {
			table     string
			converter string
			client    *marstek.Marstek
			id        common.RegDbID
		}{
			table:     table,
			converter: l.MicroConverter,
			client:    client,
			id:        id,
		})

		// Check, create and write into table
		if energymonitor.CheckTableExists(id, table, func() []*common.Column {
			keys := generateKeys("marstek", marstekMap)
			columns := make([]*common.Column, 0)
			// prefix := ""
			fmt.Println("Marstek check keys: ", keys)
			for _, k := range keys {
				v := evaluateValue(k, marstekMap)
				name := "marstek_" + strings.ReplaceAll(k, ".", "_")
				if k == "timestamp" {
					name = "timestamp"
				}
				log.Log.Debugf("Add column %s=%v %T -> %s", k, v, v, name)
				column := energymonitor.CreateValueColumn(name, v)
				columns = append(columns, column)
			}
			return columns
		}) {
			energymonitor.AddTableColumns(id, table, "marstek", marstekMap)
		}
	}

	if len(clients) == 0 {
		services.ServerMessage("No Marstek devices defined")
		return
	}
	services.ServerMessage("Init Marstek plugin data store")

	// Loop reading and writing data into table
	counter := uint64(0)
	services.ServerMessage("Marstek API loop is started %d seconds interval", energymonitor.LoopSeconds)

	connectSuccess := uint64(0)
	connectFail := uint64(0)

	for {
		counter++
		if counter%350 == 0 {
			services.ServerMessage("Marstek requests: %04d (success=%04d,fail=%04d)",
				counter, connectSuccess, connectFail)
		}

		select {
		case <-httpDone:
			services.ServerMessage("Ecoflow API loop is stopped")

			return
		case <-time.After(time.Second * time.Duration(energymonitor.LoopSeconds)):
			if counter%350 == 0 {
				services.ServerMessage("Send HTTP requests: %04d", counter)
			}

			for _, l := range clients {
				log.Log.Debugf("Connect device at: %s", l.converter)
				client := marstek.New(l.converter)
				resp, err := client.Summary()
				if err != nil {
					connectFail++
					log.Log.Errorf("Error getting device list %s: %v", l.converter, err)
					services.ServerMessage("Error getting device list %s: %v", l.converter, err)
					statusChange[l.converter] = false
				} else {
					connectSuccess++
					log.Log.Debugf("Checking missing table fields for : %s", l.converter)
					if _, ok := resp["serial_number"]; !ok {
						resp["serial_number"] = l.converter
					}
					if _, ok := resp["timestamp"]; !ok {
						resp["timestamp"] = time.Now()
					}
					energymonitor.AddTableColumns(l.id, l.table, "marstek", resp)
					err = energymonitor.InsertTable(l.id, l.table, "marstek", resp, energymonitor.InsertDeapData)
					if err != nil && strings.Contains(err.Error(), "conn closed") {
						l.id.Close()
						l.id = energymonitor.ConnnectDatabase()
					}
					httpCounter++
					status, ok := statusChange[l.converter]
					if !ok {
						statusChange[l.converter] = true
						status = false
					} else {
						if !status {
							if status && !ok {
								services.ServerMessage("'%s' device is getting offline", l.converter)
							}
							statusChange[l.converter] = false
						} else {
							statusChange[l.converter] = true
							if !status {
								services.ServerMessage("'%s' device is getting online", l.converter)
							}
						}
					}
				}
			}
		}
	}
}

func evaluateValue(key string, marstekMap map[string]interface{}) interface{} {
	keys := strings.Split(key, "_")
	currentMap := marstekMap
	for i, k := range keys {
		v, ok := currentMap[k]
		if !ok {
			log.Log.Errorf("Key %s not found in map", k)
			return nil
		}
		switch val := v.(type) {
		case map[string]interface{}:
			currentMap = val
			continue
		default:
			if i == len(keys)-1 {
				return v
			}
			log.Log.Errorf("Key %s not found in map", k)
			return nil
		}
	}
	return nil
}

func generateKeys(prefix string, marstekMap map[string]interface{}) []string {
	keys := evaluateKeys(prefix, marstekMap)
	sort.Strings(keys)
	return keys
}

func evaluateKeys(prefix string, marstekMap map[string]interface{}) []string {
	keys := make([]string, 0)
	for k, v := range marstekMap {
		switch val := v.(type) {
		case map[string]interface{}:
			prefix := prefix + "_" + k
			subkeys := evaluateKeys(prefix, val)
			keys = append(keys, subkeys...)
		default:
			log.Log.Debugf("Key %s=%v %T", k, v, v)
			keys = append(keys, prefix+"_"+k)
		}
	}
	return keys
}
