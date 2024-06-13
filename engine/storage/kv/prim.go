package kv

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	kvStringSep = "," // using a comma is probably dangerous

	// step bucket
	keySfxStepMeta     = ".meta"     // marshalled step metadata
	keySfxStepIDs      = ".ids"      // marshalled step enrollment identifiers
	keySfxStepCtx      = ".ctx"      // step context
	keySfxStepCmds     = ".cmds"     // marshalled command UUIDs
	keySfxStepNotUntil = ".notuntil" // step NotUntil time
	keySfxStepTimeout  = ".timeout"  // step Timeout time

	// id-command bucket
	keySfxCmdStepID   = ".step"     // associated step ID
	keySfxCmdReqType  = ".reqtype"  // MDM command request type
	keySfxCmdRaw      = ".raw"      // raw MDM command plist
	keySfxCmdLastPush = ".lastpush" // MDM command last push time
	keySfxCmdResult   = ".result"   // raw MDM command result report plist
	keySfxCmdComplete = ".done"     // command is complete indicator
	keySfxCmdID       = ".id"       // enrollment ID for this command
)

var keySfxCmdKeys = []string{
	keySfxCmdStepID,  // should always exist
	keySfxCmdReqType, // should always exist
	keySfxCmdRaw,
	keySfxCmdLastPush, // should always exist
	keySfxCmdResult,
	keySfxCmdComplete,
	keySfxCmdID,
}

var keySfxStepKeys = []string{
	keySfxStepMeta, // should always exist
	keySfxStepIDs,  // should always exist
	keySfxStepCtx,
	keySfxStepCmds, // should always exist
	keySfxStepNotUntil,
	keySfxStepTimeout,
}

func marshalStrings(s []string) []byte {
	return []byte(strings.Join(s, kvStringSep))
}

func unmarshalStrings(b []byte) []string {
	return strings.Split(string(b), kvStringSep)
}

// kvSetStep writes step to b.
func kvSetStep(ctx context.Context, b kv.Bucket, stepID string, step *storage.StepEnqueuingWithConfig) error {
	// check of the step is valid
	err := step.Validate()
	if err != nil {
		return fmt.Errorf("validating step: %w", err)
	}

	var notUntilBytes []byte
	if !step.NotUntil.IsZero() {
		notUntilBytes, err = step.NotUntil.MarshalText()
		if err != nil {
			return fmt.Errorf("marshal not until time: %w", err)
		}
	}

	var timeoutBytes []byte
	if !step.Timeout.IsZero() {
		timeoutBytes, err = step.Timeout.MarshalText()
		if err != nil {
			return fmt.Errorf("marshal timeout time: %w", err)
		}
	}

	// gather our command UUIDs
	var cmdUUIDs []string
	for _, cmd := range step.Commands {
		cmdUUIDs = append(cmdUUIDs, cmd.CommandUUID)
	}

	// begin writing our step data
	sr := map[string][]byte{
		stepID + keySfxStepMeta: marshalStrings([]string{step.InstanceID, step.WorkflowName, step.Name}),
		stepID + keySfxStepIDs:  marshalStrings(step.IDs),
		stepID + keySfxStepCmds: marshalStrings(cmdUUIDs),
	}
	if len(step.Context) > 0 {
		sr[stepID+keySfxStepCtx] = step.Context
	}
	if len(notUntilBytes) > 0 {
		sr[stepID+keySfxStepNotUntil] = notUntilBytes
	}
	if len(timeoutBytes) > 0 {
		sr[stepID+keySfxStepTimeout] = timeoutBytes
	}
	if err = kv.SetMap(ctx, b, sr); err != nil {
		return fmt.Errorf("writing step records: %w", err)
	}

	return nil
}

// kvGetStepCmds retrieve step command UUIDs.
func kvGetStepCmds(ctx context.Context, b kv.Bucket, stepID string) ([]string, error) {
	cmdUUIDBytes, err := b.Get(ctx, stepID+keySfxStepCmds)
	return unmarshalStrings(cmdUUIDBytes), err
}

