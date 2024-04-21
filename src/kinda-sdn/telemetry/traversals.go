package telemetry

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

func runDfs(G model.TopologyGraph, source string, target string) (reversedPath []string) {
	visited := map[string]bool{}
	reversedPath = []string{}
	var dfs func(cur string)
	dfs = func(cur string) {
		visited[cur] = true
		if cur == target {
			reversedPath = append(reversedPath, cur)
			return
		}
		dev := G[cur]
		for _, neigh := range dev.GetLinks() {
			if !visited[neigh.To] {
				dfs(neigh.To)
				if len(reversedPath) > 0 {
					reversedPath = append(reversedPath, cur)
					return
				}
			}
		}
	}
	dfs(source)
	return
}

func (t *TelemetryService) findPortLeadingToDevice(G model.TopologyGraph, source model.Device, target model.Device) int {
	reversedPath := runDfs(G, source.GetName(), target.GetName())
	succesor := reversedPath[len(reversedPath) - 2]
	for i, link := range source.GetLinks() {
		if link.To == succesor {
			return i
		}
	}
	panic("no path from source to target")
}

func (t *TelemetryService) isCompatible(dev model.Device) bool {
	return dev.GetType() == model.INC_SWITCH && 
		   t.registry.Implements(dev.(*model.IncSwitch).InstalledProgram, PROGRAM_INTERFACE)
}

func (t *TelemetryService) findSourceSinkEdgesAndFillTransit(
	G model.TopologyGraph,
	sourceDevice string,
	targetDevice string,
	entities *TelemetryEntities,
) (reversedPath []string, sourcePathIdx int, sinkPathIdx int, foundIncDevicesOnPath bool) {
	foundIncDevicesOnPath = false
	if sourceDevice == targetDevice {
		return
	}
	reversedPath = runDfs(G, sourceDevice, targetDevice)
	if len(reversedPath) == 0 {
		panic("no path")
	}
	i := len(reversedPath) - 2
	for ; i >= 0; i-- {
		if t.isCompatible(G[reversedPath[i]]) {
			break
		}
	}
	if i < 0 {
		return
	}
	j := 1
	for ; j <= i; j++ {
		if t.isCompatible(G[reversedPath[j]]) {
			break
		}
	}
	if j == i {
		return // telemetry needs at least 2 devices to work properly
	}
	for k := j + 1; k < i; k++ {
		dev := G[reversedPath[k]]
		if t.isCompatible(dev) {
			entities.Transits[dev.GetName()] = struct{}{}
		}
	}
	foundIncDevicesOnPath = true
	sourcePathIdx = i
	sinkPathIdx = j
	return
}

func (t *TelemetryService) createTunneledTelemetryConfigs(
	sourceDetails *pbt.TunneledEntities,
	targetDetails *pbt.TunneledEntities,
) []TelemetrySourceConfig {
	res := []TelemetrySourceConfig{}
	for _, s := range sourceDetails.Entities {
		for _, t := range targetDetails.Entities {
			srcPort := ANY_PORT
			if s.Port != nil {
				srcPort = int(*s.Port)
			}
			dstPort := ANY_PORT
			if t.Port != nil {
				dstPort = int(*t.Port)
			}
			srcIp := ANY_IPv4
			if s.TunneledIp != ANY_IPv4 {
				srcIp = ExactIpv4Ternary(s.TunneledIp)
			}
			dstIp := ANY_IPv4
			if t.TunneledIp != ANY_IPv4 {
				dstIp = ExactIpv4Ternary(t.TunneledIp)
			}
			res = append(res, TelemetrySourceConfig{
				ips: TernaryIpPair{
					src: srcIp,
					dst: dstIp,
				},
				srcPort: srcPort,
				dstPort: dstPort,
				tunneled: true,
			})
		}
	}
	return res
}

func (t *TelemetryService) createRawTelemetryConfig(
	G model.TopologyGraph,
	sourceDetails *pbt.RawTelemetryEntity,
	targetDetails *pbt.RawTelemetryEntity,
	reversedPath []string,
) TelemetrySourceConfig {
	ipPair := TernaryIpPair{}
	source := sourceDetails.DeviceName
	target := targetDetails.DeviceName
	if G[source].GetType() == model.EXTERNAL {
		ipPair.src = ANY_IPv4
	} else {
		ipPair.src = ExactIpv4Ternary(G[source].MustGetLinkTo(reversedPath[len(reversedPath) - 2]).Ipv4)
	}
	if G[target].GetType() == model.EXTERNAL {
		ipPair.dst = ANY_IPv4
	} else {
		ipPair.dst = ExactIpv4Ternary(G[target].MustGetLinkTo(reversedPath[1]).Ipv4)
	}
	srcPort := ANY_PORT
	if sourceDetails.Port != nil {
		srcPort = int(*sourceDetails.Port)
	}
	dstPort := ANY_PORT
	if targetDetails.Port != nil {
		dstPort = int(*targetDetails.Port)
	}
	return TelemetrySourceConfig{
		ips: ipPair,
		srcPort: srcPort,
		dstPort: dstPort,
		tunneled: false,
	}
}

func (t *TelemetryService) findTelemetryEntitiesForRequest(req *pbt.ConfigureTelemetryRequest, G model.TopologyGraph) *TelemetryEntities {
	// TODO consider rewriting this with dynamic programming, it could decrease the complexity to linear
	res := newTelemetryEntities()

	for sourceIdx, source := range getSourceDeviceNames(req) {
		for targetIdx, target := range getTargetDeviceNames(req) {
			if source == target {
				continue
			}
			reversedPath, sourcePathIdx, sinkPathIdx, foundIncDeviceOnPath := t.findSourceSinkEdgesAndFillTransit(
				G, source, target, res) 	
			if !foundIncDeviceOnPath {
				continue
			}

			sinkEdge := Edge{from: reversedPath[sinkPathIdx], to: reversedPath[sinkPathIdx-1]}
			sourceEdge := Edge{from: reversedPath[sourcePathIdx], to: reversedPath[sourcePathIdx+1]}
			res.Sinks[sinkEdge] = struct{}{}

			sd := getSourceDetails(req, source, sourceIdx)
			td := getTargetDetails(req, target, targetIdx)
			sourceConf, ok := res.Sources[sourceEdge]
			if !ok {
				sourceConf = []TelemetrySourceConfig{}
			}

			if requiresTunneling(req) {
				newEntries := t.createTunneledTelemetryConfigs(sd.(*pbt.TunneledEntities), td.(*pbt.TunneledEntities))
				sourceConf = append(sourceConf, newEntries...)
			} else {
				newEntry := t.createRawTelemetryConfig(G, sd.(*pbt.RawTelemetryEntity), td.(*pbt.RawTelemetryEntity), reversedPath)	
				sourceConf = append(sourceConf, newEntry)
			}
			res.Sources[sourceEdge] = sourceConf
		}
	}

	// to have proper switch_ids we need to ensure that source/sink are configured as transits too
	for src := range res.Sources {
		res.Transits[src.from] = struct{}{}
	}
	for sink := range res.Sinks {
		res.Transits[sink.from] = struct{}{}
	}

	return res
}
