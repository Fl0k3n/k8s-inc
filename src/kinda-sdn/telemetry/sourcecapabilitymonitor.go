package telemetry

import (
	"fmt"
	"sync"

	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

type stoppableChan struct {
	downstreamChan chan *pbt.SourceCapabilityUpdate
	stopChan chan struct{}
}

type SourceCapabilityMonitor struct {
	queueLock *sync.Mutex
	sourceCapabilityUpdatesQueue chan *pbt.SourceCapabilityUpdate
	observerListLock *sync.Mutex
	downstreamObservers []*stoppableChan
}

func newSourceCapabilityMonitor() *SourceCapabilityMonitor {
	m := &SourceCapabilityMonitor{
		queueLock: &sync.Mutex{},
		sourceCapabilityUpdatesQueue: make(chan *pbt.SourceCapabilityUpdate, 1),
		observerListLock: &sync.Mutex{},
		downstreamObservers: []*stoppableChan{},
	}
	go m.run()
	return m
}

func (m *SourceCapabilityMonitor) run() {
	for update := range m.sourceCapabilityUpdatesQueue {
		m.observerListLock.Lock()
		observersShallowCopy := []*stoppableChan{}
		observersShallowCopy = append(observersShallowCopy, m.downstreamObservers...)
		m.observerListLock.Unlock()
		removedObservers := []int{}

		for i, obs := range observersShallowCopy {
			select {
			case _, isOpen := <- obs.stopChan:
				if !isOpen {
					removedObservers = append(removedObservers, i)
					close(obs.downstreamChan)
				}
			default:
				obs.downstreamChan <- update
			}
		}

		if len(removedObservers) == 0 {
			continue
		}
		
		remainingObservers := []*stoppableChan{}
		m.observerListLock.Lock()
		j := 0
		for i, obs := range observersShallowCopy {
			if j < len(removedObservers) && i == removedObservers[j] {
				fmt.Printf("Removing observer %d\n", removedObservers[j])
				j++
			} else {
				remainingObservers = append(remainingObservers, obs)
			}
		}
		for j := len(observersShallowCopy); j < len(m.downstreamObservers); j++ {
			remainingObservers = append(remainingObservers, m.downstreamObservers[j])
		}
		m.downstreamObservers = remainingObservers
		m.observerListLock.Unlock()
	}
}

// older messages that weren't consumed are discarded, guaranteed not to block the caller
func (m *SourceCapabilityMonitor) NotifyChanged(update *pbt.SourceCapabilityUpdate) {
	m.queueLock.Lock()
	defer m.queueLock.Unlock()
	// we assume that this won't be called  millions of times per sec, so another goroutine
	// will eventually have a chance to consume some value
	for {
		select {
		case m.sourceCapabilityUpdatesQueue <- update:
			return
		default:
			select {
			// consume the older value and retry writing, we are the only writer here (due to having the lock) 
			// so second write must succeed, regardless of other goroutines consuming older message
			case <- m.sourceCapabilityUpdatesQueue:
			default:
			}
		}
	}
}

// caller should close the stop chan to unsubscribe, messages are written to the returned channel
// this implementation assumes that this would be called rarily and that only a few observers may exist
func (m *SourceCapabilityMonitor) RegisterDownstreamObserver(stopChan chan struct{}) chan *pbt.SourceCapabilityUpdate{
	chans := &stoppableChan{
		downstreamChan: make(chan *pbt.SourceCapabilityUpdate),
		stopChan: stopChan,
	}
	m.observerListLock.Lock()
	defer m.observerListLock.Unlock()
	m.downstreamObservers = append(m.downstreamObservers, chans)
	return chans.downstreamChan
}
