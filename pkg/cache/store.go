// Package cache provides a thread-safe in-memory counter store used for rate limiting.
package cache

import (
	"sync"
	"time"
)

// Store is a thread-safe in-memory counter store used for rate limiting.
type Store struct {
	mu      sync.Mutex
	entries map[string]*entry
}

type entry struct {
	count     int
	expiresAt time.Time
}

// NewStore returns an initialised Store.
func NewStore() *Store {
	return &Store{entries: make(map[string]*entry)}
}

// Incr increments the counter for key and returns the new value.
// If no entry exists or the existing entry has expired, a new counter starting
// at 1 is created.
func (s *Store) Incr(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entries[key]
	if !ok || (!e.expiresAt.IsZero() && time.Now().After(e.expiresAt)) {
		s.entries[key] = &entry{count: 1}
		return 1
	}

	e.count++
	return e.count
}

// Expire sets the TTL for key. If the key does not exist the call is a no-op.
func (s *Store) Expire(key string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.entries[key]; ok {
		e.expiresAt = time.Now().Add(ttl)
	}
}

// IncrWithExpire increments the counter for key and, on the first increment,
// sets the TTL to ttl. Returns the new counter value.
func (s *Store) IncrWithExpire(key string, ttl time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok || (!e.expiresAt.IsZero() && time.Now().After(e.expiresAt)) {
		s.entries[key] = &entry{count: 1, expiresAt: time.Now().Add(ttl)}
		return 1
	}
	e.count++
	return e.count
}
