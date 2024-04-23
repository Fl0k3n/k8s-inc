package telemetry

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

type TableState struct {
	counter int
	entry connector.RawTableEntry
}

type DeviceState struct {
	lock *sync.Mutex
	tableState map[MatchIdentifier]TableState
	numEntries *atomic.Int32
}

type StateCounter struct {
	deviceState sync.Map // map[model.DeviceName]*DeviceState
}

func newStateCounter() *StateCounter {
	return &StateCounter{
		deviceState: sync.Map{},
	}
}

func (s *StateCounter) AddDevice(devName model.DeviceName) {
	numEntries := &atomic.Int32{}
	numEntries.Store(0)
	state := &DeviceState{
		lock: &sync.Mutex{},
		tableState: map[MatchIdentifier]TableState{},
		numEntries: numEntries,
	}
	s.deviceState.Store(devName, state)
}

func (s *StateCounter) TakePerDeviceNumberOfEntriesSnapshot() map[model.DeviceName]int32 {
	res := map[model.DeviceName]int32{}
	s.deviceState.Range(func(key, value any) bool {
		v := value.(*DeviceState)
		res[key.(model.DeviceName)] = int32(v.numEntries.Load())
		return true
	})
	return res
} 

// counter is incremented and runnable is run atomically if counter was 0 prior to this call
// lock is released once runnable returns, if new goroutines are created and not awaited their
// actions won't be atomic with respect to the counter change, if runnable returns error counter 
// is not incremented, returns error returned by runnable or nil if it wasn't run
func (s *StateCounter) IncrementAndRunOnTransitionToOne(
	devName model.DeviceName,
	key MatchKey,
	val connector.RawTableEntry,
	runnable func() error,
) error {
	devState, ok := s.deviceState.Load(devName)
	if !ok {
		panic(fmt.Sprintf("Device name %s state not initialized", devName))
	}
	state := devState.(*DeviceState)
	state.lock.Lock()
	defer state.lock.Unlock()
	tableState, ok := state.tableState[key.ToIdentifier()]
	if !ok {
		tableState.counter = 0
		tableState.entry = val
		state.numEntries.Add(1)
	}
	if tableState.counter == 0 {
		if err := runnable(); err != nil {
			return err	
		}
	}
	tableState.counter += 1
	state.tableState[key.ToIdentifier()] = tableState
	return nil
}

func (s *StateCounter) DecrementAndRunOnTransitionToZero(
	devName model.DeviceName,
	key MatchKey,
	runnable func(val connector.RawTableEntry) error,
) error {
	devState, ok := s.deviceState.Load(devName)
	if !ok {
		panic(fmt.Sprintf("Device name %s state not initialized", devName))
	}
	state := devState.(*DeviceState)
	state.lock.Lock()
	defer state.lock.Unlock()
	tableState, ok := state.tableState[key.ToIdentifier()]
	if !ok {
		return fmt.Errorf("attempted to decrement counter for non-existing entry")
	}
	if tableState.counter == 1 {
		if err := runnable(tableState.entry); err != nil {
			return err	
		}
		state.numEntries.Add(-1)
		delete(state.tableState, key.ToIdentifier())
	} else {
		tableState.counter--
		state.tableState[key.ToIdentifier()] = tableState
	}
	return nil
}
