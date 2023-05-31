// Package inmem implements an in-memory storage backend for the Profile subsystem.
package inmem

import (
	"context"
	"fmt"
	"sync"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
)

type profile struct {
	info storage.ProfileInfo
	raw  []byte
}

// InMem is an in-memory storage backend for the Profile subsystem.
type InMem struct {
	m sync.RWMutex
	p map[string]profile
}

func New() *InMem {
	return &InMem{p: make(map[string]profile)}
}

// RetrieveProfileInfos implements the storage interface.
func (s *InMem) RetrieveProfileInfos(ctx context.Context, names []string) (map[string]storage.ProfileInfo, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	if len(names) < 1 {
		names = make([]string, 0, len(s.p))
		for key := range s.p {
			names = append(names, key)
		}
	}
	ret := make(map[string]storage.ProfileInfo)
	for _, name := range names {
		profile, ok := s.p[name]
		if !ok {
			return ret, fmt.Errorf("%w: %s", storage.ErrProfileNotFound, name)
		}
		ret[name] = profile.info
	}
	return ret, nil
}

// RetrieveRawProfiles implements the storage interface.
func (s *InMem) RetrieveRawProfiles(ctx context.Context, names []string) (map[string][]byte, error) {
	if len(names) < 1 {
		return nil, storage.ErrNoNames
	}
	s.m.RLock()
	defer s.m.RUnlock()
	ret := make(map[string][]byte)
	for _, name := range names {
		profile, ok := s.p[name]
		if !ok {
			return ret, fmt.Errorf("%w: %s", storage.ErrProfileNotFound, name)
		}
		ret[name] = profile.raw
	}
	return ret, nil
}

// StoreProfile implements the storage interface.
func (s *InMem) StoreProfile(ctx context.Context, name string, info storage.ProfileInfo, raw []byte) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.p[name] = profile{info: info, raw: raw}
	return nil
}

// DeleteProfile implements the storage interface.
func (s *InMem) DeleteProfile(ctx context.Context, name string) error {
	s.m.Lock()
	defer s.m.Unlock()
	_, ok := s.p[name]
	if !ok {
		return storage.ErrProfileNotFound
	}
	delete(s.p, name)
	return nil
}
