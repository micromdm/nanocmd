// Package inmem implements a storage backend for the Profile subsystem backed by diskv.
package diskv

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is a storage backend for the Profile subsystem backed by diskv.
type Diskv struct {
	diskv *diskv.Diskv
}

// New creates a new initialized profile data store.
func New(path string) *Diskv {
	flatTransform := func(s string) []string { return []string{} }
	return &Diskv{
		diskv: diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "profile"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		}),
	}
}

// RetrieveProfileInfos implements the storage interface.
func (s *Diskv) RetrieveProfileInfos(ctx context.Context, names []string) (map[string]storage.ProfileInfo, error) {
	if len(names) < 1 {
		for name := range s.diskv.Keys(nil) {
			if strings.HasSuffix(name, ".identifier") {
				names = append(names, name[:len(name)-11])
			}
		}
	}
	ret := make(map[string]storage.ProfileInfo)
	for _, name := range names {
		if !s.diskv.Has(name + ".identifier") {
			return ret, fmt.Errorf("profile not found for %s: %w", name, storage.ErrProfileNotFound)
		}
		idBytes, err := s.diskv.Read(name + ".identifier")
		if err != nil {
			return ret, fmt.Errorf("reading identifier for %s: %w", name, err)
		}
		uuidBytes, err := s.diskv.Read(name + ".uuid")
		if err != nil {
			return ret, fmt.Errorf("reading uuid for %s: %w", name, err)
		}
		ret[name] = storage.ProfileInfo{
			Identifier: string(idBytes),
			UUID:       string(uuidBytes),
		}
	}
	return ret, nil
}

// RetrieveRawProfiles implements the storage interface.
func (s *Diskv) RetrieveRawProfiles(ctx context.Context, names []string) (map[string][]byte, error) {
	if len(names) < 1 {
		return nil, storage.ErrNoNames
	}
	ret := make(map[string][]byte)
	for _, name := range names {
		if !s.diskv.Has(name + ".raw") {
			continue
		}
		var err error
		if ret[name], err = s.diskv.Read(name + ".raw"); err != nil {
			return ret, fmt.Errorf("reading raw for %s: %w", name, err)
		}
	}
	return ret, nil
}

// StoreProfile implements the storage interface.
func (s *Diskv) StoreProfile(ctx context.Context, name string, info storage.ProfileInfo, raw []byte) error {
	err := s.diskv.Write(name+".raw", raw)
	if err != nil {
		return fmt.Errorf("writing raw: %w", err)
	}
	if err = s.diskv.Write(name+".identifier", []byte(info.Identifier)); err != nil {
		return fmt.Errorf("writing identifier: %w", err)
	}
	if err = s.diskv.Write(name+".uuid", []byte(info.UUID)); err != nil {
		return fmt.Errorf("writing uuid: %w", err)
	}
	return nil
}

// DeleteProfile implements the storage interface.
func (s *Diskv) DeleteProfile(ctx context.Context, name string) error {
	err := s.diskv.Erase(name + ".identifier")
	if err != nil {
		return fmt.Errorf("delete identifier for %s: %w", name, err)
	}
	if err := s.diskv.Erase(name + ".uuid"); err != nil {
		return fmt.Errorf("delete uuid for %s: %w", name, err)
	}
	if err := s.diskv.Erase(name + ".raw"); err != nil {
		return fmt.Errorf("delete raw for %s: %w", name, err)
	}
	return nil
}
