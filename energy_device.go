package energymonitor

import (
	"github.com/tknie/log"
	"github.com/tknie/services"
)

func SetOverallPowerConsumption(newRequested float64) {
	if adapter.DefaultConfig.RealtimeRequest {
		realtimeRequest := newRequested
		if realtimeRequest < float64(adapter.DefaultConfig.BaseRequest) {
			realtimeRequest = float64(adapter.DefaultConfig.BaseRequest)
		}
		if realtimeRequest > float64(adapter.DefaultConfig.UpperBatLimit) {
			realtimeRequest = float64(adapter.DefaultConfig.UpperBatLimit)
		}
		// TODO use different microconverter or modules to set power request
		//converter := EcoflowMicroConverter()
		// client.SetEnvironmentPowerConsumption(converter, newRequested)
		services.ServerMessage("Set overall power consumption to %f", realtimeRequest)
		for _, p := range plugins {
			for _, c := range p.Loader.Converter() {
				if realtimeRequest > 0 {
					services.ServerMessage("Set rest power to %f", realtimeRequest)
					r, err := p.Loader.SetPower(c, realtimeRequest)
					if err != nil {
						log.Log.Errorf("Error setting power: %v", err)
					} else {
						realtimeRequest -= r
					}
					log.Log.Debugf("Device: %s set power of %d", c, r)
				}
			}
		}
	}
}

func refreshCurrentPowerRequest() {
	currentRequested = 0
	for _, p := range plugins {
		for _, c := range p.Loader.Converter() {
			v, err := p.Loader.GetPower(c)
			if err != nil {
				log.Log.Errorf("Error getting power: %v", err)
			} else {
				currentRequested += v[0]
				currentDelivered += v[1]
			}
			log.Log.Debugf("Device: %s", c)
		}
	}
	log.Log.Debugf("Final requested: %f", currentRequested)
}
