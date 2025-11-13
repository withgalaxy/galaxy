//go:build js && wasm
// +build js,wasm

package store

import (
	"sync"
)

type Computed[T any] struct {
	value       T
	subscribers []func(T)
	mu          sync.RWMutex
	unsub       Unsubscriber
}

type ReadableStore[T any] interface {
	Get() T
	Subscribe(callback func(T)) Unsubscriber
}

func NewComputed[S any, T any](source ReadableStore[S], transform func(S) T) *Computed[T] {
	c := &Computed[T]{
		value:       transform(source.Get()),
		subscribers: make([]func(T), 0),
	}

	c.unsub = source.Subscribe(func(val S) {
		c.mu.Lock()
		c.value = transform(val)
		newValue := c.value
		subs := make([]func(T), len(c.subscribers))
		copy(subs, c.subscribers)
		c.mu.Unlock()

		for _, callback := range subs {
			callback(newValue)
		}
	})

	return c
}

func (c *Computed[T]) Get() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (c *Computed[T]) Value() T {
	return c.Get()
}

func (c *Computed[T]) Subscribe(callback func(T)) Unsubscriber {
	c.mu.Lock()
	c.subscribers = append(c.subscribers, callback)
	index := len(c.subscribers) - 1
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if index < len(c.subscribers) {
			c.subscribers = append(c.subscribers[:index], c.subscribers[index+1:]...)
		}
	}
}

func (c *Computed[T]) Destroy() {
	if c.unsub != nil {
		c.unsub()
	}
}
