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

const (
	TELEMETRY_MTU = 1500
	COLLECTION_ID_POOL_SIZE = 128
)

const PROGRAM_INTERFACE = "inc.kntp.com/v1alpha1/telemetry"

type TelemetryService struct {
	intentMapLock sync.Mutex
	intentLocks map[string]*sync.Mutex
	registry programs.P4ProgramRegistry
	transitCounters map[model.DeviceName]int
	transitLocks map[model.DeviceName]*sync.Mutex
	collectionIdPool *CollectionIdPool
	// key=intentId: str -> value=telemetryEntities: *TelemetryEntities
	intentEntityState sync.Map 
}

func NewService(registry programs.P4ProgramRegistry) *TelemetryService {
	return &TelemetryService{
		intentMapLock: sync.Mutex{},
		intentLocks: make(map[string]*sync.Mutex),
		registry: registry,
		transitCounters: make(map[string]int),
		transitLocks: make(map[string]*sync.Mutex),
		collectionIdPool: newPool(COLLECTION_ID_POOL_SIZE),
		intentEntityState: sync.Map{},
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
	switchIds := getSwitchIds(topo)
	for switchName := range switchIds {
		t.transitCounters[switchName] = 0
		t.transitLocks[switchName] = &sync.Mutex{}
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
	t.intentMapLock.Lock()
	if mapLock, ok := t.intentLocks[req.IntentId]; ok {
		locked := mapLock.TryLock()
		if !locked {
			description := "Already handling intent"
			t.intentMapLock.Unlock()
			return &pbt.EnableTelemetryResponse{
				TelemetryState: pbt.TelemetryState_IN_PROGRESS,
				Description: &description,
			}, nil
		}
	} else {
		lock := &sync.Mutex{}
		lock.Lock()
		t.intentLocks[req.IntentId] = lock
	}
	t.intentMapLock.Unlock()
	defer func() {
		t.intentMapLock.Lock()
		defer t.intentMapLock.Unlock()
		t.intentLocks[req.IntentId].Unlock()
	}()

	// TODO store state and remove not needed entries 

	G := model.TopologyToGraph(topo)
	desiredEntities := t.findTelemetryEntitiesForRequest(req, G)

	if len(desiredEntities.Transits) > 0 {
		switchIds := getSwitchIds(topo)
		for transit := range desiredEntities.Transits {
			t.transitLocks[transit].Lock()
			counter := t.transitCounters[transit]
			t.transitCounters[transit] += 1
			if counter == 0 {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
				if err := deviceProvider(transit).WriteEntry(ctx, Transit(switchIds[transit], TELEMETRY_MTU)); err != nil {
					t.transitLocks[transit].Unlock()
					cancel()
					return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
				}
				cancel()
			}
			t.transitLocks[transit].Unlock()
		}
	}

	// TODO conflicting sinks (?)
	collectorId, err := t.collectionIdPool.AllocOrGet(req.CollectionId)
	if err != nil {
		return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
	}
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
			nextHopToCollectorMac, collector.GetLinks()[portToSink].Ipv4, int(req.CollectorPort), collectorId,
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
				4, 10, 8, FULL_TELEMETRY_KEY, collectorId, config.tunneled,
			)); err != nil {
				return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, err
			}
		}
	}

	return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_OK}, nil
}

func (t *TelemetryService) DisableTelemetry(
	req *pbt.DisableTelemetryRequest,
	topo *model.Topology,
	deviceProvider DeviceProvider,
) (*pbt.DisableTelemetryResponse, error) {
	return nil, nil
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

