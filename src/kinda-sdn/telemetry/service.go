package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/device"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/programs"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

type DeviceProvider = func(model.DeviceName) device.IncSwitch

const TELEMETRY_MTU = 1500

const PROGRAM_INTERFACE = "inc.kntp.com/v1alpha1/telemetry"

type TelemetryService struct {
	entityMapLock sync.Mutex
	entityLocks map[string]*sync.Mutex
	registry programs.P4ProgramRegistry
	transitCounters sync.Map // key = deviceName, val = counter of transit requests
}

func NewService(registry programs.P4ProgramRegistry) *TelemetryService {
	return &TelemetryService{
		entityMapLock: sync.Mutex{},
		entityLocks: make(map[string]*sync.Mutex),
		registry: registry,
		transitCounters: sync.Map{},
	}
}

func (t *TelemetryService) GetArpEntry(reqIp string, respMac string) connector.RawTableEntry {
	return Arp(reqIp, respMac)
}
func (t *TelemetryService) GetForwardEntry(maskedIp string, srcMac string, dstMac string, port int) connector.RawTableEntry {
	return Forward(maskedIp, srcMac, dstMac, fmt.Sprintf("%d", port))
}
func (t *TelemetryService) GetDefaultRouteEntry(srcMac string, dstMac string, port int) connector.RawTableEntry {
	return DefaultRoute(srcMac, dstMac, fmt.Sprintf("%d", port))
}

func (t *TelemetryService) InitDevices(ctx context.Context, topo *model.Topology, deviceProvider DeviceProvider) error {
	// mark all inc switches as transit as it doesn't hurt us
	// this assumes that topo and programs don't change
	// TODO don't do this like this

	switchIds := getSwitchIds(topo)
	// for _, dev := range topo.Devices {
	// 	if dev.GetType() == model.INC_SWITCH && dev.(*model.IncSwitch).InstalledProgram == "telemetry" {
	// 		swId := switchIds[dev.GetName()]
	// 		dev := deviceProvider(dev.GetName())
	// 		if err := dev.WriteEntry(ctx, Transit(swId, TELEMETRY_MTU)); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	for switchName := range switchIds {
		t.transitCounters.Store(switchName, 0)
	}

	return nil
}

func (t *TelemetryService) EnableTelemetry(
	req *pbt.EnableTelemetryRequest,
	topo *model.Topology,
	deviceProvider DeviceProvider,
) (*pbt.EnableTelemetryResponse, error) {
	// TODO use transacion log and log every update, in case of error perform rollback
	// TODO also consider rewriting this with dynamic programming, it could decrease the complexity to linear
	t.entityMapLock.Lock()
	if mapLock, ok := t.entityLocks[req.CollectionId]; ok {
		locked := mapLock.TryLock()
		if !locked {
			description := "Already handling collection"
			t.entityMapLock.Unlock()
			return &pbt.EnableTelemetryResponse{
				TelemetryState: pbt.TelemetryState_IN_PROGRESS,
				Description: &description,
			}, nil
		}
	} else {
		lock := &sync.Mutex{}
		lock.Lock()
		t.entityLocks[req.CollectionId] = lock
	}
	t.entityMapLock.Unlock()
	defer func() {
		t.entityMapLock.Lock()
		defer t.entityMapLock.Unlock()
		t.entityLocks[req.CollectionId].Unlock()
	}()

	// TODO store state and remove not needed entries 

	G := model.TopologyToGraph(topo)
	desiredEntities := t.findTelemetryEntitiesForRequest(req, G)

	if len(desiredEntities.Transits) > 0 {
		switchIds := getSwitchIds(topo)
		for transit := range desiredEntities.Transits {
			for {
				counter, ok := t.transitCounters.Load(transit)
				if !ok {
					panic("Unitialized counter")
				}
				if ok := t.transitCounters.CompareAndSwap(transit, counter, counter.(int) + 1); ok {
					if counter.(int) == 0 {
						ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
						if err := deviceProvider(transit).WriteEntry(ctx, Transit(switchIds[transit], TELEMETRY_MTU)); err != nil {
							cancel()
							return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
						}
						cancel()
					}
					break
				}
			}
		}
	}

	// TODO conflicting sinks (?)
	for sink := range desiredEntities.Sinks {
		deviceMeta := G[sink.from].(*model.IncSwitch)
		device := deviceProvider(sink.from)
		portNumber, ok := deviceMeta.GetPortNumberTo(sink.to)
		if !ok {
			panic("no such neighbor")
		}
		collector := G[req.CollectorNodeName]
		portToCollector := t.findPortLeadingToDevice(G, G[sink.from], collector)

		if err := t.writeEntryIgnoringDuplicateError(device, ConfigureSink(portNumber + 1, portToCollector + 1)); err != nil {
			return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
		}

		nextHopToCollector := G[deviceMeta.GetLinks()[portToCollector].To]
		nextHopToCollectorMac := nextHopToCollector.MustGetLinkTo(deviceMeta.Name).MacAddr

		portToSink := t.findPortLeadingToDevice(G, collector, G[sink.from])
		if err := t.writeEntryIgnoringDuplicateError(device, Reporting(
			deviceMeta.GetLinks()[portToCollector].MacAddr, deviceMeta.GetLinks()[portToCollector].Ipv4,
			nextHopToCollectorMac, collector.GetLinks()[portToSink].Ipv4, int(req.CollectorPort),
		)); err != nil {
			return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
		}
	}

	for source, configEntries := range desiredEntities.Sources {
		deviceMeta := G[source.from].(*model.IncSwitch)
		device := deviceProvider(source.from)
		portToActivate, ok := deviceMeta.GetPortNumberTo(source.to)
		if !ok {
			panic("no such neighbor")
		}
		if err := t.writeEntryIgnoringDuplicateError(device, ActivateSource(portToActivate + 1)); err != nil {
			return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
		}
		for _, config := range configEntries {
			if err := t.writeEntryIgnoringDuplicateError(device, ConfigureSource(
				config.ips.src, config.ips.dst, config.srcPort, config.dstPort,
				4, 10, 8, FULL_TELEMETRY_KEY, config.tunneled,
			)); err != nil {
				return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
			}
		}
	}

	return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_OK}, nil
}

