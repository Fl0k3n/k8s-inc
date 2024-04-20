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

var ENTRY_CONFIG_TIMEOUT = 2 * time.Second

const PROGRAM_INTERFACE = "inc.kntp.com/v1alpha1/telemetry"

type IntentState struct {
	Entities *TelemetryEntities
	CollectionId string
}

type TelemetryService struct {
	intentMapLock sync.Mutex
	intentLocks map[string]*sync.Mutex
	registry programs.P4ProgramRegistry

	transitStateCounter    *StateCounter[model.DeviceName]
	sinkStateCounter       *StateCounter[ConfigureSinkKey]
	raportingStateCounter  *StateCounter[ReportingKey]
	activateSourceCounter  *StateCounter[ActivateSourceKey]
	configureSourceCounter *StateCounter[ConfigureSourceKey]

	collectionIdPool *CollectionIdPool
	// key=intentId: str -> value=telemetryEntities: *IntentEntityState
	intentState sync.Map 
}

func NewService(registry programs.P4ProgramRegistry) *TelemetryService {
	return &TelemetryService{
		intentMapLock: sync.Mutex{},
		intentLocks: make(map[string]*sync.Mutex),
		registry: registry,
		transitStateCounter: newStateCounter[model.DeviceName](),
		sinkStateCounter: newStateCounter[ConfigureSinkKey](),
		raportingStateCounter: newStateCounter[ReportingKey](),
		activateSourceCounter: newStateCounter[ActivateSourceKey](),
		configureSourceCounter: newStateCounter[ConfigureSourceKey](),
		collectionIdPool: newPool(COLLECTION_ID_POOL_SIZE),
		intentState: sync.Map{},
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
		t.transitStateCounter.AddDevice(switchName)
		t.sinkStateCounter.AddDevice(switchName)
		t.raportingStateCounter.AddDevice(switchName)
		t.activateSourceCounter.AddDevice(switchName)
		t.configureSourceCounter.AddDevice(switchName)
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
	if ok := t.lockIntent(req.IntentId); !ok {
		description := "Already handling intent"
		return &pbt.EnableTelemetryResponse{
			TelemetryState: pbt.TelemetryState_IN_PROGRESS,
			Description: &description,
		}, nil
	}
	defer t.unlockIntent(req.IntentId)

	switchIds := getSwitchIds(topo)
	G := model.TopologyToGraph(topo)
	desiredEntities := t.findTelemetryEntitiesForRequest(req, G)
	currentIntentState_, hasCurrentState := t.intentState.Load(req.IntentId)
	var entriesToAdd, entriesToRemove *TelemetryEntities

	if hasCurrentState {
		currentIntentState := currentIntentState_.(*IntentState)
		if currentIntentState.CollectionId != req.CollectionId {
			// TODO: support it, not critical though
			description := "Changing collection id is unsupported"
			return &pbt.EnableTelemetryResponse{
				TelemetryState: pbt.TelemetryState_FAILED,
				Description: &description,
			}, nil 
		}
		entriesToAdd, entriesToRemove = computeDifferences(currentIntentState.Entities, desiredEntities)
	} else {
		entriesToAdd = desiredEntities
		entriesToRemove = newTelemetryEntities()
	}
	collectionid, err := t.collectionIdPool.AllocOrGet(req.IntentId, req.CollectionId)
	if err != nil {
		description := "Resources exhausted, use different collection id"
		return &pbt.EnableTelemetryResponse{
			TelemetryState: pbt.TelemetryState_FAILED,
			Description: &description,
		}, nil 
	}
	if !entriesToAdd.IsEmpty() {
		if err := t.applyConfiguration(G, switchIds, deviceProvider, entriesToAdd,
				req.CollectorNodeName, int(req.CollectorPort), collectionid); err != nil {
			return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, nil
		}
	}
	if !entriesToRemove.IsEmpty() {
		if err := t.deleteConfiguration(G, deviceProvider, entriesToRemove, collectionid); err != nil {
			return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_FAILED}, nil
		}
	}
	currentIntentState := &IntentState{
		Entities: desiredEntities,
		CollectionId: req.CollectionId,
	}
	t.intentState.Store(req.IntentId, currentIntentState)
	return &pbt.EnableTelemetryResponse{TelemetryState: pbt.TelemetryState_OK}, nil
}

