package test

import (
	"context"
	"sync"

	"github.com/micromdm/nanocmd/workflow"
)

// NullEnqueuer is an enqueuer that doesn't do anything.
type NullEnqueuer struct{}

// Enqueue does nothing and returns nil.
func (n *NullEnqueuer) Enqueue(_ context.Context, _ []string, _ []byte) error { return nil }

// SupportsMultiCommands returns true.
func (n *NullEnqueuer) SupportsMultiCommands() bool { return true }

type CollectedStep struct {
	WorkflowName string
	*workflow.StepEnqueueing
}

type CollectingStepEnqueur struct {
	next   workflow.StepEnqueuer
	steps  []CollectedStep
	stepMu sync.RWMutex
}

func NewCollectingStepEnqueur(next workflow.StepEnqueuer) *CollectingStepEnqueur {
	return &CollectingStepEnqueur{next: next}
}

func (c *CollectingStepEnqueur) Steps() []CollectedStep {
	c.stepMu.RLock()
	defer c.stepMu.RUnlock()
	return c.steps
}

func (c *CollectingStepEnqueur) EnqueueStep(ctx context.Context, n workflow.Namer, es *workflow.StepEnqueueing) error {
	c.stepMu.Lock()
	c.steps = append(c.steps, CollectedStep{
		WorkflowName:   n.Name(),
		StepEnqueueing: es,
	})
	c.stepMu.Unlock()
	return c.next.EnqueueStep(ctx, n, es)
}
