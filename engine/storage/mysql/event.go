package mysql

import (
	"context"
	"fmt"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/workflow"
)

// RetrieveEventSubscriptions retrieves event subscriptions by names.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveEventSubscriptions(ctx context.Context, names []string) (map[string]*storage.EventSubscription, error) {
	events, err := s.q.GetEventsByNames(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("get events by name: %w", err)
	}
	retEvents := make(map[string]*storage.EventSubscription)
	for _, event := range events {
		retEvents[event.EventName] = &storage.EventSubscription{
			Event:    event.EventType,
			Workflow: event.WorkflowName,
			Context:  event.Context.String,
		}
	}
	return retEvents, nil
}

// RetrieveEventSubscriptionsByEvent retrieves event subscriptions by event flag.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveEventSubscriptionsByEvent(ctx context.Context, f workflow.EventFlag) ([]*storage.EventSubscription, error) {
	events, err := s.q.GetEventsByType(ctx, f.String())
	if err != nil {
		return nil, fmt.Errorf("get events by type: %w", err)
	}
	var retEvents []*storage.EventSubscription
	for _, event := range events {
		retEvents = append(retEvents, &storage.EventSubscription{
			Event:    event.EventType,
			Workflow: event.WorkflowName,
			Context:  event.Context.String,
		})
	}
	return retEvents, nil
}

// StoreEventSubscription stores an event subscription.
// See the storage interface type for further docs.
func (s *MySQLStorage) StoreEventSubscription(ctx context.Context, name string, es *storage.EventSubscription) error {
	_, err := s.db.ExecContext(
		ctx,
		`
INSERT INTO wf_events
  (event_name, event_type, workflow_name, context)
VALUES
  (?, ?, ?, ?) AS new
ON DUPLICATE KEY
UPDATE
  workflow_name = new.workflow_name,
  event_type = new.event_type,
  context = new.context;`,
		name,
		es.Event,
		es.Workflow,
		sqlNullString(es.Context),
	)
	return err
}

// DeleteEventSubscription removes an event subscription.
// See the storage interface type for further docs.
func (s *MySQLStorage) DeleteEventSubscription(ctx context.Context, name string) error {
	return s.q.RemoveEvent(ctx, name)
}
