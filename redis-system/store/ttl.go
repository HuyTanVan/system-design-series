package store

import (
	"time"
)

// // Set a key with a TTL. After the TTL expires, the key will be automatically deleted.
// func (s *Store) SetWithTTL(key, value string, ttl time.Duration) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	s.data[key] = value                 // key1 -> value1
// 	s.expiry[key] = time.Now().Add(ttl) // key1 -> time.Now() + ttl
// }

// https://redis.io/docs/latest/commands/expire/
// Set an expiry time on a key. After the expiry time, the key will be automatically deleted. Returns 1 if the timeout was set, 0 if key does not exist.
func (s *Store) SetExpiry(key string, ttl time.Duration) int {
	// fmt.Println("SetExpiry called", key, ttl)
	s.mu.Lock()
	// fmt.Println("lock acquired")
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		// fmt.Println("key not found")
		return 0
	}
	s.expiry[key] = time.Now().Add(ttl)
	// fmt.Println("expiry set")
	return 1
}

// https://redis.io/docs/latest/commands/ttl/
// Returns the remaining TTL for a key. Returns -2 if key does not exist, -1 if the key exists but has no associated expire.
func (s *Store) TTL(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.expiry[key]
	if !ok {
		return -2 // key does not exist
	}
	remaining := int(time.Until(exp).Seconds())
	if remaining <= 0 {
		return -1 // key exists but no expire is set
	}
	return remaining
}

// Checks if a key is expired.
func (s *Store) isExpired(key string) bool {
	exp, ok := s.expiry[key]
	if !ok {
		return false
	}
	return time.Now().After(exp)
}

// cleanup should be called in background, deletes expired keys every second
func (s *Store) cleanup() {
	for {
		time.Sleep(time.Second) // 1s
		s.mu.Lock()
		for key := range s.expiry {
			if time.Now().After(s.expiry[key]) {
				delete(s.data, key)
				delete(s.expiry, key)
			}
		}
		s.mu.Unlock()
	}
}