// kvGetStepIDs returns the enrollment IDs a step is enqueued for.
func kvGetStepIDs(ctx context.Context, b kv.Bucket, stepID string) ([]string, error) {
	stepEnrIDs, err := b.Get(ctx, stepID+keySfxStepIDs)
	return unmarshalStrings(stepEnrIDs), err
}

// kvGetStepResult creates and populates a step result from a stored step.
func kvGetStepResult(ctx context.Context, b kv.Bucket, stepID string) (*storage.StepResult, error) {
	step := new(storage.StepResult)

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

	// fetch the step context
	if ok, err := b.Has(ctx, stepID+keySfxStepCtx); err != nil {
		return step, fmt.Errorf("checking context: %w", err)
	} else if ok {
		if step.Context, err = b.Get(ctx, stepID+keySfxStepCtx); err != nil {
			return step, fmt.Errorf("reading context: %w", err)
		}
	}

	return step, nil
}

// kvGetReqType checks for and returns
func kvGetIDCmdReqType(ctx context.Context, b kv.Bucket, id, cmdUUID string) (string, bool, error) {
	// then, return get the request type
	val, err := b.Get(ctx, id+cmdUUID+keySfxCmdReqType)
	if err != nil {
		return "", false, fmt.Errorf("getting request type for %s: %w", cmdUUID, err)
	}
	return string(val), true, nil
}

// kvSetIDCmd writes sc to b.
func kvSetIDCmd(ctx context.Context, b kv.Bucket, stepID, id string, sc *storage.StepCommandRaw, lastPush time.Time, saveRaw bool) error {
	cr := map[string][]byte{
		id + sc.CommandUUID + keySfxCmdStepID:  []byte(stepID),
		id + sc.CommandUUID + keySfxCmdReqType: []byte(sc.RequestType),
		id + sc.CommandUUID + keySfxCmdID:      []byte(id),
	}
	if saveRaw && len(sc.Command) > 0 {
		cr[id+sc.CommandUUID+keySfxCmdRaw] = sc.Command
	}

	var err error
	cr[id+sc.CommandUUID+keySfxCmdLastPush], err = lastPush.MarshalText()
	if err != nil {
		return fmt.Errorf("marshal last push time: %w", err)
	}

	if err = kv.SetMap(ctx, b, cr); err != nil {
		return fmt.Errorf("writing command records: %w", err)
	}

	return nil
}

// kvIDCmdExists checks to see if a command UUID exists for an enrollment ID.
func kvIDCmdExists(ctx context.Context, b kv.Bucket, id, cmdUUID string) (bool, error) {
	return b.Has(ctx, id+cmdUUID+keySfxCmdStepID)
}

// kvIDCmdIsComplete checks to see if a command UUID is complete for an enrollment ID.
func kvIDCmdIsComplete(ctx context.Context, b kv.Bucket, id, cmdUUID string) (bool, error) {
	return b.Has(ctx, id+cmdUUID+keySfxCmdComplete)
}

// kvGetIDCmdResult retrieves the previuosly saved command result (likely from kvSetIDCmdUpdate())
func kvGetIDCmdResult(ctx context.Context, b kv.Bucket, id, cmdUUID string) ([]byte, error) {
	return b.Get(ctx, id+cmdUUID+keySfxCmdResult)
}

// kvSetIDCmdUpdate updates the command in sc with the result data.
func kvSetIDCmdUpdate(ctx context.Context, b kv.Bucket, id string, sc *storage.StepCommandResult) error {
	cr := map[string][]byte{
		id + sc.CommandUUID + keySfxCmdResult: sc.ResultReport,
	}
	if sc.Completed {
		cr[id+sc.CommandUUID+keySfxCmdComplete] = []byte{'1'}
	}
	return kv.SetMap(ctx, b, cr)
}

// kvGetIDCmdStepID tries to read the step ID of a command.
func kvGetIDCmdStepID(ctx context.Context, b kv.Bucket, id, cmdUUID string) (string, error) {
	stepIDBytes, err := b.Get(ctx, id+cmdUUID+keySfxCmdStepID)
	return string(stepIDBytes), err
}

func kvDeleteKeysIfExists(ctx context.Context, b kv.Bucket, keys []string) error {
	for _, k := range keys {
		ok, err := b.Has(ctx, k)
		if err != nil {
			return fmt.Errorf("checking key %s: %w", k, err)
		} else if !ok {
			continue
		}
		if err = b.Delete(ctx, k); err != nil {
			return fmt.Errorf("delete key %s: %w", k, err)
		}
	}
	return nil
}

