// Package kv implements a profile storage backend using key-value storage.
package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyPfxUUID = "uuid."
	keyPfxID   = "id."
	keyPfxRaw  = "raw."
)

// KV is a profile storage backend using key-value storage.
type KV struct {
	b kv.KeysPrefixTraversingBucket
}

func New(b kv.KeysPrefixTraversingBucket) *KV {
	return &KV{b: b}
}

// RetrieveProfileInfos returns the profile metadata in the key-value store by name.
// Will return all keys.
func (s *KV) RetrieveProfileInfos(ctx context.Context, names []string) (map[string]storage.ProfileInfo, error) {
	if len(names) < 1 {
		for k := range s.b.KeysPrefix(ctx, keyPfxID, nil) {
			names = append(names, k[len(keyPfxID):])
		}
	}

	r := make(map[string]storage.ProfileInfo)
	for _, name := range names {
		id, err := s.b.Get(ctx, keyPfxID+name)
		if errors.Is(err, kv.ErrKeyNotFound) {
			return r, fmt.Errorf("%w: %s: %v", storage.ErrProfileNotFound, name, err)
		} else if err != nil {
			return r, err
		}

		uuid, err := s.b.Get(ctx, keyPfxUUID+name)
		if errors.Is(err, kv.ErrKeyNotFound) {
			return r, fmt.Errorf("%w: %s: %v", storage.ErrProfileNotFound, name, err)
		} else if err != nil {
			return r, err
		}

		r[name] = storage.ProfileInfo{
			Identifier: string(id),
			UUID:       string(uuid),
		}
	}
	return r, nil
}

// RetrieveRawProfiles returns the raw profile bytes in the key-value store by name.
func (s *KV) RetrieveRawProfiles(ctx context.Context, names []string) (map[string][]byte, error) {
	if len(names) < 1 {
		return nil, storage.ErrNoNames
	}
	r := make(map[string][]byte)
	for _, name := range names {
		profile, err := s.b.Get(ctx, keyPfxRaw+name)
		if errors.Is(err, kv.ErrKeyNotFound) {
			return r, fmt.Errorf("%w: %s: %v", storage.ErrProfileNotFound, name, err)
		} else if err != nil {
			return r, err
		}
		r[name] = profile
	}
	return r, nil
}

// StoreProfile stores a raw profile and associated info in the key-value store by name.
func (s *KV) StoreProfile(ctx context.Context, name string, info storage.ProfileInfo, raw []byte) error {
	return kv.SetMap(ctx, s.b, map[string][]byte{
		keyPfxID + name:   []byte(info.Identifier),
		keyPfxUUID + name: []byte(info.UUID),
		keyPfxRaw + name:  raw,
	})
}

// DeleteProfile deletes a profile from the key-value store by name.
func (s *KV) DeleteProfile(ctx context.Context, name string) error {
	return kv.DeleteSlice(ctx, s.b, []string{
		keyPfxID + name,
		keyPfxUUID + name,
		keyPfxRaw + name,
	})
}
