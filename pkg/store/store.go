//go:build js && wasm
// +build js,wasm

package store

type Store[T any] interface {
	Get() T
	Set(value T)
	Subscribe(callback func(T)) Unsubscriber
	Value() T
}

type Unsubscriber func()

type ReadonlyStore[T any] interface {
	Get() T
	Subscribe(callback func(T)) Unsubscriber
	Value() T
}
