//go:build js && wasm
// +build js,wasm

package store

import (
	"sync"
)

type Atom[T any] struct {
	value       T
	subscribers []func(T)
	mu          sync.RWMutex
	hmrKey      string
}

func NewAtom[T any](initial T) *Atom[T] {
	return &Atom[T]{
		value:       initial,
		subscribers: make([]func(T), 0),
	}
}

func NewAtomWithHMR[T any](initial T, hmrKey string) *Atom[T] {
	a := &Atom[T]{
		value:       initial,
		subscribers: make([]func(T), 0),
		hmrKey:      hmrKey,
	}
	a.loadFromHMR()
	return a
}

func (a *Atom[T]) Get() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.value
}

func (a *Atom[T]) Value() T {
	return a.Get()
}

func (a *Atom[T]) Set(value T) {
	a.mu.Lock()
	a.value = value
	subs := make([]func(T), len(a.subscribers))
	copy(subs, a.subscribers)
	a.mu.Unlock()

	a.saveToHMR()

	for _, callback := range subs {
		callback(value)
	}
}

func (a *Atom[T]) Update(fn func(T) T) {
	a.mu.Lock()
	a.value = fn(a.value)
	newValue := a.value
	subs := make([]func(T), len(a.subscribers))
	copy(subs, a.subscribers)
	a.mu.Unlock()

	a.saveToHMR()

	for _, callback := range subs {
		callback(newValue)
	}
}

func (a *Atom[T]) Subscribe(callback func(T)) Unsubscriber {
	a.mu.Lock()
	a.subscribers = append(a.subscribers, callback)
	index := len(a.subscribers) - 1
	a.mu.Unlock()

	return func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if index < len(a.subscribers) {
			a.subscribers = append(a.subscribers[:index], a.subscribers[index+1:]...)
		}
	}
}

func (a *Atom[T]) saveToHMR() {
	if a.hmrKey != "" {
		saveStateToHMR(a.hmrKey, a.value)
	}
}

func (a *Atom[T]) loadFromHMR() {
	if a.hmrKey != "" {
		if val, ok := loadStateFromHMR[T](a.hmrKey); ok {
			a.value = val
		}
	}
}
