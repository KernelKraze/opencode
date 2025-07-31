// Package memoization implement a simple memoization cache. It's designed to
// improve performance in textarea.
package textarea

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

// Hasher is an interface that requires a Hash method. The Hash method is
// expected to return a string representation of the hash of the object.
type Hasher interface {
	Hash() string
}

// entry is a struct that holds a key-value pair. It is used as an element
// in the evictionList of the MemoCache.
type entry[T any] struct {
	key   string
	value T
}

// MemoCache is a struct that represents a cache with a set capacity. It
// uses an LRU (Least Recently Used) eviction policy. It is safe for
// concurrent use.
type MemoCache[H Hasher, T any] struct {
	capacity      int
	mutex         sync.Mutex
	cache         map[string]*list.Element // The cache holding the results
	evictionList  *list.List               // A list to keep track of the order for LRU
	hashableItems map[string]T             // This map keeps track of the original hashable items (optional)
}

// NewMemoCache is a function that creates a new MemoCache with a given
// capacity. It returns a pointer to the created MemoCache.
func NewMemoCache[H Hasher, T any](capacity int) *MemoCache[H, T] {
	return &MemoCache[H, T]{
		capacity:      capacity,
		cache:         make(map[string]*list.Element),
		evictionList:  list.New(),
		hashableItems: make(map[string]T),
	}
}

// Capacity is a method that returns the capacity of the MemoCache.
func (m *MemoCache[H, T]) Capacity() int {
	return m.capacity
}

// Size is a method that returns the current size of the MemoCache. It is
// the number of items currently stored in the cache.
func (m *MemoCache[H, T]) Size() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.evictionList.Len()
}

// Get is a method that returns the value associated with the given
// hashable item in the MemoCache. If there is no corresponding value, the
// method returns nil.
func (m *MemoCache[H, T]) Get(h H) (T, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hashedKey := h.Hash()
	if element, found := m.cache[hashedKey]; found {
		m.evictionList.MoveToFront(element)
		return element.Value.(*entry[T]).value, true
	}
	var result T
	return result, false
}

// Set is a method that sets the value for the given hashable item in the
// MemoCache. If the cache is at capacity, it evicts the least recently
// used item before adding the new item.
func (m *MemoCache[H, T]) Set(h H, value T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hashedKey := h.Hash()
	if element, found := m.cache[hashedKey]; found {
		m.evictionList.MoveToFront(element)
		element.Value.(*entry[T]).value = value
		return
	}

	// Check if the cache is at capacity
	if m.evictionList.Len() >= m.capacity {
		// Evict the least recently used item from the cache
		toEvict := m.evictionList.Back()
		if toEvict != nil {
			evictedEntry := m.evictionList.Remove(toEvict).(*entry[T])
			delete(m.cache, evictedEntry.key)
			delete(m.hashableItems, evictedEntry.key) // if you're keeping track of original items
		}
	}

	// Add the value to the cache and the evictionList
	newEntry := &entry[T]{
		key:   hashedKey,
		value: value,
	}
	element := m.evictionList.PushFront(newEntry)
	m.cache[hashedKey] = element
	m.hashableItems[hashedKey] = value // if you're keeping track of original items
}

// HashVersion represents the cache version for compatibility
const (
	HashVersionV1 = "v1" // SHA256 (legacy)
	HashVersionV2 = "v2" // FNV-1a (current)
)

// VersionedHasher extends Hasher with version-aware hashing
type VersionedHasher interface {
	Hasher
	HashWithVersion(version string) string
}

// HString is a type that implements the VersionedHasher interface for strings.
type HString string

// Hash returns the current version hash (FNV-1a for performance)
func (h HString) Hash() string {
	return h.HashWithVersion(HashVersionV2)
}

// HashWithVersion returns hash based on specified version for compatibility
func (h HString) HashWithVersion(version string) string {
	data := []byte(h)
	switch version {
	case HashVersionV1:
		// Legacy SHA256 for backward compatibility
		hash := sha256.Sum256(data)
		return fmt.Sprintf("v1:%x", hash)
	case HashVersionV2:
		// Fast FNV-1a for performance
		hasher := fnv.New64a()
		hasher.Write(data)
		return fmt.Sprintf("v2:%x", hasher.Sum64())
	default:
		// Default to current version
		return h.HashWithVersion(HashVersionV2)
	}
}

// HInt is a type that implements the VersionedHasher interface for integers.
type HInt int

// Hash returns the current version hash (FNV-1a for performance)
func (h HInt) Hash() string {
	return h.HashWithVersion(HashVersionV2)
}

// HashWithVersion returns hash based on specified version for compatibility
func (h HInt) HashWithVersion(version string) string {
	data := []byte(fmt.Sprintf("%d", h))
	switch version {
	case HashVersionV1:
		// Legacy SHA256 for backward compatibility
		hash := sha256.Sum256(data)
		return fmt.Sprintf("v1:%x", hash)
	case HashVersionV2:
		// Fast FNV-1a for performance
		hasher := fnv.New64a()
		hasher.Write(data)
		return fmt.Sprintf("v2:%x", hasher.Sum64())
	default:
		// Default to current version
		return h.HashWithVersion(HashVersionV2)
	}
}

// MigrationAwareCache extends MemoCache with version migration support
type MigrationAwareCache[H VersionedHasher, T any] struct {
	*MemoCache[H, T]
	migrationDeadline time.Time
}

// NewMigrationAwareCache creates a cache that supports hash version migration
func NewMigrationAwareCache[H VersionedHasher, T any](capacity int, migrationPeriod time.Duration) *MigrationAwareCache[H, T] {
	return &MigrationAwareCache[H, T]{
		MemoCache:         NewMemoCache[H, T](capacity),
		migrationDeadline: time.Now().Add(migrationPeriod),
	}
}

// Get attempts to retrieve value using current hash, falls back to legacy hash during migration
func (m *MigrationAwareCache[H, T]) Get(h H) (T, bool) {
	// Try current version first (FNV-1a)
	if value, found := m.MemoCache.Get(h); found {
		return value, true
	}

	// During migration period, also check legacy hash (SHA256)
	if time.Now().Before(m.migrationDeadline) {
		// Create a temporary hasher for legacy lookup
		legacyKey := h.HashWithVersion(HashVersionV1)
		m.mutex.Lock()
		defer m.mutex.Unlock()

		if element, found := m.cache[legacyKey]; found {
			// Found in legacy format, migrate to new format
			value := element.Value.(*entry[T]).value

			// Remove from legacy location
			m.evictionList.Remove(element)
			delete(m.cache, legacyKey)
			delete(m.hashableItems, legacyKey)

			// Add to new location (this will use current hash version)
			m.mutex.Unlock() // Unlock before calling Set to avoid deadlock
			m.Set(h, value)
			m.mutex.Lock() // Re-lock for defer

			return value, true
		}
	}

	var result T
	return result, false
}
