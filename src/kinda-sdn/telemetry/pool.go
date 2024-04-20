package telemetry

import (
	"fmt"
	"sync/atomic"
)

type poolItem struct {
	key string
	counter *atomic.Int32
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
		pool[i] = poolItem{key: "", counter: counter}
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
// or error if there are no more free slots
func (p *CollectionIdPool) AllocOrGet(key string) (int, error) {
	for i := 0; i < p.maxSize; i++ {
		if p.pool[i].counter.Load() > 0 && p.pool[i].key == key {
			return i, nil
		}
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
		if ok := p.pool[i].counter.CompareAndSwap(0, 1); ok {
			p.pool[i].key = key
			return i, nil	
		}
	}
	panic("Failed to find free pool item even though allocation tracking asserted it extists")
}

func (p *CollectionIdPool) Free(key string) {
	found := false
	for i := 0; i < p.maxSize; i++ {
		if p.pool[i].counter.Load() > 0 && p.pool[i].key == key {
			p.pool[i].key = ""
			p.pool[i].counter.Store(0)
			found = true
		}
	}
	if !found {
		return
	}
	for {
		count := p.allocCounter.Load()
		if p.allocCounter.CompareAndSwap(count, count - 1) {
			break
		}
	}
}
