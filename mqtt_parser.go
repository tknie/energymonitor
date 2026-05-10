/*
* Copyright 2023-2025 Thorsten A. Knieling
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
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	tlog "github.com/tknie/log"
	"github.com/tknie/services"
)

func (topic *Topic) createEntry(x map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	tlog.Log.Debugf("Create mapping entry by %#v", x)
	for _, e := range topic.Mapping {
		tlog.Log.Debugf("From source %s", e.Source)
		mNames := strings.Split(e.Source, "/")
		var i interface{}
		i = x
		for _, s := range mNames {
			tlog.Log.Debugf("Take %s", s)
			if subMap, ok := i.(map[string]interface{})[s]; ok {
				i = subMap
			} else {
				break
			}
		}
		tlog.Log.Debugf("Destination %s = %v (%s)", e.Destination, i, e.Type)
		// t := reflect.TypeOf(e.Type)
		f := reflectType(e.Type, i)
		switch v := f.(type) {
		case int64:
			if e.IfNegative != "" && v < 0 {
				m[e.IfNegative] = -v
				m[e.Destination] = int64(0)
			} else {
				if v, ok := m[e.Destination]; ok {
					tlog.Log.Debugf("Already set %s to %v", e.Destination, v)
					if v == nil {
						m[e.Destination] = f
					}
				} else {
					m[e.Destination] = f
				}
			}
		case float64:
			if e.IfNegative != "" && v < 0 {
				m[e.IfNegative] = -v
				m[e.Destination] = float64(0)
			} else {
				if v, ok := m[e.Destination]; ok {
					if v == nil {
						m[e.Destination] = f
					}
				} else {
					m[e.Destination] = f
				}
			}
		default:
			m[e.Destination] = f
		}
		tlog.Log.Debugf("Type %s -> %T %v", e.Type, f, f)
	}
	return m
}

func reflectType(fdType string, i interface{}) interface{} {
	var t reflect.Type
	switch fdType {
	case "int64":
		t = reflect.TypeOf(int64(0))
	case "float64":
		t = reflect.TypeOf(float64(0))
	case "string":
		t = reflect.TypeOf("")
	case "time.Time":
		t = reflect.TypeOf(time.Now())

	}
	o := reflect.New(t)
	o = o.Elem()
	tlog.Log.Debugf("Resolve %s destType=%v %T", fdType, i, i)
	switch fdType {
	case "time.Time":
		tn, err := time.ParseInLocation(Layout, i.(string), time.Local)
		if err != nil {
			log.Fatalf("Parse time location failed: %v", err)
		}
		v := reflect.ValueOf(tn)
		o.Set(v)
	// case "float64":
	// 	i64 := i.(int64)
	// 	fl64 := float64(i64)
	// 	v := reflect.ValueOf(fl64)
	// 	o.Set(v)
	case "int64":
		i64 := i.(float64)
		fl64 := int64(i64)
		v := reflect.ValueOf(fl64)
		o.Set(v)
	case "float64":
		switch i.(type) {
		case int64:
			i64 := i.(int64)
			fl64 := float64(i64)
			v := reflect.ValueOf(fl64)
			o.Set(v)
		case float64:
			i64 := i.(float64)
			fl64 := float64(i64)
			v := reflect.ValueOf(fl64)
			o.Set(v)
		case string:
			if fl64, err := strconv.ParseFloat(i.(string), 64); err == nil {
				v := reflect.ValueOf(fl64)
				o.Set(v)
			}
		default:
			log.Fatalf("Unknown type for float64 mapping: %T", i)
		}
	default:
		v := reflect.ValueOf(i)
		o.Set(v)
	}
	return o.Interface()
}

func (topic *Topic) ParseMessage(x map[string]interface{}) map[string]interface{} {
	em := topic.createEntry(x)
	if em != nil {
		tlog.Log.Debugf("Return dynamic %v", em)
		return em
	}
	services.ServerMessage("No dynamic parsing mapping")
	tlog.Log.Fatalf("Mapping not defined for topic: %s", topic.Name)
	return nil
}
