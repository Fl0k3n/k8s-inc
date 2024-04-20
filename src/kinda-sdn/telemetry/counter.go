package telemetry

import (
	"fmt"
	"sync"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

type TableState struct {
	counter int
	entry connector.RawTableEntry
}

type DeviceState [K comparable] struct {
	lock *sync.Mutex
	tableState map[K]TableState
}

type StateCounter [K comparable] struct {
	deviceState sync.Map // map[model.DeviceName]*DeviceState
}

func newStateCounter[K comparable]() *StateCounter[K] {
	return &StateCounter[K]{
		deviceState: sync.Map{},
	}
}

func (s *StateCounter[K]) AddDevice(devName model.DeviceName) {
	state := &DeviceState[K]{
		lock: &sync.Mutex{},
		tableState: map[K]TableState{},
	}
	s.deviceState.Store(devName, state)
}

// counter is incremented and runnable is run atomically if counter was 0 prior to this call
// lock is released once runnable returns, if new goroutines are created and not awaited their
// actions won't be atomic with respect to the counter change, if runnable returns error counter 
// is not incremented, returns error returned by runnable or nil if it wasn't run
func (s *StateCounter[K]) IncrementAndRunOnTransitionToOne(
	devName model.DeviceName,
	key K,
	val connector.RawTableEntry,
	runnable func() error,
) error {
	devState, ok := s.deviceState.Load(devName)
	if !ok {
		panic(fmt.Sprintf("Device name %s state not initialized", devName))
	}
	state := devState.(*DeviceState[K])
	state.lock.Lock()
	defer state.lock.Unlock()
	tableState, ok := state.tableState[key]
	if !ok {
		tableState.counter = 0
		tableState.entry = val
	}
	if tableState.counter == 0 {
		if err := runnable(); err != nil {
			return err	
		}
	}
	tableState.counter += 1
	state.tableState[key] = tableState
	return nil
}

func (s *StateCounter[K]) DecrementAndRunOnTransitionToZero(
	devName model.DeviceName,
	key K,
	runnable func(val connector.RawTableEntry) error,
) error {
	devState, ok := s.deviceState.Load(devName)
	if !ok {
		panic(fmt.Sprintf("Device name %s state not initialized", devName))
	}
	state := devState.(*DeviceState[K])
	state.lock.Lock()
	defer state.lock.Unlock()
	tableState, ok := state.tableState[key]
	if !ok {
		return fmt.Errorf("attempted to decrement counter for non-existing entry")
	}
	if tableState.counter == 1 {
		if err := runnable(tableState.entry); err != nil {
			return err	
		}
		delete(state.tableState, key)
	} else {
		tableState.counter--
		state.tableState[key] = tableState
	}
	return nil
}
