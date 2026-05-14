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

package energymonitor

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/tknie/log"
	"github.com/tknie/services"
)

// PluginTypes different types of plugins for
// - auditing
// - database operation
type PluginTypes int

const (
	// NoPlugin no plugin but may be used in module
	NoPlugin PluginTypes = iota
	// DevicePlugin device plugin for handling device-specific operations
	DevicePlugin
)

const suffix = ".so"

var plugins = make(map[string]*PluginLoader)

// PluginInfo metdata of the plugin
type PluginInfo struct {
	Name         string
	Version      string
	Types        []PluginTypes
	Identifier   []string
	AbortOnError bool
}

// Loader plugin Loader module to load plugin features
type Loader interface {
	Name() string
	Version() string
	Register(config *AdapterConfig)
	Info() *PluginInfo
	GetPower(converter string) ([]float64, error)
	SetPower(converter string, power float64) (float64, error)
	Converter() []string
	Stop()
}

// Executor auditing method to send to plugin
type Executor interface {
	// LoginAudit(string, string, *auth.SessionInfo, *auth.UserInfo) error
}

// AuditLoader auditing loader structure
type PluginLoader struct {
	Loader   Loader
	Executor Executor
}

var shutOnce = new(sync.Once)

// InitPlugins initialize plugins in given plugin directory
func InitPlugins() {
	services.ServerMessage("Initialize plugins")
	pluginDir, ok := os.LookupEnv("ENERGY_METER_PLUGINS")
	if !ok {
		pluginDir = "./plugins"
	}
	pluginDir = os.ExpandEnv(pluginDir)
	if pluginDir == "" {
		return
	}
	pluginEnabled, filterPlugins := os.LookupEnv("ENERGY_METER_PLUGENABLED")
	var plugList []string
	if filterPlugins {
		plugList = strings.Split(pluginEnabled, ",")
	}
	services.ServerMessage("Searching plugins in %s", pluginDir)
	err := filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		log.Log.Debugf("Path: %s, Info: %s", path, info.Name())
		if !info.IsDir() && strings.HasSuffix(info.Name(), suffix) {
			plug, err := loadPlugin(pluginDir + "/" + info.Name())
			if err != nil {
				return nil
			}
			symLanguage, err := plug.Lookup("Loader")
			if err != nil {
				services.ServerMessage("Error resolve plugin methods: %v", err)
				return nil
			}
			if loader, ok := symLanguage.(Loader); ok {
				found := !filterPlugins
				if !found && plugList != nil {
					n := loader.Name()
					for _, v := range plugList {
						if n == v {
							found = true
							break
						}
					}
				}
				if found {
					load(loader, info, plug)
				}
			} else {
				services.ServerMessage("Error opening plugin, error loading methods")
			}
		}
		return nil
	})
	if err != nil {
		return
	}

}

// load loading the plugin
func load(loader Loader, info os.FileInfo, plug *plugin.Plugin) {
	pi := loader.Info()
	services.ServerMessage("Loading plugin: %s, version: %s, type: %s", pi.Name, pi.Version, pi.Identifier[0])
	loader.Register(adapter)
	plugins[pi.Identifier[0]] = &PluginLoader{Loader: loader}
}

// ShutdownPlugins shutdown receiving message in plugins
func ShutdownPlugins() {
	shutOnce.Do(func() {
		services.ServerMessage("Shutdown all plugins ...")

		for _, v := range plugins {
			v.Loader.Stop()
		}
	})
}

func loadPlugin(mod string) (*plugin.Plugin, error) {
	// load module
	// 1. open the so file to load the symbols
	plug, err := plugin.Open(mod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return plug, nil
}
