package services

import "sync"

type Cache[TKey comparable, TValue any] interface {
	TryGet(key TKey) (TValue, bool)
	Put(key TKey, pair TValue)
	Clear()
}

type memoryCache[TKey comparable, TValue any] struct {
	mu       sync.RWMutex
	keyPairs map[TKey]TValue
}

func NewMemoryCache[TKey comparable, TValue any]() Cache[TKey, TValue] {
	return &memoryCache[TKey, TValue]{
		keyPairs: make(map[TKey]TValue),
	}
}

func (k *memoryCache[TKey, TValue]) TryGet(key TKey) (TValue, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	keyPair, ok := k.keyPairs[key]
	return keyPair, ok
}

func (k *memoryCache[TKey, TValue]) Put(key TKey, pair TValue) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.keyPairs[key] = pair
}

func (k *memoryCache[TKey, TValue]) Clear() {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.keyPairs = make(map[TKey]TValue)
}
