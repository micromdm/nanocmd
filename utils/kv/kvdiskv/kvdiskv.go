// Package kvdiskv wraps diskv to a standard interface for a key-value store.
package kvdiskv

import (
	"context"

	"github.com/peterbourgon/diskv/v3"
)

// KVDiskv wraps a diskv object to implement an on-disk key-value store.
type KVDiskv struct {
	diskv *diskv.Diskv
}

func NewBucket(dv *diskv.Diskv) *KVDiskv {
	return &KVDiskv{diskv: dv}
}

func (s *KVDiskv) Get(_ context.Context, k string) ([]byte, error) {
	return s.diskv.Read(k)
}

func (s *KVDiskv) Set(_ context.Context, k string, v []byte) error {
	return s.diskv.Write(k, v)
}

func (s *KVDiskv) Has(_ context.Context, k string) (bool, error) {
	return s.diskv.Has(k), nil
}

func (s *KVDiskv) Delete(_ context.Context, k string) error {
	return s.diskv.Erase(k)
}

func (s *KVDiskv) Keys(cancel <-chan struct{}) <-chan string {
	return s.diskv.Keys(cancel)
}
