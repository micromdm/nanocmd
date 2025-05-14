// Package kv implements a cmdplan storage backend using JSON with key-value storage.
package kv

import (
	"context"
	"encoding/json"

	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

// KV is a cmdplan storage backend using JSON with key-value storage.
type KV struct {
	b kv.Bucket
}

func New(b kv.Bucket) *KV {
	return &KV{b: b}
}

// RetrieveCMDPlan unmarshals the JSON stored using name and returns the command plan.
func (s *KV) RetrieveCMDPlan(ctx context.Context, name string) (*storage.CMDPlan, error) {
	raw, err := s.b.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	cmdPlan := new(storage.CMDPlan)
	return cmdPlan, json.Unmarshal(raw, cmdPlan)
}

// StoreCMDPlan marshals p into JSON and stores it using name.
func (s *KV) StoreCMDPlan(ctx context.Context, name string, p *storage.CMDPlan) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return s.b.Set(ctx, name, raw)
}

// DeleteCMDPlan deletes the JSON stored using name.
func (s *KV) DeleteCMDPlan(ctx context.Context, name string) error {
	return s.b.Delete(ctx, name)
}
