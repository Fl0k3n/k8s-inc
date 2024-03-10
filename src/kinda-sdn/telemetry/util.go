package telemetry

import "github.com/Fl0k3n/k8s-inc/kinda-sdn/model"

type Edge struct {
	from string
	to string
}

type TernaryIpPair struct {
	src string
	dst string 
}

type TelemetryEntities struct {
	Sources map[Edge][]TernaryIpPair
	Transits map[string]struct{}
	Sinks map[Edge]struct{}
}


// TODO, they are needed for telemetry bookkeeping (mapping raport headers to switches)
func getSwitchIds(topo *model.Topology) map[string]int {
	res := map[string]int{}
	counter := 0
	for _, dev := range topo.Devices {
		if dev.GetType() == model.INC_SWITCH {
			res[dev.GetName()] = counter
			counter++
		}
	}
	return res
}
