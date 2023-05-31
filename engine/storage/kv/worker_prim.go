package kv

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/utils/kv"
)

func kvFindNotUntilStepsWithIDs(ctx context.Context, b kv.TraversingBucket) ([]string, error) {
	var stepIDs []string

	now := time.Now()

	// this.. is not very efficient. perhaps it would be better to
	// make a specific bucket/index for this.
	for k := range b.Keys(nil) {
		if !strings.HasSuffix(k, keySfxStepNotUntil) {
			continue
		}
		notUntilBytes, err := b.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("getting step meta for %s: %w", k, err)
		}
		var notUntil time.Time
		if err = notUntil.UnmarshalText(notUntilBytes); err != nil {
			return nil, fmt.Errorf("unmarshal not until time: %w", err)
		}
		if notUntil.After(now) || notUntil.IsZero() {
			continue
		}
		stepID := k[:len(k)-len(keySfxStepNotUntil)]
		// found that this step (stepID) contains our enrollment ID
		stepIDs = append(stepIDs, stepID)
	}

	return stepIDs, nil
}

func kvGetStepEnqueueing(ctx context.Context, b kv.Bucket, cb kv.Bucket, stepID string) (*storage.StepEnqueueing, error) {
	step := new(storage.StepEnqueueing)

	// populate all the metadata of the step
	metaBytes, err := b.Get(ctx, stepID+keySfxStepMeta)
	if err != nil {
		return step, fmt.Errorf("getting step meta: %w", err)
	}
	s := unmarshalStrings(metaBytes)
	if len(s) != 3 {
		return step, errors.New("invalid step metadata length")
	}
	step.InstanceID = s[0]
	step.WorkflowName = s[1]
	step.Name = s[2]

	// get enrollment IDs
	stepEnrIDsBytes, err := b.Get(ctx, stepID+keySfxStepIDs)
	if err != nil {
		return step, fmt.Errorf("getting step enrollment ID for %s: %w", stepID, err)
	}
	step.IDs = unmarshalStrings(stepEnrIDsBytes)

	// retrieve the list of commands
	stepCmdUUIDs, err := kvGetStepCmds(ctx, b, stepID)
	if err != nil {
		return step, fmt.Errorf("getting step commands for %s: %w", stepCmdUUIDs, err)
	}

	for _, stepCmdUUID := range stepCmdUUIDs {
		sc, err := kvGetIDCmdRaw(ctx, cb, stepID, stepCmdUUID)
		if err != nil {
			return nil, fmt.Errorf("getting command queue data for %s: %w", stepCmdUUID, err)
		}
		step.Commands = append(step.Commands, *sc)
		if err = kvDeleteIDCmd(ctx, cb, stepID, stepCmdUUID); err != nil {
			return nil, fmt.Errorf("deleting command for %s: %w", stepID, err)
		}
	}

	// no longer consider this for for future NotUntils
	if err = b.Delete(ctx, stepID+keySfxStepNotUntil); err != nil {
		return nil, fmt.Errorf("deleting not now for %s: %w", stepID, err)
	}

	return step, nil
}

func kvGetIDCmdRaw(ctx context.Context, b kv.Bucket, id, cmdUUID string) (*storage.StepCommandRaw, error) {
	cr, err := kv.GetMap(ctx, b, []string{
		id + cmdUUID + keySfxCmdReqType,
		id + cmdUUID + keySfxCmdRaw,
	})
	if err != nil {
		return nil, err
	}
	return &storage.StepCommandRaw{
		CommandUUID: cmdUUID,
		RequestType: string(cr[id+cmdUUID+keySfxCmdReqType]),
		Command:     cr[id+cmdUUID+keySfxCmdRaw],
	}, nil
}

