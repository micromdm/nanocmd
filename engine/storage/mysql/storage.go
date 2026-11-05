package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/engine/storage/mysql/sqlc"
	"github.com/micromdm/nanocmd/logkeys"

	"github.com/micromdm/nanolib/log/ctxlog"
)

// RetrieveCommandRequestType retrieves a command request type given id and uuid.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveCommandRequestType(ctx context.Context, id string, uuid string) (string, bool, error) {
	if id == "" || uuid == "" {
		return "", false, errors.New("empty id or command uuid")
	}
	reqType, err := s.q.GetRequestType(ctx, sqlc.GetRequestTypeParams{EnrollmentID: id, CommandUuid: uuid})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return reqType, reqType != "", err
}

// StoreCommandResponseAndRetrieveCompletedStep stores a command response and returns the completed step for the id.
// See the storage interface type for further docs.
func (s *MySQLStorage) StoreCommandResponseAndRetrieveCompletedStep(ctx context.Context, id string, sc *storage.StepCommandResult) (*storage.StepResult, error) {
	if sc == nil {
		return nil, errors.New("nil storage command")
	}
	if !sc.Completed {
		// if this command is not completed (i.e. NotNow) then the step cannot be completed, either.
		err := s.q.UpdateIDCommandTimestamp(ctx, sqlc.UpdateIDCommandTimestampParams{
			EnrollmentID: id,
			CommandUuid:  sc.CommandUUID,
		})
		if err != nil {
			err = fmt.Errorf("updating id command timestamp: %w", err)
		}
		return nil, err
	}

	var (
		ret    *storage.StepResult
		stepID int64
	)

	// The completion decision is made inside the transaction, from the rows it
	// locks -- not from a count read beforehand, which the worker's timeout
	// sweep could invalidate before the transaction acted on it. See
	// GetIDCommandsByStepIDAndLock.
	err := tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		var err error
		stepID, err = qtx.GetStepIDByCommandUUID(ctx, sqlc.GetStepIDByCommandUUIDParams{
			EnrollmentID: id,
			CommandUuid:  sc.CommandUUID,
		})
		if err != nil {
			return fmt.Errorf("get step id (id=%s, uuid=%s): %w", id, sc.CommandUUID, err)
		}
		if stepID < 1 {
			return fmt.Errorf("no step ID found (id=%s, uuid=%s)", id, sc.CommandUUID)
		}

		// locks this enrollment's rows for the step, and only those
		cmdR, err := qtx.GetIDCommandsByStepIDAndLock(ctx, sqlc.GetIDCommandsByStepIDAndLockParams{
			EnrollmentID: id,
			StepID:       stepID,
		})
		if err != nil {
			return fmt.Errorf("get id commands by step by id (%d): %w", stepID, err)
		}

		var outstanding int
		for _, dbSC := range cmdR {
			if dbSC.CommandUuid != sc.CommandUUID && !dbSC.Completed {
				outstanding++
			}
		}

		if outstanding > 0 {
			// other commands of ours for this step are still outstanding.
			// record this result for whichever of them comes in last.
			err = qtx.UpdateIDCommand(ctx, sqlc.UpdateIDCommandParams{
				Completed: sc.Completed,
				Result:    sc.ResultReport,
				// where
				EnrollmentID: id,
				CommandUuid:  sc.CommandUUID,
			})
			if err != nil {
				return fmt.Errorf("updating id command: %w", err)
			}
			return nil
		}

		// reaching here implies this is the last command to be completed
		// for the workflow step, for this instance ID for this enrollment ID.

		sd, err := qtx.GetStepByID(ctx, stepID)
		if err != nil {
			return fmt.Errorf("get step by id (%d): %w", stepID, err)
		}

		ret = &storage.StepResult{
			IDs: []string{id},
			StepContext: storage.StepContext{
				WorkflowName: sd.WorkflowName,
				InstanceID:   sd.InstanceID,
				Name:         sd.StepName.String,
				Context:      sd.Context,
			},
			// this command result
			Commands: []storage.StepCommandResult{*sc},
		}

		for _, dbSC := range cmdR {
			if dbSC.CommandUuid == sc.CommandUUID {
				continue // this command: already included, above
			}
			ret.Commands = append(ret.Commands, storage.StepCommandResult{
				RequestType:  dbSC.RequestType,
				CommandUUID:  dbSC.CommandUuid,
				ResultReport: dbSC.Result,
				Completed:    true,
			})
		}

		err = qtx.RemoveIDCommandsByStepID(ctx, sqlc.RemoveIDCommandsByStepIDParams{
			EnrollmentID: id,
			StepID:       stepID,
		})
		if err != nil {
			return fmt.Errorf("remove id commands by step by id (%d): %w", stepID, err)
		}

		return nil
	})
	if err != nil {
		return ret, fmt.Errorf("tx step completed: %w", err)
	}
	if ret == nil {
		return nil, nil
	}

	// The step result is complete and committed; what follows only reclaims
	// the step row. Its error is logged rather than returned: the caller
	// discards the step result if we return one (see the engine's step
	// completion handling), and losing a workflow step is far worse than
	// leaving a row behind for the cost of the collection having failed.
	if err = s.collectFinishedSteps(ctx, []int64{stepID}); err != nil {
		ctxlog.Logger(ctx, s.logger).Info(
			logkeys.Message, "collecting finished step",
			logkeys.EnrollmentID, id,
			logkeys.CommandUUID, sc.CommandUUID,
			logkeys.Error, err,
		)
	}

	return ret, nil
}

