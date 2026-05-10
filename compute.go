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
	"time"
)

const DefaultIntermediateSize = 15

type parameter struct {
	timestamp  time.Time
	solargen   int64
	batinput   float64
	batout     float64
	housein    int64
	gridwatts  float64
	requested  int64
	batreqfill int64
	powercurr  int32
	powerout   int32
	batfill    int64
}

func (p *parameter) toString() string {
	return fmt.Sprintf("%15v %10v %10v %10v %10v %10v %10v %10v %10v %10v %10v",
		p.timestamp.Format(shortLayout),
		p.solargen,
		p.batinput,
		p.batout,
		p.housein,
		p.gridwatts,
		p.requested,
		p.batreqfill,
		p.powercurr,
		p.powerout,
		p.batfill)
}
