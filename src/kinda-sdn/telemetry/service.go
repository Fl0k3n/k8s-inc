package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/device"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

type DeviceProvider = func(model.DeviceName) device.IncSwitch

const TELEMETRY_MTU = 1500

type TelemetryService struct {
	entityMapLock sync.Mutex
	entityLocks map[string]*sync.Mutex
}

func NewTelemetryService() *TelemetryService {
	return &TelemetryService{
		entityMapLock: sync.Mutex{},
		entityLocks: make(map[string]*sync.Mutex),
	}
}

func (t *TelemetryService) InitDevices(ctx context.Context, topo *model.Topology, deviceProvider DeviceProvider) error {
	// mark all inc switches as transit as it doesn't hurt us
	// this assumes that topo and programs don't change
	// TODO don't do this like this

	switchIds := getSwitchIds(topo)
	for _, dev := range topo.Devices {
		if dev.GetType() == model.INC_SWITCH && dev.(*model.IncSwitch).InstalledProgram == "telemetry" {
			swId := switchIds[dev.GetName()]
			dev := deviceProvider(dev.GetName())
			if err := dev.WriteEntry(ctx, Transit(swId, TELEMETRY_MTU)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *TelemetryService) EnableTelemetry(
	req *pbt.EnableTelemetryRequest,
	topo *model.Topology,
	deviceProvider DeviceProvider,
) (*pbt.EnableTelemetryResponse, error) {
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

	G := model.TopologyToGraph(topo)
	desiredEntities := t.findTelemetryEntitiesForRequest(req, G)
	// transits are already handled so ignore them, otherwise keep transit counters and if
	// counter transitions from 0 to n then configure transit and if it drops from 1 to 0 delete it


	// TODO conflicting sinks
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

	for source, ipPairs := range desiredEntities.Sources {
		deviceMeta := G[source.from].(*model.IncSwitch)
		device := deviceProvider(source.from)
		portToActivate, ok := deviceMeta.GetPortNumberTo(source.to)
		if !ok {
			panic("no such neighbor")
		}
		if err := t.writeEntryIgnoringDuplicateError(device, ActivateSource(portToActivate + 1)); err != nil {
			return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
		}
		// TODO ports
		for _, ipPair := range ipPairs {
			if err := t.writeEntryIgnoringDuplicateError(device, ConfigureSource(
				ipPair.src, ipPair.dst, 4607, 8959, 4, 10, 8, 0xFF00,
			)); err != nil {
				return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
			}
		}
	}

	return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_OK}, nil
}

func (t *TelemetryService) writeEntryIgnoringDuplicateError(device device.IncSwitch, entry connector.RawTableEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 1)
	defer cancel()
	if err := device.WriteEntry(ctx, entry); err != nil {
		if connector.IsEntryExistsError(err) {
			fmt.Printf("Entry already exists, table = %s\n", entry.TableName)
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
	succesor := reversedPath[1]
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

func (t *TelemetryService) isCompatible(dev model.Device, programName string) bool {
	return dev.GetType() == model.INC_SWITCH && dev.(*model.IncSwitch).InstalledProgram == programName
}

func (t *TelemetryService) findTelemetryEntitiesForRequest(req *pbt.EnableTelemetryRequest, G model.TopologyGraph) TelemetryEntities {
	res := TelemetryEntities{
		Sources: map[Edge][]TernaryIpPair{},
		Transits: map[string]struct{}{},
		Sinks: map[Edge]struct{}{},
	}

	// TODO handle tunneled
	sources := req.Sources.(*pbt.EnableTelemetryRequest_RawSources).RawSources
	targets := req.Targets.(*pbt.EnableTelemetryRequest_RawTargets).RawTargets

	for _, source := range sources.DeviceNames {
		for _, target := range targets.DeviceNames {
			if source == target {
				continue
			}
			visited := map[string]bool{}
			reversedPath := []string{}
			dfs(G, target, source, &reversedPath, visited)
			if len(reversedPath) == 0 {
				panic("no path")
			}
			i := len(reversedPath) - 2
			for ; i >= 0; i-- {
				if t.isCompatible(G[reversedPath[i]], req.ProgramName) {
					break
				}
			}
			if i < 0 {
				continue
			}
			sourceDev := reversedPath[i]
			j := 1
			for ; j <= i; j++ {
				if t.isCompatible(G[reversedPath[j]], req.ProgramName) {
					break
				}
			}
			sinkDev := reversedPath[j]
			for k := j + 1; k < i; k++ {
				dev := G[reversedPath[k]]
				if t.isCompatible(dev, req.ProgramName) {
					res.Transits[dev.GetName()] = struct{}{}
				}
			}
			res.Sinks[Edge{from: sinkDev, to: reversedPath[j-1]}] = struct{}{}
			sourceEdge := Edge{from: sourceDev, to: reversedPath[i+1]}
			ipPair := TernaryIpPair{}
			if G[source].GetType() == model.EXTERNAL {
				ipPair.src = AnyIpv4Ternary()
			} else {
				ipPair.src = ExactIpv4Ternary(G[source].MustGetLinkTo(reversedPath[len(reversedPath) - 2]).Ipv4)
			}
			if G[target].GetType() == model.EXTERNAL {
				ipPair.dst = AnyIpv4Ternary()
			} else {
				ipPair.dst = ExactIpv4Ternary(G[target].MustGetLinkTo(reversedPath[1]).Ipv4)
			}
			if ips, ok := res.Sources[sourceEdge]; ok {
				res.Sources[sourceEdge] = append(ips, ipPair)
			} else {
				res.Sources[sourceEdge] = []TernaryIpPair{ipPair}
			}
		}
	}

	return res
}
