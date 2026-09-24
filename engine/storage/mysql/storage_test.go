package mysql

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"

	_ "github.com/go-sql-driver/mysql"
)

// newTestStorage returns storage against the test DSN, skipping if unset.
func newTestStorage(t *testing.T) *MySQLStorage {
	t.Helper()
	dsn := os.Getenv("NANOCMD_MYSQL_STORAGE_TEST_DSN")
	if dsn == "" {
		t.Skip("NANOCMD_MYSQL_STORAGE_TEST_DSN not set")
	}
	s, err := New(WithDSN(dsn))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// countWorkflowRows counts the rows left for workflowName. Other tests share
// this database and deliberately leave steps outstanding, so the counts are
// scoped to the calling test's own workflow rather than to the whole table.
func countWorkflowRows(t *testing.T, s *MySQLStorage, workflowName string) (steps, idCommands, stepCommands int) {
	t.Helper()
	const q = `
SELECT
  (SELECT COUNT(*) FROM steps WHERE workflow_name = ?),
  (SELECT COUNT(*) FROM id_commands c
     JOIN steps s ON c.step_id = s.id WHERE s.workflow_name = ?),
  (SELECT COUNT(*) FROM step_commands sc
     JOIN steps s ON sc.step_id = s.id WHERE s.workflow_name = ?)`
	err := s.db.QueryRow(q, workflowName, workflowName, workflowName).
		Scan(&steps, &idCommands, &stepCommands)
	if err != nil {
		t.Fatal(err)
	}
	return
}

// TestNotUntilStepCompletes completes a step that was enqueued with a
// NotUntil, which is the only kind of step that has step_commands rows.
func TestNotUntilStepCompletes(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	const (
		id           = "NotUntilCompletes-ID"
		workflowName = "workflow.name.notuntil.completes"
	)
	uuids := []string{"NUC-UUID-1", "NUC-UUID-2", "NUC-UUID-3"}

	step := &storage.StepEnqueuingWithConfig{
		StepEnqueueing: storage.StepEnqueueing{
			IDs: []string{id},
			StepContext: storage.StepContext{
				WorkflowName: workflowName,
				InstanceID:   "notuntil-completes",
			},
			Commands: []storage.StepCommandRaw{
				{CommandUUID: uuids[0], RequestType: "DeviceInformation", Command: []byte("c1")},
				{CommandUUID: uuids[1], RequestType: "SecurityInfo", Command: []byte("c2")},
				{CommandUUID: uuids[2], RequestType: "ProfileList", Command: []byte("c3")},
			},
		},
		NotUntil: time.Now().Add(time.Hour),
	}
	if err := s.StoreStep(ctx, step, time.Now()); err != nil {
		t.Fatal(err)
	}

	var completed *storage.StepResult
	for _, uuid := range uuids {
		r, err := s.StoreCommandResponseAndRetrieveCompletedStep(ctx, id, &storage.StepCommandResult{
			CommandUUID:  uuid,
			Completed:    true,
			ResultReport: []byte("result-" + uuid),
		})
		if err != nil {
			t.Fatalf("responding to %s: %v", uuid, err)
		}
		if r != nil {
			completed = r
		}
	}

	if completed == nil {
		t.Fatal("step never reported as completed")
	}
	if have, want := len(completed.Commands), len(uuids); have != want {
		for _, c := range completed.Commands {
			t.Logf("  uuid=%s request_type=%s", c.CommandUUID, c.RequestType)
		}
		t.Errorf("returned command results: have %d, want %d", have, want)
	}

	steps, idc, sc := countWorkflowRows(t, s, workflowName)
	if steps != 0 || idc != 0 || sc != 0 {
		t.Errorf("rows left after completion: steps=%d id_commands=%d step_commands=%d", steps, idc, sc)
	}
}

// TestConcurrentCommandsOfOneEnrollment answers all of an enrollment's
// commands for a step at the same time.
//
// MDM does not do this: an enrollment reports one command result at a time.
// So this asserts a storage invariant rather than reproducing a live failure
// -- that the storage decides step completion from rows it holds, and does
// not silently depend on its caller for the serialization. Exactly one of the
// responses has to come back as the completed step; if each decides
// separately that another is still outstanding, none of them completes it and
// the step is stranded with nothing left to finish it.
func TestConcurrentCommandsOfOneEnrollment(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	const (
		enrollments  = 10
		cmdsEach     = 4
		workflowName = "workflow.name.concurrent.commands"
	)

	type response struct{ id, uuid string }
	var responses []response
	for i := 0; i < enrollments; i++ {
		id := fmt.Sprintf("Concurrent-ID-%03d", i)
		var cmds []storage.StepCommandRaw
		for c := 0; c < cmdsEach; c++ {
			uuid := fmt.Sprintf("Concurrent-UUID-%03d-%d", i, c)
			cmds = append(cmds, storage.StepCommandRaw{CommandUUID: uuid, RequestType: "DeviceInformation"})
			responses = append(responses, response{id, uuid})
		}
		step := &storage.StepEnqueuingWithConfig{
			StepEnqueueing: storage.StepEnqueueing{
				IDs: []string{id},
				StepContext: storage.StepContext{
					WorkflowName: workflowName,
					InstanceID:   fmt.Sprintf("concurrent-%d", i),
				},
				Commands: cmds,
			},
		}
		if err := s.StoreStep(ctx, step, time.Now()); err != nil {
			t.Fatal(err)
		}
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		errs    []error
		nDone   int
		ready   sync.WaitGroup
		release = make(chan struct{})
	)
	ready.Add(len(responses))
	for _, r := range responses {
		wg.Add(1)
		go func(r response) {
			defer wg.Done()
			ready.Done()
			<-release
			res, err := s.StoreCommandResponseAndRetrieveCompletedStep(ctx, r.id, &storage.StepCommandResult{
				CommandUUID:  r.uuid,
				Completed:    true,
				ResultReport: []byte("result"),
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			}
			if res != nil {
				nDone++
				if have, want := len(res.Commands), cmdsEach; have != want {
					errs = append(errs, fmt.Errorf("command results: have %d, want %d", have, want))
				}
			}
		}(r)
	}

	ready.Wait()
	close(release)
	wg.Wait()

	for _, err := range errs {
		t.Errorf("responding: %v", err)
	}
	if have, want := nDone, enrollments; have != want {
		t.Errorf("completed steps: have %d, want %d", have, want)
	}

	steps, idc, sc := countWorkflowRows(t, s, workflowName)
	if steps != 0 || idc != 0 || sc != 0 {
		t.Errorf("rows left behind: steps=%d id_commands=%d step_commands=%d", steps, idc, sc)
	}
}

// TestStepFanOutCompletes has every enrollment of one step respond at the same
// time, which is what happens when a step is enqueued to many enrollments at
// once. All of them share a single steps row, so all of them contend for it.
func TestStepFanOutCompletes(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	const (
		fanOut       = 25
		cmdUUID      = "FanOut-UUID-1"
		workflowName = "workflow.name.fanout"
	)

	ids := make([]string, fanOut)
	for i := range ids {
		ids[i] = fmt.Sprintf("FanOut-ID-%03d", i)
	}
	step := &storage.StepEnqueuingWithConfig{
		StepEnqueueing: storage.StepEnqueueing{
			IDs: ids,
			StepContext: storage.StepContext{
				WorkflowName: workflowName,
				InstanceID:   "fanout",
			},
			Commands: []storage.StepCommandRaw{
				{CommandUUID: cmdUUID, RequestType: "DeviceInformation"},
			},
		},
	}
	if err := s.StoreStep(ctx, step, time.Now()); err != nil {
		t.Fatal(err)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		errs    []error
		nDone   int
		ready   sync.WaitGroup
		release = make(chan struct{})
	)
	ready.Add(fanOut)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			ready.Done()
			<-release
			r, err := s.StoreCommandResponseAndRetrieveCompletedStep(ctx, id, &storage.StepCommandResult{
				CommandUUID:  cmdUUID,
				Completed:    true,
				ResultReport: []byte("result"),
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			}
			if r != nil {
				nDone++
			}
		}(id)
	}

	// release them together so the transactions overlap
	ready.Wait()
	close(release)
	wg.Wait()

	for _, err := range errs {
		t.Errorf("responding: %v", err)
	}
	if have, want := nDone, fanOut; have != want {
		t.Errorf("completed steps: have %d, want %d", have, want)
	}

	steps, idc, sc := countWorkflowRows(t, s, workflowName)
	if steps != 0 || idc != 0 || sc != 0 {
		t.Errorf("rows left after fan-out: steps=%d id_commands=%d step_commands=%d", steps, idc, sc)
	}
}
