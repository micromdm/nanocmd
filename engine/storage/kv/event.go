package kv

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/workflow"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keySfxEventFlag         = ".flag" // contains a strconv integer
	keySfxEventWorkflow     = ".name"
	keySfxEventContext      = ".ctx"
	keySfxEventEventContext = ".evctx"
)

type kvEventSubscription struct {
	*storage.EventSubscription
}

func (es *kvEventSubscription) set(ctx context.Context, b kv.Bucket, name string) error {
	if es == nil || name == "" {
		return errors.New("invalid")
	}
	err := es.Validate()
	if err != nil {
		return fmt.Errorf("validating: %w", err)
	}
	esMap := map[string][]byte{
		name + keySfxEventWorkflow: []byte(es.Workflow),
		name + keySfxEventFlag:     []byte(strconv.Itoa(int(workflow.EventFlagForString(es.Event)))),
	}
	if len(es.Context) > 0 {
		esMap[name+keySfxEventContext] = []byte(es.Context)
	}
	if len(es.EventContext) > 0 {
		esMap[name+keySfxEventEventContext] = []byte(es.EventContext)
	}
	return kv.SetMap(ctx, b, esMap)
}

func (es *kvEventSubscription) get(ctx context.Context, b kv.Bucket, name string) error {
	if es == nil || name == "" {
		return errors.New("invalid")
	}
	if es.EventSubscription == nil {
		es.EventSubscription = new(storage.EventSubscription)
	}
	esMap, err := kv.GetMap(ctx, b, []string{
		name + keySfxEventWorkflow,
		name + keySfxEventFlag,
	})
	if err != nil {
		return err
	}
	es.Workflow = string(esMap[name+keySfxEventWorkflow])
	eventFlag, err := strconv.Atoi(string(esMap[name+keySfxEventFlag]))
	if err != nil {
		return fmt.Errorf("getting event flag: %w", err)
	}
	es.Event = workflow.EventFlag(eventFlag).String()
	if ok, err := b.Has(ctx, name+keySfxEventContext); err != nil {
		return fmt.Errorf("checking event context: %w", err)
	} else if ok {
		if ctxBytes, err := b.Get(ctx, name+keySfxEventContext); err != nil {
			return fmt.Errorf("getting event context: %w", err)
		} else {
			es.Context = string(ctxBytes)
		}
	}
	if ok, err := b.Has(ctx, name+keySfxEventEventContext); err != nil {
		return fmt.Errorf("checking event event_context: %w", err)
	} else if ok {
		if evCtxBytes, err := b.Get(ctx, name+keySfxEventEventContext); err != nil {
			return fmt.Errorf("getting event event_context: %w", err)
		} else {
			es.EventContext = string(evCtxBytes)
		}
	}
	return nil
}

func (s *KV) RetrieveEventSubscriptions(ctx context.Context, names []string) (map[string]*storage.EventSubscription, error) {
	if len(names) < 1 {
		return nil, errors.New("no names specified")
	}
	ret := make(map[string]*storage.EventSubscription)
	for _, name := range names {
		wrapped := new(kvEventSubscription)
		if err := wrapped.get(ctx, s.eventStore, name); err != nil {
			return ret, fmt.Errorf("getting event subscription record for %s: %w", name, err)
		}
		ret[name] = wrapped.EventSubscription
	}
	return ret, nil
}

func kvFindEventSubNamesByEvent(ctx context.Context, b kv.KeysPrefixTraversingBucket, f workflow.EventFlag) ([]string, error) {
	var names []string

	// this.. is not very efficient. perhaps it would be better to
	// make a specific bucket/index for this.
	for k := range b.Keys(ctx, nil) {
		if !strings.HasSuffix(k, keySfxEventFlag) {
			continue
		}
		flagBytes, err := b.Get(ctx, k)
		if err != nil {
			return nil, err
		}
		eventFlag, err := strconv.Atoi(string(flagBytes))
		if err != nil {
			continue
		}
		if eventFlag != int(f) {
			continue
		}
		names = append(names, k[:len(k)-len(keySfxEventFlag)])
	}
	return names, nil
}

func (s *KV) RetrieveEventSubscriptionsByEvent(ctx context.Context, f workflow.EventFlag) ([]*storage.EventSubscription, error) {
	if f < 1 {
		return nil, errors.New("invalid event flag")
	}
	names, err := kvFindEventSubNamesByEvent(ctx, s.eventStore, f)
	if err != nil {
		return nil, fmt.Errorf("finding event subscriptions: %w", err)
	}
	var ret []*storage.EventSubscription
	for _, name := range names {
		es := new(kvEventSubscription)
		if err = es.get(ctx, s.eventStore, name); err != nil {
			return ret, fmt.Errorf("getting event subscription for %s: %w", name, err)
		}
		ret = append(ret, es.EventSubscription)
	}
	return ret, nil
}

func (s *KV) StoreEventSubscription(ctx context.Context, name string, es *storage.EventSubscription) error {
	wrapped := &kvEventSubscription{EventSubscription: es}
	if err := wrapped.set(ctx, s.eventStore, name); err != nil {
		return fmt.Errorf("setting event subscription record for %s: %w", name, err)
	}
	return nil
}

func (s *KV) DeleteEventSubscription(ctx context.Context, name string) error {
	return kvDeleteKeysIfExists(ctx, s.eventStore, []string{
		name + keySfxEventFlag,
		name + keySfxEventWorkflow,
		name + keySfxEventContext,
		name + keySfxEventEventContext,
	})
}
