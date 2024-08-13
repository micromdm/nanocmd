package test

import (
	"context"
	"testing"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/workflow"
)

func TestEventStatusStorage(t *testing.T, ctx context.Context, store storage.WorkflowStatusStorage) {
	_, err := store.RetrieveWorkflowStarted(ctx, "id.should.not.exist", "wfname.whaa")
	if err != nil {
		// should not error for a non-found item
		t.Fatal(err)
	}
}

func TestEventStorage(t *testing.T, store storage.EventSubscriptionStorage) {
	ctx := context.Background()

	evTest := &storage.EventSubscription{
		Event:        "Enrollment",
		Workflow:     "wf",
		Context:      "ctx",
		EventContext: "evCtx",
	}

	testEventData := func(t *testing.T, es *storage.EventSubscription) {
		if es == nil {
			t.Fatal("nil event subscription")
		}

		err := es.Validate()
		if err != nil {
			t.Fatalf("invalid test data")
		}

		if have, want := es.Event, evTest.Event; have != want {
			t.Errorf("[event] have: %v, want: %v", have, want)
		}

		if have, want := es.Workflow, evTest.Workflow; have != want {
			t.Errorf("[workflow] have: %v, want: %v", have, want)
		}

		if have, want := es.Context, evTest.Context; have != want {
			t.Errorf("[context] have: %v, want: %v", have, want)
		}

		if have, want := es.EventContext, evTest.EventContext; have != want {
			t.Errorf("[context] have: %v, want: %v", have, want)
		}
	}

	t.Run("testdata", func(t *testing.T) {
		testEventData(t, evTest)
	})

	err := store.StoreEventSubscription(ctx, "test", evTest)
	if err != nil {
		t.Fatal(err)
	}

	events, err := store.RetrieveEventSubscriptions(ctx, []string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(events), 1; have != want {
		t.Fatalf("have: %v, want: %v", have, want)
	}

	t.Run("retrieve-by-name", func(t *testing.T) {
		testEventData(t, events["test"])
	})

	eventsList, err := store.RetrieveEventSubscriptionsByEvent(ctx, workflow.EventEnrollment)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(eventsList), 1; have != want {
		t.Fatalf("have: %v, want: %v", have, want)
	}

	t.Run("retrieve-by-event", func(t *testing.T) {
		testEventData(t, eventsList[0])
	})
}
