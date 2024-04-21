package telemetry

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)


type StateChangeAction struct {
	deviceName model.DeviceName
	key MatchKey
	entry connector.RawTableEntry
	counter *StateCounter
	delta int
	isCreate bool
} 

func (s *StateChangeAction) Reverse() *StateChangeAction {
	return &StateChangeAction{
		deviceName: s.deviceName,
		key: s.key,
		entry: s.entry,
		counter: s.counter,
		delta: -s.delta,
		isCreate: !s.isCreate,
	}
}

func newCreateAction(deviceName model.DeviceName, key MatchKey, entry connector.RawTableEntry, counter *StateCounter) *StateChangeAction {
	return &StateChangeAction{
		deviceName: deviceName,
		key: key,
		entry: entry,
		counter: counter,
		delta: 1,
		isCreate: true,
	}
}

func newDeleteAction(deviceName model.DeviceName, key MatchKey, entry connector.RawTableEntry, counter *StateCounter) *StateChangeAction {
	return &StateChangeAction{
		deviceName: deviceName,
		key: key,
		entry: entry,
		counter: counter,
		delta: -1,
		isCreate: false,
	}
}

// not thread-safe
type ChangeLog struct {
	actions []*StateChangeAction
	doneCounter int
}

func newChangeLog() *ChangeLog {
	return &ChangeLog{
		actions: []*StateChangeAction{},
		doneCounter: -1,
	}
}

// other's actions are appended at the end
func (c *ChangeLog) ExtendFrom(other *ChangeLog) {
	if c.doneCounter != -1 || other.doneCounter != -1 {
		panic("only uncommited changelogs can be merged")
	}
	c.actions = append(c.actions, other.actions...)	
}

func (c *ChangeLog) AddAction(action *StateChangeAction) {
	c.actions = append(c.actions, action)
}

func (c *ChangeLog) Commit(consumer func (action *StateChangeAction) error) error {
	for _, action := range c.actions {
		err := consumer(action)
		if err != nil {
			return err
		}
		c.doneCounter++
	}
	return nil
}

func (c *ChangeLog) Rollback(consumer func (action *StateChangeAction) error) []*StateChangeAction {
	undoFailedActions := []*StateChangeAction{}
	for i := c.doneCounter; i >= 0; i-- {
		err := consumer(c.actions[i].Reverse())
		if err != nil {
			undoFailedActions = append(undoFailedActions, c.actions[i])
		}
		c.doneCounter--
	}
	return undoFailedActions
}