func (t *TelemetryService) writeEntryIgnoringDuplicateError(device device.IncSwitch, entry connector.RawTableEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
	defer cancel()
	if err := device.WriteEntry(ctx, entry); err != nil {
		if connector.IsEntryExistsError(err) {
			fmt.Printf("Entry already exists, table = %s\n", entry.TableName)
			return nil
		}
		fmt.Printf("Failed to write entry %s\n%e", entry, err)
		return err
	}
	return nil
}

func (t *TelemetryService) findPortLeadingToDevice(G model.TopologyGraph, source model.Device, target model.Device) int {
	visited := map[string]bool{}
	reversedPath := []string{}
	dfs(G, target.GetName(), source.GetName(), &reversedPath, visited)
	succesor := reversedPath[len(reversedPath) - 2]
	for i, link := range source.GetLinks() {
		if link.To == succesor {
			return i
		}
	}
	panic("no path from source to target")
}

func dfs(G model.TopologyGraph, target string, cur string, reversedPath *[]string, visited map[string]bool) {
	visited[cur] = true
	if cur == target {
		*reversedPath = append(*reversedPath, cur)
		return
	}
	dev := G[cur]
	for _, neigh := range dev.GetLinks() {
		if !visited[neigh.To] {
			dfs(G, target, neigh.To, reversedPath, visited)
			if len(*reversedPath) > 0 {
				*reversedPath = append(*reversedPath, cur)
				return
			}
		}
	}
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
) (reversedPath []string, sourcePathIdx int, sinkPathIdx int, foundIncDeviceOnPath bool) {
	foundIncDeviceOnPath = false
	if sourceDevice == targetDevice {
		return
	}
	visited := map[string]bool{}
	reversedPath = []string{}
	dfs(G, targetDevice, sourceDevice, &reversedPath, visited)
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
	for k := j + 1; k < i; k++ {
		dev := G[reversedPath[k]]
		if t.isCompatible(dev) {
			entities.Transits[dev.GetName()] = struct{}{}
		}
	}
	foundIncDeviceOnPath = true
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

func (t *TelemetryService) findTelemetryEntitiesForRequest(req *pbt.EnableTelemetryRequest, G model.TopologyGraph) TelemetryEntities {
	res := TelemetryEntities{
		Sources: map[Edge][]TelemetrySourceConfig{},
		Transits: map[string]struct{}{},
		Sinks: map[Edge]struct{}{},
	}

	for sourceIdx, source := range getSourceDeviceNames(req) {
		for targetIdx, target := range getTargetDeviceNames(req) {
			if source == target {
				continue
			}
			reversedPath, sourcePathIdx, sinkPathIdx, foundIncDeviceOnPath := t.findSourceSinkEdgesAndFillTransit(
				G, source, target, &res) 	
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
