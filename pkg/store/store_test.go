//go:build js && wasm
// +build js,wasm

package store

import (
	"testing"
)

func TestAtomGetSet(t *testing.T) {
	atom := NewAtom(10)

	if got := atom.Get(); got != 10 {
		t.Errorf("Get() = %v, want %v", got, 10)
	}

	atom.Set(20)

	if got := atom.Get(); got != 20 {
		t.Errorf("After Set(20), Get() = %v, want %v", got, 20)
	}
}

func TestAtomValue(t *testing.T) {
	atom := NewAtom("test")

	if got := atom.Value(); got != "test" {
		t.Errorf("Value() = %v, want %v", got, "test")
	}
}

func TestAtomUpdate(t *testing.T) {
	atom := NewAtom(5)

	atom.Update(func(v int) int {
		return v * 2
	})

	if got := atom.Get(); got != 10 {
		t.Errorf("After Update, Get() = %v, want %v", got, 10)
	}
}

func TestAtomSubscribe(t *testing.T) {
	atom := NewAtom(0)
	called := false
	var receivedValue int

	unsub := atom.Subscribe(func(val int) {
		called = true
		receivedValue = val
	})
	defer unsub()

	atom.Set(42)

	if !called {
		t.Error("Subscribe callback was not called")
	}

	if receivedValue != 42 {
		t.Errorf("Subscribe callback received %v, want %v", receivedValue, 42)
	}
}

func TestAtomMultipleSubscribers(t *testing.T) {
	atom := NewAtom(0)

	calls1 := 0
	calls2 := 0

	unsub1 := atom.Subscribe(func(val int) {
		calls1++
	})
	defer unsub1()

	unsub2 := atom.Subscribe(func(val int) {
		calls2++
	})
	defer unsub2()

	atom.Set(1)
	atom.Set(2)

	if calls1 != 2 {
		t.Errorf("Subscriber 1 called %d times, want 2", calls1)
	}

	if calls2 != 2 {
		t.Errorf("Subscriber 2 called %d times, want 2", calls2)
	}
}

func TestAtomUnsubscribe(t *testing.T) {
	atom := NewAtom(0)

	calls := 0
	unsub := atom.Subscribe(func(val int) {
		calls++
	})

	atom.Set(1)
	unsub()
	atom.Set(2)

	if calls != 1 {
		t.Errorf("After unsubscribe, callback called %d times, want 1", calls)
	}
}

func TestMapStoreGetSet(t *testing.T) {
	m := NewMap(map[string]any{"name": "Alice", "age": 30})

	val := m.Get()
	if val["name"] != "Alice" {
		t.Errorf("Get()[name] = %v, want Alice", val["name"])
	}

	m.Set(map[string]any{"name": "Bob", "age": 25})

	val = m.Get()
	if val["name"] != "Bob" {
		t.Errorf("After Set, Get()[name] = %v, want Bob", val["name"])
	}
}

func TestMapStoreNilInit(t *testing.T) {
	m := NewMap(nil)

	val := m.Get()
	if val == nil {
		t.Error("NewMap(nil) returned nil map")
	}

	if len(val) != 0 {
		t.Errorf("NewMap(nil) returned map with %d items, want 0", len(val))
	}
}

func TestMapStoreGetKey(t *testing.T) {
	m := NewMap(map[string]any{"foo": "bar"})

	if got := m.GetKey("foo"); got != "bar" {
		t.Errorf("GetKey(foo) = %v, want bar", got)
	}

	if got := m.GetKey("missing"); got != nil {
		t.Errorf("GetKey(missing) = %v, want nil", got)
	}
}

func TestMapStoreSetKey(t *testing.T) {
	m := NewMap(map[string]any{"foo": "bar"})

	m.SetKey("foo", "baz")

	if got := m.GetKey("foo"); got != "baz" {
		t.Errorf("After SetKey, GetKey(foo) = %v, want baz", got)
	}

	m.SetKey("new", "value")

	if got := m.GetKey("new"); got != "value" {
		t.Errorf("After SetKey new key, GetKey(new) = %v, want value", got)
	}
}

func TestMapStoreDeleteKey(t *testing.T) {
	m := NewMap(map[string]any{"foo": "bar", "baz": "qux"})

	m.DeleteKey("foo")

	if got := m.GetKey("foo"); got != nil {
		t.Errorf("After DeleteKey, GetKey(foo) = %v, want nil", got)
	}

	if got := m.GetKey("baz"); got != "qux" {
		t.Errorf("DeleteKey affected other keys, GetKey(baz) = %v, want qux", got)
	}
}

func TestMapStoreSubscribe(t *testing.T) {
	m := NewMap(map[string]any{"count": 0})

	called := false
	var receivedValue map[string]any

	unsub := m.Subscribe(func(val map[string]any) {
		called = true
		receivedValue = val
	})
	defer unsub()

	m.SetKey("count", 5)

	if !called {
		t.Error("Subscribe callback was not called")
	}

	if receivedValue["count"] != 5 {
		t.Errorf("Subscribe callback received count=%v, want 5", receivedValue["count"])
	}
}

func TestMapStoreSubscribeOnSet(t *testing.T) {
	m := NewMap(map[string]any{"a": 1})

	calls := 0
	unsub := m.Subscribe(func(val map[string]any) {
		calls++
	})
	defer unsub()

	m.Set(map[string]any{"b": 2})

	if calls != 1 {
		t.Errorf("Subscribe callback called %d times, want 1", calls)
	}
}

