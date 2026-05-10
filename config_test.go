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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdapterLoad(t *testing.T) {
	LoadConfig("./adapter_test.yaml")
	assert.Equal(t, int64(180), adapter.DefaultConfig.BaseRequest)
	assert.Equal(t, "info", adapter.DefaultConfig.Debug)
	assert.Equal(t, "device_quota", adapter.DatabaseConfig.Table)
	assert.Equal(t, "test123", adapter.EcoflowConfig.Password)
	assert.Equal(t, []string([]string{"abc", "ddd", "${ECOFLOW_DEVICE_SN}"}),
		adapter.EcoflowConfig.MicroConverter)
}
