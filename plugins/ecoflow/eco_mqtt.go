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
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/tknie/log"
	"github.com/tknie/services"
)

// insertMqttData prepare MQTT data into column data for database storage
func insertMqttData(prefix string, data map[string]interface{}) ([]string, [][]any) {
	keys := make([]string, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	columns := make([]any, 0)
	// prefix := ""
	fields := make([]string, 0)
	for _, k := range keys {
		v := data[k]
		name := prefix + "_" + strings.ReplaceAll(k, ".", "_")
		fields = append(fields, name)
		log.Log.Debugf(" %s=%v %T -> %s", k, v, v, name)
		switch val := v.(type) {
		case string:
			columns = append(columns, val)
		case bool:
			if val {
				columns = append(columns, byte(1))
			} else {
				columns = append(columns, byte(0))
			}
		case int64:
			columns = append(columns, val)
		case float64:
			if val == math.Trunc(val) {
				columns = append(columns, int64(val))
			} else {
				columns = append(columns, val)
			}
		case time.Time:
			columns = append(columns, val)
		case []interface{}, map[string]interface{}:
			b, err := json.Marshal(val)
			if err != nil {
				services.ServerMessage("Error marshal: %#v", val)
				columns = append(columns, nil)
			} else {
				s := string(b)
				columns = append(columns, s)
			}
		default:
			services.ServerMessage("Unknown type %s=%T", k, v)
			log.Log.Errorf("Unknown type %s=%T", k, v)
		}
	}
	return fields, [][]any{columns}
}
