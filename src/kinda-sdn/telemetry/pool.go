package telemetry

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type poolItem struct {
	itemLock *sync.Mutex
	collectionId string
	intents map[string]struct{} // key is indentId
}

type CollectionIdPool struct {
	maxSize int
	pool []poolItem
	allocCounter *atomic.Int32
}

// returns thread-safe collectionId pool
func newPool(maxSize int) *CollectionIdPool {
	// assumes that pool is quite small (<256)
	pool := make([]poolItem, maxSize)
	for i := 0; i < maxSize; i++ {
		counter := &atomic.Int32{}
		counter.Store(0)
		pool[i] = poolItem{collectionId: "", itemLock: &sync.Mutex{}, intents: map[string]struct{}{}}
	}
	counter := &atomic.Int32{}
	counter.Store(0)
	return &CollectionIdPool{
		maxSize: maxSize,
		pool: pool,
		allocCounter: counter,
	}
}

// returns integral identifier for strigified collectionId, in bounds [0, maxSize - 1]
// or error if there are no more free slots, intent may hold only 1 allocation, however, 
// multiple different intents may use same collectionId
func (p *CollectionIdPool) AllocOrGet(intentId string, collectionId string) (int, error) {
	for i := 0; i < p.maxSize; i++ {
		p.pool[i].itemLock.Lock()
		if p.pool[i].collectionId == collectionId {
			if _, ok := p.pool[i].intents[intentId]; !ok {
				p.pool[i].intents[intentId] = struct{}{}
			}
			p.pool[i].itemLock.Unlock()
			return i, nil
		}
		p.pool[i].itemLock.Unlock()
	}
	for {
		count := p.allocCounter.Load()
		if count >= int32(p.maxSize) {
			return -1, fmt.Errorf("can't reserve more keys, pool is full")
		}
		if p.allocCounter.CompareAndSwap(count, count + 1) {
			break
		}
	}
	for i := 0; i< p.maxSize; i++ {
		p.pool[i].itemLock.Lock()
		if len(p.pool[i].intents) == 0 {
			p.pool[i].collectionId = collectionId
			p.pool[i].intents[intentId] = struct{}{}
			p.pool[i].itemLock.Unlock()
			return i, nil
		}
		p.pool[i].itemLock.Unlock()
	}
	panic("Failed to find free pool item even though allocation tracking asserted it extists")
}

func (p *CollectionIdPool) GetIfAllocated(intentId string, collectionId string) (int, bool) {
	for i := 0; i < p.maxSize; i++ {
		p.pool[i].itemLock.Lock()
		_, owned := p.pool[i].intents[intentId]
		if owned && p.pool[i].collectionId == collectionId {
			p.pool[i].itemLock.Unlock()
			return i, true
		}
		p.pool[i].itemLock.Unlock()
	}
	return -1, false
}

// returns silently if key has already been freed
func (p *CollectionIdPool) Free(intentId string, collectionId string) {
	for i := 0; i < p.maxSize; i++ {
		p.pool[i].itemLock.Lock()
		_, owned := p.pool[i].intents[intentId]
		if owned && p.pool[i].collectionId == collectionId {
			delete(p.pool[i].intents, intentId)
			if len(p.pool[i].intents) == 0 {
				p.pool[i].collectionId = ""
				for {
					count := p.allocCounter.Load()
					if p.allocCounter.CompareAndSwap(count, count - 1) {
						break
					}
				}
			}
			p.pool[i].itemLock.Unlock()
			return
		}
		p.pool[i].itemLock.Unlock()
	}
}
