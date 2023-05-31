package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
)

// RetrieveStepsToEnqueue implements the storage interface method.
func (s *KV) RetrieveStepsToEnqueue(ctx context.Context, pushTime time.Time) ([]*storage.StepEnqueueing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stepIDs, err := kvFindNotUntilStepsWithIDs(ctx, s.stepStore)
	if err != nil {
		return nil, fmt.Errorf("finding not util steps: %w", err)
	}

	var ret []*storage.StepEnqueueing

	for _, stepID := range stepIDs {
		se, err := kvGetStepEnqueueing(ctx, s.stepStore, s.idCmdStore, stepID)
		if err != nil {
			return nil, fmt.Errorf("getting step enqueueing for %s: %w", stepID, err)
		}
		if err = kvDeleteStepNotUntil(ctx, s.stepStore, stepID); err != nil {
			return nil, fmt.Errorf("deleting step not until for %s: %w", stepID, err)
		}
		// reset our push times
		for _, id := range se.IDs {
			for _, cmd := range se.Commands {
				if err = kvSetIDCmdLastPush(ctx, s.idCmdStore, id, cmd.CommandUUID, pushTime); err != nil {
					return nil, fmt.Errorf("setting last push for %s: %w", cmd.CommandUUID, err)
				}
			}
		}
		ret = append(ret, se)
	}

	return ret, nil
}

// RetrieveTimedOutSteps implements the storage interface method.
func (s *KV) RetrieveTimedOutSteps(ctx context.Context) ([]*storage.StepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stepIDs, err := kvFindTimedOutStepIDs(ctx, s.stepStore)
	if err != nil {
		return nil, fmt.Errorf("finding timeout steps: %w", err)
	}

	var steps []*storage.StepResult

	for _, stepID := range stepIDs {
		step, err := kvGetStepResult(ctx, s.stepStore, stepID)
		if err != nil {
			return nil, fmt.Errorf("retrieving step result: %w", err)
		}
		stepEnrIDs, err := kvGetStepIDs(ctx, s.stepStore, stepID)
		if err != nil {
			return nil, fmt.Errorf("retrieving step enrollment IDs: %w", err)
		}
		stepCmdUUIDs, err := kvGetStepCmds(ctx, s.stepStore, stepID)
		if err != nil {
			return nil, fmt.Errorf("retrieving step enrollment IDs: %w", err)
		}

		for _, id := range stepEnrIDs {
			// make a per-id copy of our step for workflow processing
			step2 := *step
			step2.IDs = []string{id}

			for _, stepCmdUUID := range stepCmdUUIDs {
				result, err := kvGetIDCmdStepResult(ctx, s.idCmdStore, id, stepCmdUUID, false)
				if err != nil {
					return nil, fmt.Errorf("retrieving command result for %s: %w", stepCmdUUID, err)
				}
				if result != nil {
					step2.Commands = append(step2.Commands, *result)

					// clear out the step commands
					if err = kvDeleteIDCmd(ctx, s.idCmdStore, id, stepCmdUUID); err != nil {
						return nil, fmt.Errorf("retrieving command result for %s: %w", stepCmdUUID, err)
					}
				}
			}

			if len(step2.Commands) > 0 {
				steps = append(steps, &step2)
			}
		}

		// clear out the step
		if err = kvDeleteStep(ctx, s.stepStore, stepID); err != nil {
			return nil, fmt.Errorf("deleting step for %s: %w", stepID, err)
		}
	}

	return steps, nil
}

// RetrieveAndMarkRePushed implements the storage interface method.
func (s *KV) RetrieveAndMarkRePushed(ctx context.Context, ifBefore time.Time, setTo time.Time) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return kvFindCommandsToRePush(ctx, s.idCmdStore, ifBefore, setTo)
}
