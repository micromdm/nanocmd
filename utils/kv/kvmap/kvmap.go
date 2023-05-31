// Package kvmap implements an in-memory key-value store backed by a Go map.
package kvmap

import (
	"context"
	"fmt"
	"sync"
)

// KVMap is an in-memory key-value store backed by a Go map.
type KVMap struct {
	mu sync.RWMutex
	m  map[string][]byte
}

func NewBucket() *KVMap {
	return &KVMap{m: make(map[string][]byte)}
}

func (s *KVMap) Get(_ context.Context, k string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[k]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", k)
	}
	return v, nil
}

func (s *KVMap) Set(_ context.Context, k string, v []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
	return nil
}

func (s *KVMap) Has(_ context.Context, k string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.m[k]
	return ok, nil
}

func (s *KVMap) Delete(_ context.Context, k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, k)
	return nil
}

// Keys returns the keys in this bucket.
// Note that this function spawns a go routine that keeps a read lock on
// the internal map. This means that if you attempt to write to the map
// while you're, say, iterating over a keys list you will likely deadlock.
func (s *KVMap) Keys(cancel <-chan struct{}) <-chan string {
	r := make(chan string)
	go func() {
		s.mu.RLock()
		defer s.mu.RUnlock()
		defer close(r)
		for k := range s.m {
			select {
			case <-cancel:
				return
			case r <- k:
			}
		}
	}()
	return r
}