func kvFindTimedOutStepIDs(ctx context.Context, b kv.TraversingBucket) ([]string, error) {
	var stepIDs []string

	now := time.Now()

	// this.. is not very efficient. perhaps it would be better to
	// make a specific bucket/index for this.
	for k := range b.Keys(nil) {
		if !strings.HasSuffix(k, keySfxStepTimeout) {
			continue
		}
		timeoutBytes, err := b.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("getting step meta for %s: %w", k, err)
		}
		var timeout time.Time
		if err = timeout.UnmarshalText(timeoutBytes); err != nil {
			return nil, fmt.Errorf("unmarshal not until time: %w", err)
		}
		if timeout.After(now) || timeout.IsZero() {
			continue
		}
		stepID := k[:len(k)-len(keySfxStepTimeout)]
		// found that this step (stepID) contains our enrollment ID
		stepIDs = append(stepIDs, stepID)
	}

	return stepIDs, nil
}

func kvGetIDCmdStepResult(ctx context.Context, b kv.Bucket, id, cmdUUID string, noCheckExists bool) (*storage.StepCommandResult, error) {
	if !noCheckExists {
		ok, err := kvIDCmdExists(ctx, b, id, cmdUUID)
		if err != nil {
			return nil, fmt.Errorf("checking command exists for %s: %w", cmdUUID, err)
		} else if !ok {
			return nil, nil
		}
	}
	result := &storage.StepCommandResult{
		CommandUUID:  cmdUUID,
		ResultReport: []byte{},
	}
	var err error
	result.Completed, err = kvIDCmdIsComplete(ctx, b, id, cmdUUID)
	if err != nil {
		return nil, fmt.Errorf("checking command completed for %s: %w", cmdUUID, err)
	}
	result.RequestType, _, err = kvGetIDCmdReqType(ctx, b, id, cmdUUID)
	if err != nil {
		return nil, fmt.Errorf("getting command req type for %s: %w", cmdUUID, err)
	}
	if ok, err := b.Has(ctx, id+cmdUUID+keySfxCmdResult); err != nil {
		return nil, fmt.Errorf("checking result exists for %s: %w", cmdUUID, err)
	} else if ok {
		result.ResultReport, err = kvGetIDCmdResult(ctx, b, id, cmdUUID)
		if err != nil {
			return nil, fmt.Errorf("getting command result for %s: %w", cmdUUID, err)
		}
	}
	return result, err
}

func kvFindCommandsToRePush(ctx context.Context, b kv.TraversingBucket, ifBefore time.Time, setTo time.Time) ([]string, error) {
	var ids []string

	resetLastPushes := make(map[string][]byte)

	// this.. is not very efficient. perhaps it would be better to
	// make a specific bucket/index for this.
	for k := range b.Keys(nil) {
		if !strings.HasSuffix(k, keySfxCmdLastPush) {
			continue
		}
		lastPushBytes, err := b.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("getting step meta for %s: %w", k, err)
		}
		var lastPush time.Time
		if err = lastPush.UnmarshalText(lastPushBytes); err != nil {
			return nil, fmt.Errorf("unmarshal not until time: %w", err)
		}

		if lastPush.IsZero() || lastPush.After(ifBefore) {
			continue
		}

		idCmd := k[:len(k)-len(keySfxCmdLastPush)]
		// lookup the enrollment ID of this command
		idBytes, err := b.Get(ctx, idCmd+keySfxCmdID)
		if err != nil {
			return nil, fmt.Errorf("getting command for %s: %w", idCmd, err)
		}
		id := string(idBytes)

		// reset the last push time
		if lastPushBytes, err = setTo.MarshalText(); err != nil {
			return nil, fmt.Errorf("marshal now: %w", err)
		}
		resetLastPushes[k] = lastPushBytes

		// found that this step (stepID) contains our enrollment ID
		ids = append(ids, id)
	}

	if err := kv.SetMap(ctx, b, resetLastPushes); err != nil {
		return ids, fmt.Errorf("resetting pushes: %w", err)
	}

	return ids, nil
}

func kvSetIDCmdLastPush(ctx context.Context, b kv.Bucket, id, cmdUUID string, lastPush time.Time) error {
	lastPushBytes, err := lastPush.MarshalText()
	if err != nil {
		return fmt.Errorf("marshal last push: %w", err)
	}
	return b.Set(ctx, id+cmdUUID+keySfxCmdLastPush, lastPushBytes)
}