// collectFinishedSteps deletes those of stepIDs that nothing refers to any
// more, together with any raw commands still held for them.
//
// Must be called after the caller's own transaction has committed, never from
// within it. Whether a step is finished is a question about rows other
// enrollments own: answering it inside a transaction means locking either
// those rows or the step they all share, which is what serialized the fan-out.
// Asked afterwards it needs no lock at all.
//
// The trade is that a step can be left behind -- two callers committing at the
// same instant can each still see the other's rows, a collector that finds the
// step already claimed leaves it rather than waiting, and a process that stops
// between the two phases never gets here. An uncollected step is inert, but it
// is not reclaimed.
//
// Work is done in chunks, each its own transaction. A chunk that fails does
// not stop the others: no invariant spans them, so a step left behind by one
// is only the same inert row the paragraph above already allows for.
func (s *MySQLStorage) collectFinishedSteps(ctx context.Context, stepIDs []int64) error {
	var chunks, failed int
	var lastErr error

	for len(stepIDs) > 0 {
		chunk := stepIDs
		if len(chunk) > collectChunkSize {
			chunk = chunk[:collectChunkSize]
		}
		stepIDs = stepIDs[len(chunk):]

		chunks++
		if err := s.collectChunk(ctx, chunk); err != nil {
			// carry on: the remaining chunks are unaffected. only the last
			// error is kept -- they are overwhelmingly the same error, and
			// the count is what says how much was not collected.
			failed++
			lastErr = err
		}
	}

	if lastErr != nil {
		return fmt.Errorf("collecting steps (%d of %d chunks failed), last error: %w", failed, chunks, lastErr)
	}
	return nil
}

// collectChunkSize bounds how many step IDs are handled at once. Each chunk
// becomes one IN list -- MySQL allows 65535 placeholders in a prepared
// statement -- and one transaction's worth of binlog, so keeping it small
// bounds both, and keeps the claim below from holding its locks for long.
const collectChunkSize = 1000

// collectChunk collects a single chunk of step IDs. See collectFinishedSteps.
func (s *MySQLStorage) collectChunk(ctx context.Context, stepIDs []int64) error {
	// deliberately not in a transaction: see collectFinishedSteps.
	unreferenced, err := s.q.SelectUnreferencedStepIDs(ctx, stepIDs)
	if err != nil {
		return fmt.Errorf("selecting unreferenced steps: %w", err)
	}
	if len(unreferenced) < 1 {
		return nil
	}

	// the claim has to hold its lock through the deletes, so these do want a
	// transaction.
	return tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		claimed, err := qtx.SelectStepIDsForDelete(ctx, unreferenced)
		if err != nil {
			return fmt.Errorf("claiming steps: %w", err)
		}
		if len(claimed) < 1 {
			return nil
		}

		// raw commands are only kept to enqueue a NotUntil step later. they
		// go with the step, and must go first to satisfy their foreign key.
		if err = qtx.DeleteStepCommandsByStepIDs(ctx, claimed); err != nil {
			return fmt.Errorf("deleting step commands: %w", err)
		}

		if err = qtx.DeleteStepsByStepIDs(ctx, claimed); err != nil {
			return fmt.Errorf("deleting steps: %w", err)
		}
		return nil
	})
}

// StoreStep stores a step and its commands for later state tracking.
// See the storage interface type for further docs.
func (s *MySQLStorage) StoreStep(ctx context.Context, step *storage.StepEnqueuingWithConfig, pushTime time.Time) error {
	err := step.Validate()
	if err != nil {
		return fmt.Errorf("validating step: %w", err)
	}
	return tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		params := sqlc.CreateStepParams{
			WorkflowName: step.WorkflowName,
			InstanceID:   step.InstanceID,
			StepName:     sqlNullString(step.Name),
			NotUntil:     sqlNullTime(step.NotUntil),
			Timeout:      sqlNullTime(step.Timeout),
			Context:      step.Context,
		}
		stepID, err := qtx.CreateStep(ctx, params)
		if err != nil {
			return fmt.Errorf("creating step: %w", err)
		}

		for _, sc := range step.Commands {
			if !step.NotUntil.IsZero() {
				err = qtx.CreateStepCommand(ctx, sqlc.CreateStepCommandParams{
					StepID:      stepID,
					CommandUuid: sc.CommandUUID,
					RequestType: sc.RequestType,
					Command:     sc.Command,
				})
				if err != nil {
					return fmt.Errorf("creating step command: %w", err)
				}
			}
			for _, id := range step.IDs {
				params := sqlc.CreateIDCommandParams{
					EnrollmentID: id,
					CommandUuid:  sc.CommandUUID,
					RequestType:  sc.RequestType,
					StepID:       stepID,
				}
				if step.NotUntil.IsZero() {
					// assume we've successfully pushed
					params.LastPush = sql.NullTime{Valid: true, Time: pushTime}
				}
				if err := qtx.CreateIDCommand(ctx, params); err != nil {
					return fmt.Errorf("creating id command: %w", err)
				}
			}
		}
		return nil
	})
}

