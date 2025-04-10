package syncx

import (
	"sync/atomic"
	"testing"
)

func TestMap_StoreAndLoad(t *testing.T) {
	m := &Map[string, int]{}
	key := "test"
	value := 42

	m.Store(key, value)
	got, ok := m.Load(key)

	if !ok {
		t.Error("Load returned false for existing key")
	}
	if got != value {
		t.Errorf("Load returned %v, want %v", got, value)
	}
}

func TestMap_LoadNonExistent(t *testing.T) {
	m := &Map[string, int]{}
	key := "nonexistent"

	got, ok := m.Load(key)

	if ok {
		t.Error("Load returned true for non-existent key")
	}
	if got != 0 {
		t.Errorf("Load returned %v, want 0", got)
	}
}

func TestMap_LoadAndDelete(t *testing.T) {
	m := &Map[string, int]{}
	key := "test"
	value := 42

	m.Store(key, value)
	got, loaded := m.LoadAndDelete(key)

	if !loaded {
		t.Error("LoadAndDelete returned false for existing key")
	}
	if got != value {
		t.Errorf("LoadAndDelete returned %v, want %v", got, value)
	}

	// Verify the key was deleted
	_, ok := m.Load(key)
	if ok {
		t.Error("Key still exists after LoadAndDelete")
	}
}

func TestMap_LoadOrStore(t *testing.T) {
	m := &Map[string, int]{}
	key := "test"
	value := 42

	// First store
	got, loaded := m.LoadOrStore(key, value)
	if loaded {
		t.Error("LoadOrStore returned true for first store")
	}
	if got != value {
		t.Errorf("LoadOrStore returned %v, want %v", got, value)
	}

	// Second store with same key
	newValue := 100
	got, loaded = m.LoadOrStore(key, newValue)
	if !loaded {
		t.Error("LoadOrStore returned false for existing key")
	}
	if got != value {
		t.Errorf("LoadOrStore returned %v, want %v", got, value)
	}
}

func TestMap_Range(t *testing.T) {
	m := &Map[string, int]{}
	items := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	// Store items
	for k, v := range items {
		m.Store(k, v)
	}

	// Count items using Range
	var count atomic.Int32
	m.Range(func(key string, value int) bool {
		count.Add(1)
		if items[key] != value {
			t.Errorf("Range: got value %v for key %v, want %v", value, key, items[key])
		}
		return true
	})

	if count.Load() != int32(len(items)) {
		t.Errorf("Range visited %v items, want %v", count.Load(), len(items))
	}
}

func TestMap_Delete(t *testing.T) {
	m := &Map[string, int]{}
	key := "test"
	value := 42

	m.Store(key, value)
	m.Delete(key)

	_, ok := m.Load(key)
	if ok {
		t.Error("Key still exists after Delete")
	}
}
