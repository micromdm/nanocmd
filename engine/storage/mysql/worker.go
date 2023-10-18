package mysql

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/engine/storage/mysql/sqlc"
)

// randHexString generates 40-character string of hex-encoded random data.
func (s *MySQLStorage) randHexString(prefix string) sql.NullString {
	p := make([]byte, 20)
	s.randMu.Lock()
	defer s.randMu.Unlock()
	s.rand.Read(p)
	return sql.NullString{String: prefix + "." + hex.EncodeToString(p), Valid: true}
}

// RetrieveStepsToEnqueue fetches steps to be enqueued that were enqueued "later" with NotUntil.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveStepsToEnqueue(ctx context.Context, pushTime time.Time) ([]*storage.StepEnqueueing, error) {
	if pushTime.IsZero() {
		return nil, errors.New("empty push time")
	}

	var ret []*storage.StepEnqueueing
	err := tx(ctx, s.db, s.q, func(ctx context.Context, qtx *sqlc.Queries) error {

		// this smells like a bad SQL paradigm
		notUntilProcVal := s.randHexString("notu")

		err := qtx.UpdateStepAfterNotUntil(ctx, sqlc.UpdateStepAfterNotUntilParams{
			ProcessID: notUntilProcVal,
			NotUntil:  sql.NullTime{Valid: true, Time: pushTime},
		})
		if err != nil {
			return fmt.Errorf("update step with not until proc (%s): %w", notUntilProcVal.String, err)
		}

		steps, err := qtx.GetStepsByNotUntilProc(ctx, notUntilProcVal)
		if err != nil {
			return fmt.Errorf("get step: %w", err)
		}

		seID := make(map[int64]*storage.StepEnqueueing)
		for _, se := range steps {
			seID[se.ID] = &storage.StepEnqueueing{
				StepContext: storage.StepContext{
					Name:         se.StepName.String,
					WorkflowName: se.WorkflowName,
					InstanceID:   se.InstanceID,
				},
			}
		}

		cmdIDs, err := qtx.GetIDCommandIDsByNotUntilProc(ctx, notUntilProcVal)
		if err != nil {
			return fmt.Errorf("get command ids: %w", err)
		}

		for _, cmdID := range cmdIDs {
			se, ok := seID[cmdID.StepID]
			if !ok || se == nil {
				// TODO: mismatch of step, should we error here?
				continue
			}
			se.IDs = append(se.IDs, cmdID.EnrollmentID)
		}

		cmds, err := qtx.GetStepCommandsByNotUntilProc(ctx, notUntilProcVal)
		if err != nil {
			return fmt.Errorf("get step commands: %w", err)
		}

		for _, cmd := range cmds {
			se, ok := seID[cmd.StepID]
			if !ok || se == nil {
				// TODO: mismatch of step, should we error here?
				continue
			}
			se.Commands = append(se.Commands, storage.StepCommandRaw{
				CommandUUID: cmd.CommandUuid,
				RequestType: cmd.RequestType,
				Command:     cmd.Command,
			})
		}

		for _, v := range seID {
			ret = append(ret, v)
		}

		err = qtx.RemoveStepCommandsByNotUntilProc(ctx, notUntilProcVal)
		if err != nil {
			return fmt.Errorf("remove step commands by not until proc (%s): %w", notUntilProcVal.String, err)
		}

		err = qtx.UpdateLastPushByNotUntilProc(ctx, notUntilProcVal)
		if err != nil {
			return fmt.Errorf("update last push by not until proc (%s): %w", notUntilProcVal.String, err)
		}

		return nil
	})
	return ret, err
}

// RetrieveTimedOutSteps fetches steps that have timed out.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveTimedOutSteps(ctx context.Context) ([]*storage.StepResult, error) {
	var ret []*storage.StepResult

	now := time.Now()

	err := tx(ctx, s.db, s.q, func(ctx context.Context, qtx *sqlc.Queries) error {

		// this smells like a bad SQL paradigm
		timeoutProcVal := s.randHexString("tout")

		err := qtx.UpdateStepAfterTimeout(ctx, sqlc.UpdateStepAfterTimeoutParams{
			ProcessID: timeoutProcVal,
			Timeout:   sql.NullTime{Valid: true, Time: now},
		})
		if err != nil {
			return fmt.Errorf("update step with not until proc (%s): %w", timeoutProcVal.String, err)
		}

		steps, err := qtx.GetStepsByTimeoutProc(ctx, timeoutProcVal)
		if err != nil {
			return fmt.Errorf("get step: %w", err)
		}

		scID := make(map[int64]*storage.StepContext)
		for _, se := range steps {
			scID[se.ID] = &storage.StepContext{
				Name:         se.StepName.String,
				WorkflowName: se.WorkflowName,
				InstanceID:   se.InstanceID,
				Context:      se.Context,
			}
		}

		cmdIDs, err := qtx.GetIDCommandIDsByTimeoutProc(ctx, timeoutProcVal)
		if err != nil {
			return fmt.Errorf("get command ids: %w", err)
		}

		rID := make(map[string]*storage.StepResult)
		for _, cmdID := range cmdIDs {
			sc, ok := scID[cmdID.StepID]
			if !ok || sc == nil {
				// TODO: mismatch of step, should we error here?
				continue
			}
			sr, ok := rID[cmdID.EnrollmentID]
			if !ok || sr == nil {
				sr = &storage.StepResult{
					IDs:         []string{cmdID.EnrollmentID},
					StepContext: *sc,
				}
				rID[cmdID.EnrollmentID] = sr
			}
			scr := storage.StepCommandResult{
				CommandUUID:  cmdID.CommandUuid,
				RequestType:  cmdID.RequestType,
				ResultReport: cmdID.Result,
				Completed:    cmdID.Completed,
			}
			sr.Commands = append(sr.Commands, scr)
		}

		for _, v := range rID {
			ret = append(ret, v)
		}

		err = qtx.RemoveStepCommandsByTimeoutProc(ctx, timeoutProcVal)
		if err != nil {
			return fmt.Errorf("remove step commands by timeout proc (%s): %w", timeoutProcVal.String, err)
		}

		err = qtx.RemoveIDCommandsByTimeoutProc(ctx, timeoutProcVal)
		if err != nil {
			return fmt.Errorf("remove id commands by timeout proc (%s): %w", timeoutProcVal.String, err)
		}

		err = qtx.RemoveStepsByTimeoutProc(ctx, timeoutProcVal)
		if err != nil {
			return fmt.Errorf("remove steps by timeout proc (%s): %w", timeoutProcVal.String, err)
		}

		return nil
	})

	return ret, err
}

// RetrieveAndMarkRePushed retrieves a set of IDs that need to have APNs re-pushes sent.
// See the storage interface type for further docs.
func (s *MySQLStorage) RetrieveAndMarkRePushed(ctx context.Context, ifBefore time.Time, pushTime time.Time) ([]string, error) {
	var ids []string
	ifBeforeTime := sqlNullTime(ifBefore)
	err := tx(ctx, s.db, s.q, func(ctx context.Context, qtx *sqlc.Queries) error {
		var err error
		ids, err = qtx.GetRePushIDs(ctx, ifBeforeTime)
		if err != nil {
			return fmt.Errorf("get repush ids: %w", err)
		}

		err = qtx.UpdateRePushIDs(ctx, sqlc.UpdateRePushIDsParams{
			LastPush: sqlNullTime(pushTime),
			Before:   ifBeforeTime,
		})
		if err != nil {
			return fmt.Errorf("update repush ids: %w", err)
		}

		return nil
	})
	return ids, err
}
