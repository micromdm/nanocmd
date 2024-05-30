// Package kv defines an interface for key-value store.
package kv

import (
	"context"
	"fmt"
)

// Bucket defines basic CRUD operations for key-value pairs in a single "namespace."
type Bucket interface {
	Get(ctx context.Context, k string) (v []byte, err error)
	Set(ctx context.Context, k string, v []byte) error
	Has(ctx context.Context, k string) (found bool, err error)
	Delete(ctx context.Context, k string) error
}

// TraversingBucket allows us to get a list of the keys in the bucket as well.
type TraversingBucket interface {
	Bucket
	// Keys returns the unordered keys in the bucket
	Keys(cancel <-chan struct{}) <-chan string
}

// SetMap iterates over m to set the keys in b and returns any error.
func SetMap(ctx context.Context, b Bucket, m map[string][]byte) error {
	var err error
	for k, v := range m {
		if err = b.Set(ctx, k, v); err != nil {
			return fmt.Errorf("setting %s: %w", k, err)
		}
	}
	return nil
}

// SetMap iterates over keys to get the values in b and returns any error.
func GetMap(ctx context.Context, b Bucket, keys []string) (map[string][]byte, error) {
	var err error
	ret := make(map[string][]byte)
	for _, k := range keys {
		if ret[k], err = b.Get(ctx, k); err != nil {
			return ret, fmt.Errorf("getting %s: %w", k, err)
		}
	}
	return ret, nil
}

// DeleteSlice deletes s keys from b.
func DeleteSlice(ctx context.Context, b Bucket, s []string) error {
	var err error
	for _, i := range s {
		if err = b.Delete(ctx, i); err != nil {
			return fmt.Errorf("deleting %s: %w", i, err)
		}
	}
	return nil
}