func TestMapStoreSubscribeOnDelete(t *testing.T) {
	m := NewMap(map[string]any{"a": 1})

	calls := 0
	unsub := m.Subscribe(func(val map[string]any) {
		calls++
	})
	defer unsub()

	m.DeleteKey("a")

	if calls != 1 {
		t.Errorf("Subscribe callback called %d times after delete, want 1", calls)
	}
}

func TestComputedBasic(t *testing.T) {
	atom := NewAtom(5)
	computed := NewComputed(atom, func(v int) int {
		return v * 2
	})
	defer computed.Destroy()

	if got := computed.Get(); got != 10 {
		t.Errorf("Computed.Get() = %v, want 10", got)
	}

	atom.Set(10)

	if got := computed.Get(); got != 20 {
		t.Errorf("After source change, Computed.Get() = %v, want 20", got)
	}
}

func TestComputedValue(t *testing.T) {
	atom := NewAtom("hello")
	computed := NewComputed(atom, func(v string) int {
		return len(v)
	})
	defer computed.Destroy()

	if got := computed.Value(); got != 5 {
		t.Errorf("Computed.Value() = %v, want 5", got)
	}
}

func TestComputedSubscribe(t *testing.T) {
	atom := NewAtom(1)
	computed := NewComputed(atom, func(v int) int {
		return v * 10
	})
	defer computed.Destroy()

	called := false
	var receivedValue int

	unsub := computed.Subscribe(func(val int) {
		called = true
		receivedValue = val
	})
	defer unsub()

	atom.Set(5)

	if !called {
		t.Error("Computed subscribe callback was not called")
	}

	if receivedValue != 50 {
		t.Errorf("Computed subscribe received %v, want 50", receivedValue)
	}
}

func TestComputedChain(t *testing.T) {
	atom := NewAtom(2)
	doubled := NewComputed(atom, func(v int) int {
		return v * 2
	})
	defer doubled.Destroy()

	quadrupled := NewComputed(doubled, func(v int) int {
		return v * 2
	})
	defer quadrupled.Destroy()

	if got := quadrupled.Get(); got != 8 {
		t.Errorf("Chained computed.Get() = %v, want 8", got)
	}

	atom.Set(3)

	if got := quadrupled.Get(); got != 12 {
		t.Errorf("After source change, chained computed.Get() = %v, want 12", got)
	}
}

func TestComputedDestroy(t *testing.T) {
	atom := NewAtom(1)
	computed := NewComputed(atom, func(v int) int {
		return v * 2
	})

	initial := computed.Get()
	computed.Destroy()

	atom.Set(10)

	if got := computed.Get(); got != initial {
		t.Errorf("After Destroy, computed should not update, got %v", got)
	}
}

func TestAtomTypeString(t *testing.T) {
	atom := NewAtom("hello")

	if got := atom.Get(); got != "hello" {
		t.Errorf("String atom Get() = %v, want hello", got)
	}

	atom.Set("world")

	if got := atom.Get(); got != "world" {
		t.Errorf("After Set, string atom Get() = %v, want world", got)
	}
}

func TestAtomTypeBool(t *testing.T) {
	atom := NewAtom(true)

	if got := atom.Get(); got != true {
		t.Errorf("Bool atom Get() = %v, want true", got)
	}

	atom.Set(false)

	if got := atom.Get(); got != false {
		t.Errorf("After Set, bool atom Get() = %v, want false", got)
	}
}

func TestAtomTypeStruct(t *testing.T) {
	type User struct {
		Name string
		Age  int
	}

	atom := NewAtom(User{Name: "Alice", Age: 30})

	user := atom.Get()
	if user.Name != "Alice" || user.Age != 30 {
		t.Errorf("Struct atom Get() = %+v, want {Name:Alice Age:30}", user)
	}

	atom.Set(User{Name: "Bob", Age: 25})

	user = atom.Get()
	if user.Name != "Bob" || user.Age != 25 {
		t.Errorf("After Set, struct atom Get() = %+v, want {Name:Bob Age:25}", user)
	}
}

func TestAtomUpdateWithStruct(t *testing.T) {
	type Counter struct {
		Count int
	}

	atom := NewAtom(Counter{Count: 0})

	atom.Update(func(c Counter) Counter {
		c.Count++
		return c
	})

	if got := atom.Get().Count; got != 1 {
		t.Errorf("After Update, Count = %v, want 1", got)
	}
}

func TestConcurrentSubscribers(t *testing.T) {
	atom := NewAtom(0)

	const numSubs = 10
	calls := make([]int, numSubs)

	for i := 0; i < numSubs; i++ {
		idx := i
		atom.Subscribe(func(val int) {
			calls[idx]++
		})
	}

	atom.Set(1)

	for i := 0; i < numSubs; i++ {
		if calls[i] != 1 {
			t.Errorf("Subscriber %d called %d times, want 1", i, calls[i])
		}
	}
}

func TestMapStoreIsolation(t *testing.T) {
	m := NewMap(map[string]any{"value": 1})

	val1 := m.Get()
	val1["value"] = 999

	val2 := m.Get()
	if val2["value"] != 1 {
		t.Errorf("Map not isolated, val2[value] = %v, want 1", val2["value"])
	}
}
