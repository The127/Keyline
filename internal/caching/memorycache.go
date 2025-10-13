package caching

import "sync"

type Cache[TKey comparable, TValue any] interface {
	TryGet(key TKey) (TValue, bool)
	Put(key TKey, value TValue)
	Clear()
}

type memoryCache[TKey comparable, TValue any] struct {
	mu     sync.RWMutex
	values map[TKey]TValue
}

func NewMemoryCache[TKey comparable, TValue any]() Cache[TKey, TValue] {
	return &memoryCache[TKey, TValue]{
		values: make(map[TKey]TValue),
	}
}

func (k *memoryCache[TKey, TValue]) TryGet(key TKey) (TValue, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	keyPair, ok := k.values[key]
	return keyPair, ok
}

func (k *memoryCache[TKey, TValue]) Put(key TKey, value TValue) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.values[key] = value
}

func (k *memoryCache[TKey, TValue]) Clear() {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.values = make(map[TKey]TValue)
}
