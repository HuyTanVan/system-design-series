package store

import (
	"sync"
	"time"
)

type Store struct {
	mu     sync.RWMutex
	data   map[string]string // SET/GET
	ldata  *LRUCache
	hdata  map[string]map[string]string // HSET/HGET
	expiry map[string]time.Time
}

// Store is an in-memory key-value store that supports basic string operations and hash operations.
func NewStore() *Store {
	lru := NewLRUCache(100) // 100 MB
	s := &Store{
		data:   make(map[string]string),
		ldata:  lru,
		hdata:  make(map[string]map[string]string),
		expiry: make(map[string]time.Time),
	}
	// Start the cleanup goroutine to remove expired keys
	go s.cleanup()
	return s
}

// https://redis.io/docs/latest/commands/set/
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// s.data[key] = value // regular
	s.ldata.Set(key, value) // LRU
}

// https://redis.io/docs/latest/commands/get/
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isExpired(key) {
		return "", false
	}
	// val, ok := s.data[key] // regular
	val, ok := s.ldata.Get(key) // LRU
	return val, ok
}

// https://redis.io/docs/latest/commands/del/
// Note: Redis allows deleting multiple keys at once.
func (s *Store) Del(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	// if _, ok := s.data[key]; !ok {  // regular
	// 	return 0
	// }
	// delete(s.data, key)
	if _, ok := s.ldata.Get(key); !ok {
		return 0
	}
	s.ldata.Del(key)
	return 1
}

// Core commands for Hash operations (HSET, HGET, HDEL, HGETALL, HEXISTS, HLEN)
// https://redis.io/docs/latest/commands/hset/
func (s *Store) HSet(key, field, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.hdata[key]; !ok {
		s.hdata[key] = make(map[string]string)
	}
	_, exists := s.hdata[key][field]
	s.hdata[key][field] = value
	if exists {
		return 0
	}
	return 1
}

// https://redis.io/docs/latest/commands/hget/
func (s *Store) HGet(key, field string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.hdata[key]; !ok {
		return "", false
	}
	val, ok := s.hdata[key][field]
	return val, ok
}

// https://redis.io/docs/latest/commands/hdel/
func (s *Store) HDel(key, field string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.hdata[key]; !exists {
		return 0
	}
	if _, exists := s.hdata[key][field]; !exists {
		return 0
	}
	delete(s.hdata[key], field)
	return 1
}

// https://redis.io/docs/latest/commands/hgetall/
func (s *Store) HGetAll(key string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.hdata[key]; !ok {
		return nil
	}
	return s.hdata[key]
}

// https://redis.io/docs/latest/commands/hlen/
func (s *Store) Hlen(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.hdata[key]; !ok {
		return 0
	}
	return len(s.hdata[key])
}

// https://redis.io/docs/latest/commands/hexists/
func (s *Store) HExists(key, field string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.hdata[key]; !ok {
		return 0
	}
	if _, ok := s.hdata[key][field]; !ok {
		return 0
	}
	return 1
}
