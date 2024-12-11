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

	cmdCt, err := s.q.CountOutstandingIDWorkflowStepCommands(
		ctx,
		sqlc.CountOutstandingIDWorkflowStepCommandsParams{
			EnrollmentID: id,
			CommandUuid:  sc.CommandUUID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("counting outstanding id workflow steps: %w", err)
	}
	if cmdCt.StepID < 1 {
		return nil, fmt.Errorf("no step ID found (id=%s, uuid=%s)", id, sc.CommandUUID)
	}

	if cmdCt.Count > 1 {
		// if there are other uncompleted commands for us for this step
		// then just update this commands results for another command
		// to come in.
		err = s.q.UpdateIDCommand(ctx, sqlc.UpdateIDCommandParams{
			Completed: sc.Completed,
			Result:    sc.ResultReport,
			// where
			EnrollmentID: id,
			CommandUuid:  sc.CommandUUID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating id command: %w", err)
		}
		return nil, nil
	}

	// reaching here implies this is the last command to be completed
	// for the workflow step, for this instance ID for this enrollment ID.

	var ret *storage.StepResult

	err = tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		sd, err := qtx.GetStepByID(ctx, cmdCt.StepID)
		if err != nil {
			return fmt.Errorf("get step by id (%d): %w", cmdCt.StepID, err)
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

		cmdR, err := qtx.GetIDCommandsByStepIDAndLock(ctx, sqlc.GetIDCommandsByStepIDAndLockParams{
			EnrollmentID: id,
			ID:           cmdCt.StepID,
		})
		if err != nil {
			return fmt.Errorf("get id commands by step by id (%d): %w", cmdCt.StepID, err)
		}

		for _, dbSC := range cmdR {
			ret.Commands = append(ret.Commands, storage.StepCommandResult{
				RequestType:  dbSC.RequestType,
				CommandUUID:  dbSC.CommandUuid,
				ResultReport: dbSC.Result,
				Completed:    true,
			})
		}

		err = qtx.RemoveIDCommandsByStepID(ctx, sqlc.RemoveIDCommandsByStepIDParams{
			EnrollmentID: id,
			StepID:       cmdCt.StepID,
		})
		if err != nil {
			return fmt.Errorf("remove id commands by step by id (%d): %w", cmdCt.StepID, err)
		}

		err = qtx.DeleteWorkflowStepHavingNoCommandsByStepID(ctx, cmdCt.StepID)
		if err != nil {
			return fmt.Errorf("delete workflow with no commands (%d): %w", cmdCt.StepID, err)
		}

		return nil
	})
	if err != nil {
		return ret, fmt.Errorf("tx step completed: %w", err)
	}
	return ret, nil
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
	return tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		if workflowName != "" {
			err := qtx.DeleteIDCommandByWorkflow(ctx, sqlc.DeleteIDCommandByWorkflowParams{
				EnrollmentID: id,
				WorkflowName: workflowName,
			})
			if err != nil {
				return fmt.Errorf("delete id command by workflow (%s, %s): %w", id, workflowName, err)
			}
		} else {
			err := qtx.DeleteIDCommands(ctx, id)
			if err != nil {
				return fmt.Errorf("delete id command (%s): %w", id, err)
			}
		}

		err := qtx.DeleteUnusedStepCommands(ctx)
		if err != nil {
			return fmt.Errorf("delete unused step commands: %w", err)
		}

		if workflowName != "" {
			err = qtx.DeleteWorkflowStepHavingNoCommandsByWorkflowName(ctx, workflowName)
			if err != nil {
				return fmt.Errorf("delete workflow step having no commands (%s): %w", workflowName, err)
			}
		} else {
			if err = qtx.DeleteWorkflowStepHavingNoCommands(ctx); err != nil {
				return fmt.Errorf("delete workflow step having no commands (%s): %w", workflowName, err)
			}
		}
		return nil
	})
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