func (t *TelemetryService) DisableTelemetry(
	req *pbt.DisableTelemetryRequest,
	topo *model.Topology,
	deviceProvider DeviceProvider,
) (*pbt.DisableTelemetryResponse, error) {
	// don't unlock it, we hold the lock until it's removed
	if ok := t.lockIntent(req.IntentId); !ok {
		return &pbt.DisableTelemetryResponse{ShouldRetryLater: true}, nil
	}
	// remove it concurrently and answer immediately
	go func() {
		currentIntentState_, ok := t.intentState.Load(req.IntentId)
		if !ok {
			return
		}
		currentIntentState := currentIntentState_.(*IntentState)
		G := model.TopologyToGraph(topo)
		collectionIdNum, ok := t.collectionIdPool.GetIfAllocated(req.IntentId, currentIntentState.CollectionId)
		if !ok {
			panic("should be allocated")
		}
		err := t.deleteConfiguration(G, deviceProvider, currentIntentState.Entities, collectionIdNum)
		if err != nil {
			// TODO: add it to some log and schedule retries
			fmt.Printf("Failed to delete configuration %e", err)
			return
		}
		t.intentState.Delete(req.IntentId)
		t.collectionIdPool.Free(req.IntentId, currentIntentState.CollectionId)
	}()
	// treat is as a promise that it would be deleted
	return &pbt.DisableTelemetryResponse{ShouldRetryLater: false}, nil
}

func (t *TelemetryService) applyConfiguration(
	G model.TopologyGraph,
	switchIds map[string]int,
	deviceProvider DeviceProvider,
	desiredEntities *TelemetryEntities,
	collectorNodeName string,
	collectorPort int,
	collectionIdNum int,
) error {
	for sink := range desiredEntities.Sinks {
		deviceMeta := G[sink.from].(*model.IncSwitch)
		device := deviceProvider(sink.from)
		portNumber, ok := deviceMeta.GetPortNumberTo(sink.to)
		if !ok {
			panic("no such neighbor")
		}
		collector := G[collectorNodeName]
		portToCollector := t.findPortLeadingToDevice(G, G[sink.from], collector)
		sinkKey := ConfigureSinkKey{egressPort: portNumber + 1}
		entry := ConfigureSink(sinkKey, portToCollector + 1)
		err := t.sinkStateCounter.IncrementAndRunOnTransitionToOne(sink.from, sinkKey, entry, func() error {
			return t.writeEntry(device, entry)
		})
		if err != nil {
			fmt.Printf("Failed to configure sink %e\n", err)
			return err
		}

		nextHopToCollector := G[deviceMeta.GetLinks()[portToCollector].To]
		nextHopToCollectorMac := nextHopToCollector.MustGetLinkTo(deviceMeta.Name).MacAddr
		portToSink := t.findPortLeadingToDevice(G, collector, G[sink.from])
		reportingKey := ReportingKey{collectionId: collectionIdNum}
		entry = Reporting(
			reportingKey, 
			deviceMeta.GetLinks()[portToCollector].MacAddr, deviceMeta.GetLinks()[portToCollector].Ipv4,
			nextHopToCollectorMac, collector.GetLinks()[portToSink].Ipv4, collectorPort,
		)
		err = t.raportingStateCounter.IncrementAndRunOnTransitionToOne(sink.from, reportingKey, entry, func() error {
			return t.writeEntry(device, entry)
		})	
		if err != nil {
			fmt.Printf("Failed to configure reporting %e\n", err)
			return err
		}
	}
	for transit := range desiredEntities.Transits {
		entry := Transit(switchIds[transit], TELEMETRY_MTU)
		err := t.transitStateCounter.IncrementAndRunOnTransitionToOne(transit, transit, entry, func() error {
			return t.writeEntry(deviceProvider(transit), entry)
		})
		if err != nil {
			fmt.Printf("Failed to configure transit %e", err)
			return err
		}
	}
	for source, configEntries := range desiredEntities.Sources {
		deviceMeta := G[source.from].(*model.IncSwitch)
		device := deviceProvider(source.from)
		portToActivate, ok := deviceMeta.GetPortNumberTo(source.to)
		if !ok {
			panic("no such neighbor")
		}
		activateSourceKey := ActivateSourceKey{ingressPort: portToActivate + 1}
		entry := ActivateSource(activateSourceKey)
		err := t.activateSourceCounter.IncrementAndRunOnTransitionToOne(source.from, activateSourceKey, entry, func () error {
			return t.writeEntry(device, entry)
		})
		if err != nil {
			fmt.Printf("Failed to activate source %e", err)
			return err
		}
		for _, config := range configEntries {
			key := ConfigureSourceKey{
				srcAddr: config.ips.src,
				dstAddr: config.ips.dst,
				srcPort: config.srcPort,
				dstPort: config.dstPort,
				tunneled: config.tunneled,
			}
			entry := ConfigureSource(key, 6, 10, 8, FULL_TELEMETRY_KEY, collectionIdNum)
			err = t.configureSourceCounter.IncrementAndRunOnTransitionToOne(source.from, key, entry, func() error {
				return t.writeEntry(device, entry)
			})
			if err != nil {
				fmt.Printf("Failed to configure source %e", err)
				return err
			}
		}
	}
	return nil
}

