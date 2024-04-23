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
	CollectorNodeName string
	CollectorPort int
}

type TelemetryService struct {
	intentMapLock sync.Mutex
	intentLocks map[string]*sync.Mutex
	registry programs.P4ProgramRegistry

	transitStateCounter    *StateCounter
	sinkStateCounter       *StateCounter
	raportingStateCounter  *StateCounter
	activateSourceCounter  *StateCounter
	configureSourceCounter *StateCounter

	collectionIdPool *CollectionIdPool
	// key=intentId: str -> value=telemetryEntities: *IntentEntityState
	intentState sync.Map

	sourceCapabilityMonitor *SourceCapabilityMonitor
}

func NewService(registry programs.P4ProgramRegistry) *TelemetryService {
	return &TelemetryService{
		intentMapLock: sync.Mutex{},
		intentLocks: make(map[string]*sync.Mutex),
		registry: registry,
		transitStateCounter: newStateCounter(),
		sinkStateCounter: newStateCounter(),
		raportingStateCounter: newStateCounter(),
		activateSourceCounter: newStateCounter(),
		configureSourceCounter: newStateCounter(),
		collectionIdPool: newPool(COLLECTION_ID_POOL_SIZE),
		intentState: sync.Map{},
		sourceCapabilityMonitor: newSourceCapabilityMonitor(),
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

// topology should be immutable for the duration of this call, pass snapshot if it could change
func (t *TelemetryService) ConfigureTelemetry(
	req *pbt.ConfigureTelemetryRequest,
	topo *model.Topology, 
	deviceProvider DeviceProvider,
) (*pbt.ConfigureTelemetryResponse, error) {
	if ok := t.lockIntent(req.IntentId); !ok {
		description := "Already handling intent"
		return &pbt.ConfigureTelemetryResponse{
			TelemetryState: pbt.TelemetryState_IN_PROGRESS,
			Description: &description,
		}, nil
	}

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
			t.unlockIntent(req.IntentId)
			return &pbt.ConfigureTelemetryResponse{
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
		if hasCurrentState {
			t.unlockIntent(req.IntentId)
		} else {
			t.deleteIntentLock(req.IntentId)
		}
		return &pbt.ConfigureTelemetryResponse{
			TelemetryState: pbt.TelemetryState_FAILED,
			Description: &description,
		}, nil 
	}
	var changelog *ChangeLog = nil
	if !entriesToAdd.IsEmpty() {
		changelog = t.makeApplyChangelog(t.getEntriesForTelemetryEntities(G, switchIds, entriesToAdd,
			req.CollectorNodeName, int(req.CollectorPort), collectionid))
	}
	if !entriesToRemove.IsEmpty() {
		entities := t.getEntriesForTelemetryEntities(G, switchIds, entriesToRemove,
			req.CollectorNodeName, int(req.CollectorPort), collectionid)
		deleteChangeLog := t.makeDeleteChangelog(entities)
		if changelog == nil {
			changelog = deleteChangeLog
		} else {
			changelog.ExtendFrom(deleteChangeLog)
		}
	}
	if changelog != nil {
		if err := t.commit(deviceProvider, changelog); err != nil {
			if hasCurrentState {
				t.unlockIntent(req.IntentId)
			} else {
				t.collectionIdPool.Free(req.IntentId, req.CollectionId)
				t.deleteIntentLock(req.IntentId)
			}
			description := "Failed to configure telemetry, couldn't apply some of required changes, retrying may help"
			return &pbt.ConfigureTelemetryResponse{
				TelemetryState: pbt.TelemetryState_FAILED,
				Description: &description,
			}, nil
		}
	}
	currentIntentState := &IntentState{
		Entities: desiredEntities,
		CollectionId: req.CollectionId,
		CollectorNodeName: req.CollectorNodeName,
		CollectorPort: int(req.CollectorPort),
	}
	t.intentState.Store(req.IntentId, currentIntentState)
	t.unlockIntent(req.IntentId)
	return &pbt.ConfigureTelemetryResponse{TelemetryState: pbt.TelemetryState_OK}, nil
}

func (t *TelemetryService) DisableTelemetry(
	req *pbt.DisableTelemetryRequest,
	topo *model.Topology,
	deviceProvider DeviceProvider,
) (*pbt.DisableTelemetryResponse, error) {
	// don't unlock it, we hold the lock until it's removed, other callers always lock it in a non-blocking way
	exists, locked := t.tryLockIntentIfExists(req.IntentId)
	if !exists {
		return &pbt.DisableTelemetryResponse{ShouldRetryLater: false}, nil
	}	
	if !locked {
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
		entities := t.getEntriesForTelemetryEntities(G, getSwitchIds(topo), currentIntentState.Entities, 
			currentIntentState.CollectorNodeName, currentIntentState.CollectorPort, collectionIdNum)
		changelog := t.makeDeleteChangelog(entities)
		if err := t.commit(deviceProvider, changelog); err != nil {
			// TODO: add it to some log and schedule retries
			fmt.Printf("Failed to delete configuration %e", err)
			t.unlockIntent(req.IntentId) // just while we don't do anything better
			return
		}
		t.intentState.Delete(req.IntentId)
		t.collectionIdPool.Free(req.IntentId, currentIntentState.CollectionId)
		t.deleteIntentLock(req.IntentId)
	}()
	// treat is as a promise that it would be deleted
	return &pbt.DisableTelemetryResponse{ShouldRetryLater: false}, nil
}

// to unobserve caller should close the given channel
func (t *TelemetryService) ObserveSourceCapabilityUpdates(stopChan chan struct{}) chan *pbt.SourceCapabilityUpdate {
	return t.sourceCapabilityMonitor.RegisterDownstreamObserver(stopChan)
}

func (t *TelemetryService) getEntriesForTelemetryEntities(
	G model.TopologyGraph,
	switchIds map[string]int,
	telemetryEntities *TelemetryEntities,
	collectorNodeName string,
	collectorPort int,
	collectionIdNum int,
) *ConfigEntries {
	entries := &ConfigEntries{
		ActivateSourceEntries: []*ConfigEntry{},
		ConfigureSourceEntries: []*ConfigEntry{},
		ConfigureTransitEntries: []*ConfigEntry{},
		ConfigureSinkEntries: []*ConfigEntry{},
		ConfigureReportingEntry: []*ConfigEntry{},
	}
	for sink := range telemetryEntities.Sinks {
		deviceMeta := G[sink.from].(*model.IncSwitch)
		portNumber, ok := deviceMeta.GetPortNumberTo(sink.to)
		if !ok {
			panic("no such neighbor")
		}
		collector := G[collectorNodeName]
		portToCollector := t.findPortLeadingToDevice(G, G[sink.from], collector)
		sinkKey := &ConfigureSinkKey{egressPort: portNumber + 1}
		entries.ConfigureSinkEntries = append(entries.ConfigureSinkEntries, &ConfigEntry{
			deviceName: sink.from,
			key: sinkKey,
			entry: ConfigureSink(sinkKey, portToCollector + 1),
		})

		nextHopToCollector := G[deviceMeta.GetLinks()[portToCollector].To]
		nextHopToCollectorMac := nextHopToCollector.MustGetLinkTo(deviceMeta.Name).MacAddr
		portToSink := t.findPortLeadingToDevice(G, collector, G[sink.from])
		reportingKey := &ReportingKey{collectionId: collectionIdNum}
		entries.ConfigureReportingEntry = append(entries.ConfigureReportingEntry, &ConfigEntry{
			deviceName: sink.from,
			key: reportingKey,
			entry: Reporting(
				reportingKey, 
				deviceMeta.GetLinks()[portToCollector].MacAddr, deviceMeta.GetLinks()[portToCollector].Ipv4,
				nextHopToCollectorMac, collector.GetLinks()[portToSink].Ipv4, collectorPort,
			),
		})
	}
	for transit := range telemetryEntities.Transits {
		entries.ConfigureTransitEntries = append(entries.ConfigureTransitEntries, &ConfigEntry{
			deviceName: transit,
			key: &ConfigureTransitKey{},
			entry: Transit(switchIds[transit], TELEMETRY_MTU),
		})
	}
	for source, configEntries := range telemetryEntities.Sources {
		deviceMeta := G[source.from].(*model.IncSwitch)
		portToActivate, ok := deviceMeta.GetPortNumberTo(source.to)
		if !ok {
			panic("no such neighbor")
		}
		activateSourceKey := &ActivateSourceKey{ingressPort: portToActivate + 1}
		entries.ActivateSourceEntries = append(entries.ActivateSourceEntries, &ConfigEntry{
			deviceName: source.from,
			key: activateSourceKey,
			entry: ActivateSource(activateSourceKey),
		})
		for _, config := range configEntries {
			configureSourceKey := &ConfigureSourceKey{
				srcAddr: config.ips.src,
				dstAddr: config.ips.dst,
				srcPort: config.srcPort,
				dstPort: config.dstPort,
				tunneled: config.tunneled,
			}
			entries.ConfigureSourceEntries = append(entries.ConfigureSourceEntries, &ConfigEntry{
				deviceName: source.from,
				key: configureSourceKey,
				entry: ConfigureSource(configureSourceKey, 6, 10, 8, FULL_TELEMETRY_KEY, collectionIdNum),
			})
		}
	}
	return entries
}

// if error occurs changes are rolled-back
func (t *TelemetryService) commit(deviceProvider DeviceProvider, changelog *ChangeLog) error {
	applyChange := func(action *StateChangeAction) error {
		var err error = nil
		if action.isCreate {
			err = action.counter.IncrementAndRunOnTransitionToOne(
				action.deviceName,
				action.key,
				action.entry,
				func() error {
					return t.writeEntry(deviceProvider(action.deviceName), action.entry)
				},
			)	
		} else {
			err = action.counter.DecrementAndRunOnTransitionToZero(
				action.deviceName,
				action.key,
				func(entry connector.RawTableEntry) error {
					return t.deleteEntry(deviceProvider(action.deviceName), entry)
				},
			)	
		}
		return err
	}
	err := changelog.Commit(applyChange)
	if err != nil {
		failedActions := changelog.Rollback(applyChange)
		if len(failedActions) > 0 {
			// TODO: schedule retries or notify some{thing,body} or check why they failed and do something with it
			for _, action := range failedActions {
				fmt.Printf("failed to rollback action: %v\n", action)
			}
		}
	}
	t.sourceCapabilityMonitor.NotifyChanged(t.createCapabilityUpdate())
	return err
}

func (t *TelemetryService) makeApplyChangelog(entries *ConfigEntries) *ChangeLog {
	changelog := newChangeLog()
	for _, entry := range entries.ConfigureSinkEntries {
		changelog.AddAction(newCreateAction(entry.deviceName, entry.key, entry.entry, t.sinkStateCounter))
	}
	for _, entry := range entries.ConfigureReportingEntry {
		changelog.AddAction(newCreateAction(entry.deviceName, entry.key, entry.entry, t.raportingStateCounter))
	}
	for _, entry := range entries.ConfigureTransitEntries {
		changelog.AddAction(newCreateAction(entry.deviceName, entry.key, entry.entry, t.transitStateCounter))
	}
	for _, entry := range entries.ActivateSourceEntries {
		changelog.AddAction(newCreateAction(entry.deviceName, entry.key, entry.entry, t.activateSourceCounter)) 
	}
	for _, entry := range entries.ConfigureSourceEntries {
		changelog.AddAction(newCreateAction(entry.deviceName, entry.key, entry.entry, t.configureSourceCounter))
	}
	return changelog
}

func (t *TelemetryService) makeDeleteChangelog(entries *ConfigEntries) *ChangeLog {
	changelog := newChangeLog()
	for _, entry := range entries.ConfigureSourceEntries {
		changelog.AddAction(newDeleteAction(entry.deviceName, entry.key, entry.entry, t.configureSourceCounter))
	}
	for _, entry := range entries.ActivateSourceEntries {
		changelog.AddAction(newDeleteAction(entry.deviceName, entry.key, entry.entry, t.activateSourceCounter)) 
	}
	for _, entry := range entries.ConfigureTransitEntries {
		changelog.AddAction(newDeleteAction(entry.deviceName, entry.key, entry.entry, t.transitStateCounter))
	}
	for _, entry := range entries.ConfigureReportingEntry {
		changelog.AddAction(newDeleteAction(entry.deviceName, entry.key, entry.entry, t.raportingStateCounter))
	}
	for _, entry := range entries.ConfigureSinkEntries {
		changelog.AddAction(newDeleteAction(entry.deviceName, entry.key, entry.entry, t.sinkStateCounter))
	}
	return changelog
}

func (t *TelemetryService) createCapabilityUpdate() *pbt.SourceCapabilityUpdate {
	// TODO: this should be fetched from devices at startup/program installation
	CAPACITY := 127
	takenEntries := t.configureSourceCounter.TakePerDeviceNumberOfEntriesSnapshot()
	for devName, takenCount := range takenEntries {
		takenEntries[devName] = int32(CAPACITY) - takenCount
	}
	return &pbt.SourceCapabilityUpdate{
		RemainingSourceEndpoints: takenEntries,
	}
}

func (t *TelemetryService) lockIntent(intentId string) (success bool) {
	t.intentMapLock.Lock()
	defer t.intentMapLock.Unlock()
	if mapLock, ok := t.intentLocks[intentId]; ok {
		locked := mapLock.TryLock()
		if !locked {
			return false
		}
	} else {
		lock := &sync.Mutex{}
		lock.Lock()
		t.intentLocks[intentId] = lock
	}
	return true
}

func (t *TelemetryService) tryLockIntentIfExists(intentId string) (exists bool, success bool) {
	t.intentMapLock.Lock()
	defer t.intentMapLock.Unlock()
	if mapLock, ok := t.intentLocks[intentId]; ok {
		locked := mapLock.TryLock()
		if !locked {
			return true, false
		}
	} else {
		return false, true
	}
	return true, true
}

func (t *TelemetryService) unlockIntent(intentId string) {
	t.intentMapLock.Lock()
	defer t.intentMapLock.Unlock()
	t.intentLocks[intentId].Unlock()
}

// intentMapLock MUST NOT be held by the caller and intentLock MUST be held by the caller
func (t *TelemetryService) deleteIntentLock(intentId string) {
	t.intentMapLock.Lock()
	defer t.intentMapLock.Unlock()
	delete(t.intentLocks, intentId)
}

func (t *TelemetryService) writeEntry(device device.IncSwitch, entry connector.RawTableEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), ENTRY_CONFIG_TIMEOUT)
	defer cancel()
	if err := device.WriteEntry(ctx, entry); err != nil {
		if connector.IsEntryExistsError(err) {
			fmt.Printf("Entry already exists, table = %s\n", entry.TableName)
			// return nil // ignore it
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
			// return nil // ignore it
		} else {
			fmt.Printf("Failed to write entry %s\n%e", entry, err)
		}
		return err
	}
	return nil
}