// RetrieveOutstandingWorkflowStates finds enrollment IDs with an outstanding workflow step from a given set.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveOutstandingWorkflowStatus(ctx context.Context, workflowName string, ids []string) (outstandingIDs []string, err error) {
	outstandingIDs, err = s.q.GetOutstandingIDs(ctx, sqlc.GetOutstandingIDsParams{
		Ids:          ids,
		WorkflowName: workflowName,
	})
	if err != nil {
		err = fmt.Errorf("getting outstanding ids (%d): %w", len(ids), err)
	}
	return
}

// CancelSteps cancels workflow steps for id.
// See the storage interface type for further docs.
func (s *MySQLStorage) CancelSteps(ctx context.Context, id, workflowName string) error {
	if id == "" {
		return errors.New("must supply both id and workflow name")
	}
	var stepIDs []int64

	err := tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		// note the steps we're about to orphan before we orphan them.
		// finding them again afterwards, by scanning for any step that has
		// no commands, is work proportional to the whole steps table.
		var err error
		if workflowName != "" {
			stepIDs, err = qtx.GetStepIDsByEnrollmentIDAndWorkflowName(ctx, sqlc.GetStepIDsByEnrollmentIDAndWorkflowNameParams{
				EnrollmentID: id,
				WorkflowName: workflowName,
			})
		} else {
			stepIDs, err = qtx.GetStepIDsByEnrollmentID(ctx, id)
		}
		if err != nil {
			return fmt.Errorf("get step ids (%s, %s): %w", id, workflowName, err)
		}

		if len(stepIDs) < 1 {
			return nil
		}

		if workflowName != "" {
			err = qtx.DeleteIDCommandsByWorkflowName(ctx, sqlc.DeleteIDCommandsByWorkflowNameParams{
				EnrollmentID: id,
				WorkflowName: workflowName,
			})
			if err != nil {
				return fmt.Errorf("delete id command by workflow (%s, %s): %w", id, workflowName, err)
			}
		} else {
			if err = qtx.DeleteIDCommands(ctx, id); err != nil {
				return fmt.Errorf("delete id command (%s): %w", id, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// the cancel itself is done; the steps are collected separately so that
	// cancelling one enrollment never waits on another. see
	// collectFinishedSteps.
	//
	// its error is logged rather than returned for the same reason as in the
	// completion path: the cancel has already committed, and the caller
	// abandons the rest of the check-in event if we return an error --
	// leaving in place the workflow status it was about to clear, which is
	// the very thing the cancel exists to remove.
	if err = s.collectFinishedSteps(ctx, stepIDs); err != nil {
		ctxlog.Logger(ctx, s.logger).Info(
			logkeys.Message, "collecting cancelled steps",
			logkeys.EnrollmentID, id,
			logkeys.WorkflowName, workflowName,
			logkeys.GenericCount, len(stepIDs),
			logkeys.Error, err,
		)
	}

	return nil
}

// RetrieveWorkflowStarted returns the last time a workflow was started for id.
func (s *MySQLStorage) RetrieveWorkflowStarted(ctx context.Context, id, workflowName string) (time.Time, error) {
	epoch, err := s.q.GetWorkflowLastStarted(ctx, sqlc.GetWorkflowLastStartedParams{EnrollmentID: id, WorkflowName: workflowName})
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	return time.Unix(epoch, 0), err
}

// RecordWorkflowStarted stores the started time for workflowName for ids.
func (s *MySQLStorage) RecordWorkflowStarted(ctx context.Context, ids []string, workflowName string, started time.Time) error {
	if len(ids) < 1 {
		return errors.New("no id(s) provided")
	}
	const numFields = 3
	const subst = ", (?, ?, ?)"
	parms := make([]interface{}, len(ids)*numFields)
	startedUnix := started.Unix()
	for i, id := range ids {
		// these must match the SQL query, below
		parms[i*numFields] = id
		parms[i*numFields+1] = workflowName
		parms[i*numFields+2] = startedUnix
	}
	values := strings.Repeat(subst, len(ids))[2:]
	_, err := s.db.ExecContext(
		ctx,
		`
INSERT INTO wf_status
  (enrollment_id, workflow_name, last_created_unix)
VALUES
  `+values+` AS new
ON DUPLICATE KEY
UPDATE
  last_created_unix = new.last_created_unix;`,
		parms...,
	)
	return err
}

// ClearWorkflowStatus removes all workflow start times for id.
func (s *MySQLStorage) ClearWorkflowStatus(ctx context.Context, id string) error {
	return s.q.ClearWorkflowStatus(ctx, id)
}