// kvDeleteIDCmd deletes all keys for a command queued for id.
func kvDeleteIDCmd(ctx context.Context, b kv.Bucket, id, cmdUUID string) error {
	var keys []string
	for _, k := range keySfxCmdKeys {
		keys = append(keys, id+cmdUUID+k)
	}
	return kvDeleteKeysIfExists(ctx, b, keys)
}

// kvDeleteStep deletes all keys for a step.
func kvDeleteStep(ctx context.Context, b kv.Bucket, stepID string) error {
	var keys []string
	for _, k := range keySfxStepKeys {
		keys = append(keys, stepID+k)
	}
	return kvDeleteKeysIfExists(ctx, b, keys)
}

// kvDeleteStepNotUntil deletes a step's NotUntil key
func kvDeleteStepNotUntil(ctx context.Context, b kv.Bucket, stepID string) error {
	return kvDeleteKeysIfExists(ctx, b, []string{stepID + keySfxStepNotUntil})
}

// kvFindWorkflowStepsWithIDs finds specific workflow steps (step IDs) for specific enrollment IDs.
func kvFindWorkflowStepsWithIDs(ctx context.Context, b kv.KeysPrefixTraversingBucket, name string, ids []string) ([]string, error) {
	var stepIDs []string

	// this.. is not very efficient. perhaps it would be better to
	// make a specific bucket/index for this.
start:
	for k := range b.Keys(ctx, nil) {
		if !strings.HasSuffix(k, keySfxStepMeta) {
			continue
		}
		metaBytes, err := b.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("getting step meta for %s: %w", k, err)
		}
		stepID := k[:len(k)-len(keySfxStepMeta)]
		if unmarshalStrings(metaBytes)[1] != name {
			continue
		}
		stepEnrIDs, err := kvGetStepIDs(ctx, b, stepID)
		if err != nil {
			return nil, fmt.Errorf("getting step ids for %s: %w", k, err)
		}
		for _, stepEnrID := range stepEnrIDs {
			for _, id := range ids {
				if stepEnrID != id {
					continue
				}
				// found that this step (stepID) contains our enrollment ID
				stepIDs = append(stepIDs, stepID)
				continue start
			}

		}
	}

	return stepIDs, nil
}

func kvDeleteStepIfAllIDsComplete(ctx context.Context, b kv.Bucket, cb kv.Bucket, stepID string, cmdUUIDs []string) error {
	stepEnrIDs, err := kvGetStepIDs(ctx, b, stepID)
	if err != nil {
		return fmt.Errorf("getting step IDs for step %s: %w", stepID, err)
	}
	var cmdsToDelete []struct {
		id   string
		uuid string
	}
	for _, id := range stepEnrIDs {
		for _, cmdUUID := range cmdUUIDs {
			ok, err := kvIDCmdExists(ctx, cb, id, cmdUUID)
			if err != nil {
				return fmt.Errorf("checking command exists for %s: %w", cmdUUID, err)
			} else if !ok {
				// command doesn't exist. could be deleted already.
				continue
			}
			if ok, err = kvIDCmdIsComplete(ctx, cb, id, cmdUUID); err != nil {
				return fmt.Errorf("checking command complete for %s: %w", cmdUUID, err)
			} else if !ok {
				// step is not complete, bail now
				return nil
			}
			// if we got here then this command still exists but is completed
			cmdsToDelete = append(cmdsToDelete, struct {
				id   string
				uuid string
			}{
				id:   id,
				uuid: cmdUUID,
			})
		}
	}

	// delete any commands
	for _, cmd := range cmdsToDelete {
		if err = kvDeleteIDCmd(ctx, cb, cmd.id, cmd.uuid); err != nil {
			return fmt.Errorf("deleting command for %s: %w", cmd.uuid, err)
		}
	}

	// delete step
	if err = kvDeleteStep(ctx, b, stepID); err != nil {
		return fmt.Errorf("deleting step for %s: %w", stepID, err)
	}

	return nil
}
