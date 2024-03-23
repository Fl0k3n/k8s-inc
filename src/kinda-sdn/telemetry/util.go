package telemetry

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

type Edge struct {
	from string
	to string
}

type TernaryIpPair struct {
	src string
	dst string 
}

type TelemetrySourceConfig struct {
	ips TernaryIpPair
	srcPort int
	dstPort int
	tunneled bool
}

type TelemetryEntities struct {
	Sources map[Edge][]TelemetrySourceConfig
	Transits map[string]struct{}
	Sinks map[Edge]struct{}
}

func getSourceDeviceNames(req *pbt.EnableTelemetryRequest) []string {
	res := []string{}
	switch s := req.Sources.(type) {
	case *pbt.EnableTelemetryRequest_RawSources:
		for _, x := range s.RawSources.Entities {
			res = append(res, x.DeviceName)
		}
	case *pbt.EnableTelemetryRequest_TunneledSources:
		for devName := range s.TunneledSources.DeviceNamesWithEntities {
			res = append(res, devName)
		}
	}
	return res
}

func getTargetDeviceNames(req *pbt.EnableTelemetryRequest) []string {
	res := []string{}
	switch t := req.Targets.(type) {
	case *pbt.EnableTelemetryRequest_RawTargets:
		for _, x := range t.RawTargets.Entities {
			res = append(res, x.DeviceName)
		}
	case *pbt.EnableTelemetryRequest_TunneledTargets:
		for devName := range t.TunneledTargets.DeviceNamesWithEntities {
			res = append(res, devName)
		}
	}
	return res
}

func requiresTunneling(req *pbt.EnableTelemetryRequest) bool {
	_, isTunneled := req.Sources.(*pbt.EnableTelemetryRequest_TunneledSources)
	return isTunneled
}

type EntityDetails interface {}

func getSourceDetails(req *pbt.EnableTelemetryRequest, deviceName string, sourceIdx int) EntityDetails {
	if raw, ok := req.Sources.(*pbt.EnableTelemetryRequest_RawSources); ok {
		return raw.RawSources.Entities[sourceIdx]
	} else {
		tun := req.Sources.(*pbt.EnableTelemetryRequest_TunneledSources)
		return tun.TunneledSources.DeviceNamesWithEntities[deviceName]
	}
}

func getTargetDetails(req *pbt.EnableTelemetryRequest, deviceName string, targetIdx int) EntityDetails {
	if raw, ok := req.Targets.(*pbt.EnableTelemetryRequest_RawTargets); ok {
		return raw.RawTargets.Entities[targetIdx]
	} else {
		tun := req.Targets.(*pbt.EnableTelemetryRequest_TunneledTargets)
		return tun.TunneledTargets.DeviceNamesWithEntities[deviceName]
	}
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