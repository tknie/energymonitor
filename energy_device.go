package energymonitor

import "github.com/tknie/services"

func refreshCurrentPowerRequest() {
	services.ServerMessage("Refresh current power request")
}
