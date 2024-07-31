package test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
)

// stepCmdWithUUID tries to find the StepCommand having uuid within step.
func stepCmdWithUUID(step *storage.StepResult, uuid string) (storage.StepCommandResult, bool) {
	for _, cmd := range step.Commands {
		if cmd.CommandUUID == uuid {
			return cmd, true
		}
	}
	return storage.StepCommandResult{}, false
}

func TestEngineStorage(t *testing.T, newStorage func() storage.AllStorage) {
	s := newStorage()
	mainTest(t, s)

	t.Run("testEngineStorageNotUntil", func(t *testing.T) {
		testEngineStorageNotUntil(t, s)
	})

	t.Run("testEngineStepTimeout", func(t *testing.T) {
		testEngineStepTimeout(t, s)
	})

	t.Run("testRepush", func(t *testing.T) {
		testRepush(t, newStorage())
	})

	t.Run("testOutstanding", func(t *testing.T) {
		testOutstanding(t, s)
	})

	t.Run("testEvent", func(t *testing.T) {
		TestEventStorage(t, s)
	})

	ctx := context.Background()

	t.Run("testEventStatus", func(t *testing.T) {
		TestEventStatusStorage(t, ctx, s)
	})
}

func mainTest(t *testing.T, s storage.AllStorage) {
	ctx := context.Background()

	type responseTest struct {
		testName          string
		resp              *storage.StepCommandResult
		shouldBeCompleted bool
		shouldError       bool
		reqType           string
		skipReqType       bool
		id                string // enrollment id
		skipCmdLen        bool
		skipByteCompare   bool
	}

	fakeID := "456DFB"

	// yikes! bit of a beast, these tests
	for _, tStep := range []struct {
		testName    string
		step        *storage.StepEnqueuingWithConfig
		shouldError bool
		respSteps   []responseTest
	}{
		{
			"nil_step",
			nil,
			true,
			nil,
		},
		{
			"nil_id",
			&storage.StepEnqueuingWithConfig{},
			true,
			nil,
		},
		{
			"normal_test1_command_multi_id",
			&storage.StepEnqueuingWithConfig{
				StepEnqueueing: storage.StepEnqueueing{
					IDs: []string{"AAA111", "BBB222"},
					StepContext: storage.StepContext{
						WorkflowName: "workflow.name.test1",
						InstanceID:   "B",
					},
					Commands: []storage.StepCommandRaw{
						{
							CommandUUID: "UUID-2",
							RequestType: "DeviceInformation",
						},
					},
				},
			},
			false,
			[]responseTest{
				{
					testName: "UUID-2-testResp1",
					resp: &storage.StepCommandResult{
						CommandUUID:  "UUID-2",
						Completed:    true,
						ResultReport: []byte("Resp1-UUID-1"),
					},
					shouldBeCompleted: true,
					shouldError:       false,
					reqType:           "DeviceInformation",
					id:                "AAA111",
				},
				{
					testName: "UUID-2-testResp1-2nd",
					resp: &storage.StepCommandResult{
						CommandUUID:  "UUID-2",
						Completed:    true,
						ResultReport: []byte("Resp2-UUID-1"),
					},
					shouldBeCompleted: true,
					shouldError:       false,
					reqType:           "DeviceInformation",
					id:                "BBB222",
				},
				{
					// should fail (duplicate response for same id)
					testName: "UUID-2-testResp1-3rd-dup",
					resp: &storage.StepCommandResult{
						CommandUUID:  "UUID-2",
						Completed:    true,
						ResultReport: []byte("Resp2-UUID-1"),
					},
					shouldBeCompleted: false,
					shouldError:       true,
					id:                "BBB222",
					skipByteCompare:   false,
				},
			},
		},
		{
			"normal_test1_command_fail_resp",
			&storage.StepEnqueuingWithConfig{
				StepEnqueueing: storage.StepEnqueueing{
					IDs: []string{fakeID},
					StepContext: storage.StepContext{
						WorkflowName: "workflow.name.test1",
						InstanceID:   "C",
					},
					Commands: []storage.StepCommandRaw{
						{
							CommandUUID: "UUID-3",
							RequestType: "DeviceInformation",
						},
					},
				},
			},
			false,
			[]responseTest{
				{
					testName: "UUID-1-testResp1",
					resp: &storage.StepCommandResult{
						CommandUUID:  "UUID-NotFound",
						Completed:    true,
						ResultReport: []byte("UUID-NotFound"),
					},
					shouldBeCompleted: false,
					shouldError:       true,
					id:                fakeID,
				},
			},
		},
		{
			"multi-command-single-id",
			&storage.StepEnqueuingWithConfig{
				StepEnqueueing: storage.StepEnqueueing{
					IDs: []string{"AAA111"},
					StepContext: storage.StepContext{
						WorkflowName: "workflow.name.test1",
						InstanceID:   "D",
					},
					Commands: []storage.StepCommandRaw{
						{
							CommandUUID: "X-UUID-1",
							RequestType: "DeviceInformation",
						},
						{
							CommandUUID: "X-UUID-2",
							RequestType: "SecurityInfo",
						},
					},
				},
			},
			false,
			[]responseTest{
				{
					testName: "resp1",
					resp: &storage.StepCommandResult{
						CommandUUID:  "X-UUID-1",
						Completed:    true,
						ResultReport: []byte("Resp1-UUID-1"),
					},
					shouldBeCompleted: false,
					shouldError:       false,
					reqType:           "DeviceInformation",
					id:                "AAA111",
				},
				{
					testName: "resp2",
					resp: &storage.StepCommandResult{
						CommandUUID:  "X-UUID-2",
						Completed:    true,
						ResultReport: []byte("Resp2-UUID-2"),
					},
					shouldBeCompleted: true,
					shouldError:       false,
					reqType:           "SecurityInfo",
					id:                "AAA111",
				},
			},
		},

		{
			// this caused a crash in the inmem storage
			"multi-command-same-uuid",
			&storage.StepEnqueuingWithConfig{
				StepEnqueueing: storage.StepEnqueueing{
					IDs: []string{"CCC222"},
					StepContext: storage.StepContext{
						WorkflowName: "workflow.name.test1",
						InstanceID:   "E",
					},
					Commands: []storage.StepCommandRaw{
						{
							CommandUUID: "W-UUID-1",
							RequestType: "DeviceInformation",
						},
						{
							CommandUUID: "W-UUID-1",
							RequestType: "SecurityInfo",
						},
					},
				},
			},
			true,
			[]responseTest{
				{
					testName: "resp1",
					resp: &storage.StepCommandResult{
						CommandUUID:  "W-UUID-1",
						Completed:    true,
						RequestType:  "DeviceInformation",
						ResultReport: []byte("W-UUID-1"),
					},
					shouldBeCompleted: false,
					shouldError:       true,
					skipReqType:       true,
					skipCmdLen:        true,
					reqType:           "DeviceInformation",
					id:                "CCC222",
				},
			},
		},
	} {
		t.Run("step-"+tStep.testName, func(t *testing.T) {
			// if err := tStep.step.Validate(); err != nil {
			// 	t.Fatalf("invalid test data: step enqueueing with config: %v", err)
			// }

			// some backends may truncate the time and drop TZ
			// so let's truncate ourselves and eliminate the TZ.
			// since this value is used to compare the retrived value
			// we'll stick with that.
			storedAt := time.Now().UTC().Truncate(time.Second)

			err := s.StoreStep(ctx, tStep.step, storedAt)
			if tStep.shouldError && err == nil {
				t.Fatalf("StoreStep: expected error; step=%v", tStep.step)
			} else if !tStep.shouldError && err != nil {
				t.Fatalf("StoreStep: expected no error; step=%v err=%v", tStep.step, err)
			}

			if err != nil && tStep.step != nil {
				if len(tStep.step.IDs) > 0 {
					err = s.RecordWorkflowStarted(ctx, tStep.step.IDs, tStep.step.WorkflowName, storedAt)
					if err != nil {
						t.Errorf("RecordWorkflowStarted: error for step=%s: %v", tStep.step.WorkflowName, err)
					}
				}

				for _, id := range tStep.step.IDs {
					ts, err := s.RetrieveWorkflowStarted(ctx, id, tStep.step.WorkflowName)
					if err != nil {
						t.Fatalf("RetrieveWorkflowStarted: error for id=%s, step=%s err=%v", id, tStep.step.WorkflowName, err)
					}
					if ts.IsZero() {
						t.Errorf("RetrieveWorkflowStarted: nil timestamp for id=%s, step=%s err=%v", id, tStep.step.WorkflowName, err)
					} else if ts != storedAt {
						t.Errorf("RetrieveWorkflowStarted: timestamp mismatch for id=%s, step=%s expected=%v got=%v", id, tStep.step.WorkflowName, storedAt, ts)
					}
				}
			}

			for _, tRespStep := range tStep.respSteps {
				t.Run("cmd-resp-"+tRespStep.testName, func(t *testing.T) {
					reqType, _, err := s.RetrieveCommandRequestType(ctx, tRespStep.id, tRespStep.resp.CommandUUID)
					if err != nil {
						t.Fatalf("err looking up request type for uuid=%s: %v", tRespStep.resp.CommandUUID, err)
					}

					if have, want := reqType, tRespStep.reqType; !tRespStep.skipReqType && have != want {
						t.Errorf("request type does not match; have: %s, want: %s", have, want)
					}

					if err = tRespStep.resp.Validate(); err != nil {
						t.Fatalf("invalid test data: step command result: %v", err)
					}

					completedStep, err := s.StoreCommandResponseAndRetrieveCompletedStep(ctx, tRespStep.id, tRespStep.resp)

					if tRespStep.shouldError && err == nil {
						t.Errorf("StoreCommandResponseAndRetrieveCompletedStep: expected error for resp; command=%v", tRespStep.resp)
					} else if !tRespStep.shouldError && err != nil {
						t.Fatalf("StoreCommandResponseAndRetrieveCompletedStep: expected no error for resp; command=%v, err=%v", tRespStep.resp, err)
					}

					if tRespStep.shouldBeCompleted && completedStep == nil {
						t.Errorf("StoreCommandResponseAndRetrieveCompletedStep: found incomplete, expected complete; command=%v", tRespStep.resp)
					} else if !tRespStep.shouldBeCompleted && completedStep != nil {
						t.Errorf("StoreCommandResponseAndRetrieveCompletedStep: found complete, expected incomplete; command=%v completed=%v", tRespStep.resp, completedStep)
					}

					if completedStep != nil {
						if want, have := len(tStep.step.Commands), len(completedStep.Commands); !tRespStep.skipCmdLen && have != want {
							t.Errorf("mismatch of returned commands; have %d, wanted %d", have, want)
						}

						for _, eCmd := range tStep.respSteps {
							sc, ok := stepCmdWithUUID(completedStep, eCmd.resp.CommandUUID)
							if !ok {
								t.Errorf("command uuid not found: uuid=%v", eCmd.resp.CommandUUID)
							} else {
								if have, want := sc.ResultReport, eCmd.resp.ResultReport; eCmd.skipByteCompare && bytes.Compare(have, want) != 0 {
									t.Errorf("command result raw does not match: have=%s, want=%s", string(have), string(want))
								}

							}

						}
					}
				})
			}

			if tStep.step != nil && len(tStep.step.IDs) > 0 {
				// clear just this workflow
				err = s.CancelSteps(ctx, tStep.step.IDs[0], tStep.step.WorkflowName)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func testOutstanding(t *testing.T, s storage.AllStorage) {
	ctx := context.Background()

	enq := &storage.StepEnqueuingWithConfig{
		StepEnqueueing: storage.StepEnqueueing{
			IDs: []string{"EnrollmentID-4", "EnrollmentID-5"},
			StepContext: storage.StepContext{
				WorkflowName: "workflow.name.test1",
				InstanceID:   "InstanceID-1",
			},
			Commands: []storage.StepCommandRaw{
				{
					CommandUUID: "UUID-1",
					RequestType: "DeviceInformation",
					Command:     []byte("Command-1"),
				},
			},
		},
	}

	err := s.StoreStep(ctx, enq, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	outstandingIDs, err := s.RetrieveOutstandingWorkflowStatus(ctx, enq.WorkflowName, enq.IDs)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(outstandingIDs), 2; have != want {
		t.Errorf("have: %v, want: %v: %v", have, want, outstandingIDs)
	}

	// complete one of the commands
	_, err = s.StoreCommandResponseAndRetrieveCompletedStep(ctx, enq.IDs[0], &storage.StepCommandResult{
		CommandUUID:  "UUID-1",
		RequestType:  "DeviceInformation",
		ResultReport: []byte("Result-1"),
		Completed:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	outstandingIDs, err = s.RetrieveOutstandingWorkflowStatus(ctx, enq.WorkflowName, enq.IDs)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(outstandingIDs), 1; have != want {
		t.Fatalf("have: %v, want: %v: %v", have, want, outstandingIDs)
	}

	err = s.CancelSteps(ctx, outstandingIDs[0], "invalid.workflow.name")
	if err != nil {
		t.Fatal(err)
	}

	outstandingIDs, err = s.RetrieveOutstandingWorkflowStatus(ctx, enq.WorkflowName, []string{outstandingIDs[0]})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(outstandingIDs), 1; have != want {
		t.Fatalf("have: %v, want: %v: %v", have, want, outstandingIDs)
	}

	err = s.CancelSteps(ctx, outstandingIDs[0], enq.WorkflowName)
	if err != nil {
		t.Fatal(err)
	}

	outstandingIDs, err = s.RetrieveOutstandingWorkflowStatus(ctx, enq.WorkflowName, []string{outstandingIDs[0]})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(outstandingIDs), 0; have != want {
		t.Errorf("have: %v, want: %v: %v", have, want, outstandingIDs)
	}
}
