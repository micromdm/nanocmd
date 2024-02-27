// Package kv implements a workflow engine storage backend using a key-value interface.
package kv

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/utils/kv"
	"github.com/micromdm/nanocmd/utils/uuid"
)

// KV is a workflow engine storage backend using a key-value interface.
type KV struct {
	mu          sync.RWMutex
	stepStore   kv.TraversingBucket
	idCmdStore  kv.TraversingBucket
	eventStore  kv.TraversingBucket
	ider        uuid.IDer
	statusStore kv.TraversingBucket
}

// New creates a new key-value workflow engine storage backend.
func New(stepStore kv.TraversingBucket, idCmdStore kv.TraversingBucket, eventStore kv.TraversingBucket, ider uuid.IDer, statusStore kv.TraversingBucket) *KV {
	return &KV{
		stepStore:   stepStore,
		idCmdStore:  idCmdStore,
		eventStore:  eventStore,
		ider:        ider,
		statusStore: statusStore,
	}
}

// RetrieveCommandRequestType implements the storage interface method.
func (s *KV) RetrieveCommandRequestType(ctx context.Context, id string, cmdUUID string) (string, bool, error) {
	if id == "" || cmdUUID == "" {
		return "", false, errors.New("empty id or command uuid")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	// first check if we have a valid command
	if ok, err := kvIDCmdExists(ctx, s.idCmdStore, id, cmdUUID); err != nil {
		return "", false, fmt.Errorf("checking command exists for %s: %w", cmdUUID, err)
	} else if !ok {
		return "", false, nil
	}
	// then return the request type
	return kvGetIDCmdReqType(ctx, s.idCmdStore, id, cmdUUID)
}

// StoreCommandResponseAndRetrieveCompletedStep implements the storage interface method.
func (s *KV) StoreCommandResponseAndRetrieveCompletedStep(ctx context.Context, id string, sc *storage.StepCommandResult) (*storage.StepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// check to make sure this command actually exists
	if ok, err := kvIDCmdExists(ctx, s.idCmdStore, id, sc.CommandUUID); err != nil {
		return nil, fmt.Errorf("checking command exists for %s: %w", sc.CommandUUID, err)
	} else if !ok {
		// command must exist for us to try to update a response to it
		return nil, fmt.Errorf("command not found: %s", sc.CommandUUID)
	}

	// update our command response data
	// TODO: we may not need to write the result if we've finished the command
	//       to place this after the next !sc.Completed check
	if err := kvSetIDCmdUpdate(ctx, s.idCmdStore, id, sc); err != nil {
		return nil, fmt.Errorf("setting command result: %w", err)
	}

	if !sc.Completed {
		// if 'this' command itself is not complete, then the step
		// can't be completed.
		// return incomplete after we've already recorded its result.
		return nil, nil
	}
	// 'this' command is complete, let's now check to see if the step is
	// complete for this id.

	// retrieve the step ID for the command UUID for this id
	stepID, err := kvGetIDCmdStepID(ctx, s.idCmdStore, id, sc.CommandUUID)
	if err != nil {
		return nil, fmt.Errorf("getting step ID for %s: %w", sc.CommandUUID, err)
	}

	// retrieve the command UUIDs of this step
	stepCmdUUIDs, err := kvGetStepCmds(ctx, s.stepStore, stepID)
	if err != nil {
		return nil, fmt.Errorf("reading step commands: %w", err)
	}

	var commands []storage.StepCommandResult
	for _, stepCmdUUID := range stepCmdUUIDs {
		// assume complete only because if 'this' command was not
		// complete itself we should't even get this far
		stepCmdComplete := true
		if stepCmdUUID != sc.CommandUUID {
			stepCmdComplete, err = kvIDCmdIsComplete(ctx, s.idCmdStore, id, stepCmdUUID)
			if err != nil {
				return nil, fmt.Errorf("checking complete status for %s: %w", stepCmdUUID, err)
			}
		}
		if !stepCmdComplete {
			// if any of our commands aren't yet finished then return as incomplete
			return nil, nil
		}

		// start assembling our storage command result
		stepCommandResult := storage.StepCommandResult{
			CommandUUID: stepCmdUUID,
			Completed:   stepCmdComplete,
		}

		if stepCmdUUID == sc.CommandUUID {
			// used 'this' command's response rather than looking it up
			stepCommandResult.RequestType = sc.RequestType
			stepCommandResult.ResultReport = sc.ResultReport
		} else {
			stepCommandResult.ResultReport, err = kvGetIDCmdResult(ctx, s.idCmdStore, id, stepCmdUUID)
			if err != nil {
				return nil, fmt.Errorf("getting result for %s: %w", stepCmdUUID, err)
			}
		}

		if stepCommandResult.RequestType == "" {
			stepCommandResult.RequestType, _, err = kvGetIDCmdReqType(ctx, s.idCmdStore, id, stepCmdUUID)
			if err != nil {
				return nil, fmt.Errorf("getting request type for %s: %w", stepCmdUUID, err)
			}
		}

		commands = append(commands, stepCommandResult)
	}

	step, err := kvGetStepResult(ctx, s.stepStore, stepID)
	if err != nil {
		return step, fmt.Errorf("retrieving step result: %w", err)
	}
	step.IDs = []string{id}
	step.Commands = commands

	// delete all id-command records
	for _, stepCmdUUID := range stepCmdUUIDs {
		if err = kvDeleteIDCmd(ctx, s.idCmdStore, id, stepCmdUUID); err != nil {
			return step, fmt.Errorf("deleting command %s: %w", stepCmdUUID, err)
		}
	}

	err = kvDeleteStepIfAllIDsComplete(ctx, s.stepStore, s.idCmdStore, stepID, stepCmdUUIDs)
	if err != nil {
		return step, fmt.Errorf("step deletion: %w", err)
	}

	return step, nil
}

// StoreStep implements the storage interface method.
func (s *KV) StoreStep(ctx context.Context, step *storage.StepEnqueuingWithConfig, pushTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// fabricate a unique ID to track this unique step
	stepID := s.ider.ID()

	if step != nil {
		idCmdUUIDs := make(map[string]struct{})
		for _, sc := range step.Commands {
			for _, id := range step.IDs {
				if _, ok := idCmdUUIDs[id+sc.CommandUUID]; ok {
					return fmt.Errorf("duplicate command (id=%s, uuid=%s)", id, sc.CommandUUID)
				}
				idCmdUUIDs[id+sc.CommandUUID] = struct{}{}
				if ok, err := kvIDCmdExists(ctx, s.idCmdStore, id, sc.CommandUUID); err != nil {
					return fmt.Errorf("checking duplicate commands: %w", err)
				} else if ok {
					return fmt.Errorf("duplicate command (id=%s, uuid=%s)", id, sc.CommandUUID)
				}
			}
		}
	}

	err := kvSetStep(ctx, s.stepStore, stepID, step)
	if err != nil {
		return fmt.Errorf("setting step record: %w", err)
	}

	var lastPush time.Time
	if step.NotUntil.IsZero() {
		// assume that these commands have been pushed if they haven't
		// explicitly been delayed
		lastPush = time.Now()
	}

	// explode each command to be tracked for each id
	for _, sc := range step.Commands {
		if !step.NotUntil.IsZero() {
			// write the command with an enrollment ID of the stepID
			if err = kvSetIDCmd(ctx, s.idCmdStore, stepID, stepID, &sc, time.Time{}, true); err != nil {
				return fmt.Errorf("writing step not until command records: %w", err)
			}
		}
		for _, id := range step.IDs {
			if err = kvSetIDCmd(ctx, s.idCmdStore, stepID, id, &sc, lastPush, false); err != nil {
				return fmt.Errorf("writing command records: %w", err)
			}
		}
	}

	return nil
}

// StoreStep implements the storage interface method.
func (s *KV) RetrieveOutstandingWorkflowStatus(ctx context.Context, workflowName string, ids []string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stepIDs, err := kvFindWorkflowStepsWithIDs(ctx, s.stepStore, workflowName, ids)
	if err != nil {
		return nil, fmt.Errorf("finding workflow steps: %w", err)
	}

	// now that we have workflow steps of interest (that is, that were enqueued to us)
	// lets make sure we have uncompleted steps for our ids.

	idAcc := make(map[string]struct{})

	for _, stepID := range stepIDs {
		cmdUUIDs, err := kvGetStepCmds(ctx, s.stepStore, stepID)
		if err != nil {
			return nil, fmt.Errorf("getting step commands for %s: %w", stepID, err)
		}
	foundInStep:
		for _, id := range ids {
			if _, ok := idAcc[id]; ok {
				continue
			}
			for _, cmdUUID := range cmdUUIDs {
				if ok, err := kvIDCmdExists(ctx, s.idCmdStore, id, cmdUUID); err != nil {
					return nil, fmt.Errorf("checking command exists for %s: %w", cmdUUID, err)
				} else if !ok {
					// command does not exist for this id, perhaps already completed (and deleted?)
					continue
				}
				if ok, err := kvIDCmdIsComplete(ctx, s.idCmdStore, id, cmdUUID); err != nil {
					return nil, fmt.Errorf("getting command complete status for %s: %w", cmdUUID, err)
				} else if !ok {
					idAcc[id] = struct{}{}
					continue foundInStep
				}
			}
		}
	}

	outstandingIDs := make([]string, 0, len(idAcc))
	for id := range idAcc {
		outstandingIDs = append(outstandingIDs, id)
	}

	return outstandingIDs, nil
}

// CancelSteps implements the storage interface method.
func (s *KV) CancelSteps(ctx context.Context, id, workflowName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stepIDs, err := kvFindWorkflowStepsWithIDs(ctx, s.stepStore, workflowName, []string{id})
	if err != nil {
		return fmt.Errorf("finding workflow steps: %w", err)
	}
	for _, stepID := range stepIDs {
		cmdUUIDs, err := kvGetStepCmds(ctx, s.stepStore, stepID)
		if err != nil {
			return fmt.Errorf("getting step commands for %s: %w", stepID, err)
		}
		for _, cmdUUID := range cmdUUIDs {
			if err = kvDeleteIDCmd(ctx, s.idCmdStore, id, cmdUUID); err != nil {
				return fmt.Errorf("deleting commands for %s: %w", cmdUUID, err)
			}
		}
		if err = kvDeleteStep(ctx, s.stepStore, stepID); err != nil {
			return fmt.Errorf("deleting step for %s: %w", stepID, err)
		}
	}
	return nil
}

func workflowStatusKey(id, workflowName string) string {
	return id + "." + workflowName
}

// RetrieveWorkflowStarted returns the last time a workflow was started for id.
func (s *KV) RetrieveWorkflowStarted(ctx context.Context, id, workflowName string) (time.Time, error) {
	var started time.Time
	if found, err := s.statusStore.Has(ctx, workflowStatusKey(id, workflowName)); err != nil {
		return started, fmt.Errorf("status not found for id=%s workflow=%s: %w", id, workflowName, err)
	} else if !found {
		return started, nil
	}
	b, err := s.statusStore.Get(ctx, workflowStatusKey(id, workflowName))
	if err != nil {
		return started, fmt.Errorf("getting workflow status: %w", err)
	}
	if err = started.UnmarshalText(b); err != nil {
		err = fmt.Errorf("unmarshaling workflow status: %w", err)
	}
	return started, err
}

// RecordWorkflowStarted stores the started time for workflowName for ids.
func (s *KV) RecordWorkflowStarted(ctx context.Context, ids []string, workflowName string, started time.Time) error {
	b, err := started.MarshalText()
	if err != nil {
		return fmt.Errorf("marshaling workflow status: %w", err)
	}
	for _, id := range ids {
		if err = s.statusStore.Set(ctx, workflowStatusKey(id, workflowName), b); err != nil {
			return fmt.Errorf("setting workflow status for id=%s workflow=%s: %w", id, workflowName, err)
		}
	}
	return nil
}

// ClearWorkflowStatus removes all workflow start times for id.
func (s *KV) ClearWorkflowStatus(ctx context.Context, id string) error {
	var toDelete []string
	for k := range s.statusStore.Keys(nil) {
		// very inefficient! this could be a large table
		if strings.HasPrefix(k, id+".") {
			toDelete = append(toDelete, k)
		}
	}
	return kv.DeleteSlice(ctx, s.statusStore, toDelete)
}