func (t *TelemetryService) deleteConfiguration(
	G model.TopologyGraph,
	deviceProvider DeviceProvider,
	entriesToDelete *TelemetryEntities,
	collectionIdNum int,
) error {
	for source, configEntries := range entriesToDelete.Sources {
		deviceMeta := G[source.from].(*model.IncSwitch)
		device := deviceProvider(source.from)
		portToActivate, ok := deviceMeta.GetPortNumberTo(source.to)
		if !ok {
			panic("no such neighbor")
		}
		for _, config := range configEntries {
			key := ConfigureSourceKey{
				srcAddr: config.ips.src,
				dstAddr: config.ips.dst,
				srcPort: config.srcPort,
				dstPort: config.dstPort,
				tunneled: config.tunneled,
			}
			err := t.configureSourceCounter.DecrementAndRunOnTransitionToZero(source.from, key, func(entry connector.RawTableEntry) error {
				return t.deleteEntry(device, entry)
			})
			if err != nil {
				fmt.Printf("Failed to remove source configuration %e", err)
				return err
			}
		}
		activateSourceKey := ActivateSourceKey{ingressPort: portToActivate + 1}
		err := t.activateSourceCounter.DecrementAndRunOnTransitionToZero(
				source.from,
				activateSourceKey,
				func (entry connector.RawTableEntry) error {
			return t.deleteEntry(device, entry)
		})
		if err != nil {
			fmt.Printf("Failed to deactivate source %e", err)
			return err
		}
	}
	for transit := range entriesToDelete.Transits {
		err := t.transitStateCounter.DecrementAndRunOnTransitionToZero(transit, transit, func(entry connector.RawTableEntry) error {
			return t.deleteEntry(deviceProvider(transit), entry)
		})
		if err != nil {
			fmt.Printf("Failed to deactivate transit %e", err)
			return err
		}
	}
	for sink := range entriesToDelete.Sinks {
		deviceMeta := G[sink.from].(*model.IncSwitch)
		device := deviceProvider(sink.from)
		portNumber, ok := deviceMeta.GetPortNumberTo(sink.to)
		if !ok {
			panic("no such neighbor")
		}
		reportingKey := ReportingKey{collectionId: collectionIdNum}
		err := t.raportingStateCounter.DecrementAndRunOnTransitionToZero(sink.from, reportingKey, func(entry connector.RawTableEntry) error {
			return t.deleteEntry(device, entry)
		})	
		if err != nil {
			fmt.Printf("Failed to deactivate reporting %e", err)
			return err
		}
		sinkKey := ConfigureSinkKey{egressPort: portNumber + 1}
		err = t.sinkStateCounter.DecrementAndRunOnTransitionToZero(sink.from, sinkKey, func(entry connector.RawTableEntry) error {
			return t.deleteEntry(device, entry)
		})
		if err != nil {
			fmt.Printf("Failed to deactivate sink %e", err)
			return err
		}
	}
	return nil
}

func (t *TelemetryService) lockIntent(intentId string) (success bool) {
	t.intentMapLock.Lock()
	if mapLock, ok := t.intentLocks[intentId]; ok {
		locked := mapLock.TryLock()
		if !locked {
			t.intentMapLock.Unlock()
			return false
		}
	} else {
		lock := &sync.Mutex{}
		lock.Lock()
		t.intentLocks[intentId] = lock
	}
	t.intentMapLock.Unlock()
	return true
}

func (t *TelemetryService) unlockIntent(intentId string) {
	t.intentMapLock.Lock()
	defer t.intentMapLock.Unlock()
	t.intentLocks[intentId].Unlock()
}

func (t *TelemetryService) writeEntry(device device.IncSwitch, entry connector.RawTableEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), ENTRY_CONFIG_TIMEOUT)
	defer cancel()
	if err := device.WriteEntry(ctx, entry); err != nil {
		if connector.IsEntryExistsError(err) {
			fmt.Printf("Entry already exists, table = %s\n", entry.TableName)
		} else {
			fmt.Printf("Failed to write entry %s\n%e", entry, err)
		}
		return err
	}
	return nil
}

func (t *TelemetryService) deleteEntry(device device.IncSwitch, entry connector.RawTableEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), ENTRY_CONFIG_TIMEOUT)
	defer cancel()
	if err := device.DeleteEntry(ctx, entry); err != nil {
		if connector.IsEntryNotFoundError(err) {
			fmt.Printf("Entry doesn't exists, table = %s\n", entry.TableName)
		} else {
			fmt.Printf("Failed to write entry %s\n%e", entry, err)
		}
		return err
	}
	return nil
}
