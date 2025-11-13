//go:build js && wasm
// +build js,wasm

package store

import (
	"sync"
)

type MapStore struct {
	value       map[string]any
	subscribers []func(map[string]any)
	mu          sync.RWMutex
	hmrKey      string
}

func NewMap(initial map[string]any) *MapStore {
	if initial == nil {
		initial = make(map[string]any)
	}
	return &MapStore{
		value:       copyMap(initial),
		subscribers: make([]func(map[string]any), 0),
	}
}

func NewMapWithHMR(initial map[string]any, hmrKey string) *MapStore {
	if initial == nil {
		initial = make(map[string]any)
	}
	m := &MapStore{
		value:       copyMap(initial),
		subscribers: make([]func(map[string]any), 0),
		hmrKey:      hmrKey,
	}
	m.loadFromHMR()
	return m
}

func (m *MapStore) Get() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return copyMap(m.value)
}

func (m *MapStore) Value() map[string]any {
	return m.Get()
}

func (m *MapStore) GetKey(key string) any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.value[key]
}

func (m *MapStore) Set(value map[string]any) {
	m.mu.Lock()
	m.value = copyMap(value)
	subs := make([]func(map[string]any), len(m.subscribers))
	copy(subs, m.subscribers)
	valueCopy := copyMap(m.value)
	m.mu.Unlock()

	m.saveToHMR()

	for _, callback := range subs {
		callback(valueCopy)
	}
}

func (m *MapStore) SetKey(key string, value any) {
	m.mu.Lock()
	m.value[key] = value
	subs := make([]func(map[string]any), len(m.subscribers))
	copy(subs, m.subscribers)
	valueCopy := copyMap(m.value)
	m.mu.Unlock()

	m.saveToHMR()

	for _, callback := range subs {
		callback(valueCopy)
	}
}

func (m *MapStore) DeleteKey(key string) {
	m.mu.Lock()
	delete(m.value, key)
	subs := make([]func(map[string]any), len(m.subscribers))
	copy(subs, m.subscribers)
	valueCopy := copyMap(m.value)
	m.mu.Unlock()

	m.saveToHMR()

	for _, callback := range subs {
		callback(valueCopy)
	}
}

func (m *MapStore) Subscribe(callback func(map[string]any)) Unsubscriber {
	m.mu.Lock()
	m.subscribers = append(m.subscribers, callback)
	index := len(m.subscribers) - 1
	m.mu.Unlock()

	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if index < len(m.subscribers) {
			m.subscribers = append(m.subscribers[:index], m.subscribers[index+1:]...)
		}
	}
}

func (m *MapStore) saveToHMR() {
	if m.hmrKey != "" {
		saveStateToHMR(m.hmrKey, m.value)
	}
}

func (m *MapStore) loadFromHMR() {
	if m.hmrKey != "" {
		if val, ok := loadStateFromHMR[map[string]any](m.hmrKey); ok {
			m.value = val
		}
	}
}

func copyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
