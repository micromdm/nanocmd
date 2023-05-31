package test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
)

func testEngineStorageNotUntil(t *testing.T, s storage.AllStorage) {
	ctx := context.Background()

	for _, test := range []struct {
		testName string
		steps    []*storage.StepEnqueuingWithConfig

		stepsWanted   int
		stepsWanted2  int
		reqTypeWanted string
	}{
		{
			testName: "in-the-past",
			steps: []*storage.StepEnqueuingWithConfig{
				{
					StepEnqueueing: storage.StepEnqueueing{
						IDs: []string{"EnrollmentID-1", "EnrollmentID-2"},
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
					NotUntil: time.Now().Add(-time.Minute), // by setting NotUntil we should register this
				},
			},
			stepsWanted:   1,
			stepsWanted2:  0,
			reqTypeWanted: "DeviceInformation",
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			for _, step := range test.steps {
				err := s.StoreStep(ctx, step, time.Now())
				if err != nil {
					t.Fatal(err)
				}
			}

			steps, err := s.RetrieveStepsToEnqueue(ctx, time.Now())
			if err != nil {
				t.Fatal(err)
			}

			if have, want := len(steps), test.stepsWanted; have != want {
				t.Fatalf("expected steps: have %v, want %v", have, want)
			}

			if len(steps) >= 1 && test.reqTypeWanted != "" && len(steps[0].Commands) < 1 {
				t.Error("request type wanted, but no commands returned")
			}

			if len(steps) >= 1 && len(steps[0].Commands) >= 1 {
				if have, want := steps[0].Commands[0].RequestType, test.reqTypeWanted; have != want {
					t.Errorf("expected request type: have %v, want %v", have, want)
				}
			}

			for _, step := range steps {
				for _, cmd := range step.Commands {
					if len(cmd.Command) < 1 {
						t.Errorf("no command bytes for NotUntil command?")
					}
				}
				// regression test
				if len(step.IDs) < 1 {
					t.Error("no IDs for step")
				}
			}

			steps, err = s.RetrieveStepsToEnqueue(ctx, time.Now())
			if err != nil {
				t.Fatal(err)
			}

			if have, want := len(steps), test.stepsWanted2; have != want {
				t.Fatalf("expected steps (2nd): have %v, want %v", have, want)
			}
		})
	}
}

func testEngineStepTimeout(t *testing.T, s storage.AllStorage) {
	ctx := context.Background()

	type response struct {
		id        string
		sc        storage.StepCommandResult
		completed bool
	}

	for _, test := range []struct {
		testName string
		steps    []*storage.StepEnqueuingWithConfig
		resps    []response

		stepsWanted   int
		stepsWanted2  int
		stepsWanted3  int
		reqTypeWanted string
	}{
		{
			testName: "timeout-test-1",
			steps: []*storage.StepEnqueuingWithConfig{
				{
					StepEnqueueing: storage.StepEnqueueing{
						IDs: []string{"EnrollmentID-1", "EnrollmentID-2", "EnrollmentID-3"},
						StepContext: storage.StepContext{
							WorkflowName: "workflow.name.test1",
							InstanceID:   "InstanceID-1",
						},
						Commands: []storage.StepCommandRaw{
							{
								CommandUUID: "UUID-1",
								RequestType: "DeviceInformation",
							},
						},
					},
					Timeout: time.Now().Add(-time.Minute),
				},
			},
			resps: []response{
				{
					id: "EnrollmentID-1",
					sc: storage.StepCommandResult{
						CommandUUID:  "UUID-1",
						RequestType:  "DeviceInformation",
						ResultReport: []byte("Command-1"),
						Completed:    true,
					},
					completed: true,
				},
			},
			stepsWanted:   2,
			stepsWanted2:  0,
			reqTypeWanted: "DeviceInformation",
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			for _, step := range test.steps {
				err := s.StoreStep(ctx, step, time.Now())
				if err != nil {
					t.Fatal(err)
				}
			}

			for _, response := range test.resps {
				step, err := s.StoreCommandResponseAndRetrieveCompletedStep(ctx, response.id, &response.sc)
				if err != nil {
					t.Fatal(err)
				}
				if want, have := response.completed, step != nil; have != want {
					t.Errorf("mismatched completed; have: %v, want: %v", have, want)
				}
			}

			steps, err := s.RetrieveTimedOutSteps(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := len(steps), test.stepsWanted; have != want {
				t.Fatalf("expected steps: have: %v, want: %v", have, want)
			}

			for _, step := range steps {
				if step.WorkflowName == "" {
					t.Error("empty workflow name")
				}
			}

			if len(steps) >= 1 && test.reqTypeWanted != "" && len(steps[0].Commands) < 1 {
				t.Error("request type wanted, but no commands returned")
			}

			if len(steps) >= 1 && len(steps[0].Commands) >= 1 {
				if have, want := steps[0].Commands[0].RequestType, test.reqTypeWanted; have != want {
					t.Errorf("expected request type: have: %v, want: %v", have, want)
				}
			}

			steps, err = s.RetrieveTimedOutSteps(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := len(steps), test.stepsWanted2; have != want {
				t.Fatalf("expected steps (2nd): have %v, want %v", have, want)
			}
		})
	}
}

func testRepush(t *testing.T, s storage.AllStorage) {
	ctx := context.Background()
	enq := &storage.StepEnqueuingWithConfig{
		StepEnqueueing: storage.StepEnqueueing{
			IDs: []string{"EnrollmentID-1", "EnrollmentID-2"},
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
		// NotUntil: not setting NotUntil to sure these are simulated to be sent pushes "now"
	}

	now := time.Now()

	err := s.StoreStep(ctx, enq, now)
	if err != nil {
		t.Fatal(err)
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

	ifBefore := now.Add(time.Second)
	now = ifBefore.Add(time.Second)

	ids, err := s.RetrieveAndMarkRePushed(ctx, ifBefore, now)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := ids, []string{enq.IDs[1]}; !reflect.DeepEqual(have, want) {
		t.Errorf("have: %v, want: %v", have, want)
	}

	ids, err = s.RetrieveAndMarkRePushed(ctx, ifBefore, now)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ids), 0; have != want {
		t.Errorf("have: %v, want: %v: %v", have, want, ids)
	}

	enq2 := &storage.StepEnqueuingWithConfig{
		StepEnqueueing: storage.StepEnqueueing{
			IDs: []string{"EnrollmentID-3"},
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
		NotUntil: time.Now().Add(-time.Minute),
	}

	err = s.StoreStep(ctx, enq2, now)
	if err != nil {
		t.Fatal(err)
	}

	ids, err = s.RetrieveAndMarkRePushed(ctx, ifBefore, now)
	if err != nil {
		t.Fatal(err)
	}

	// should still be zero because we haven't enqueued anything (NotUntil on last command)
	// and our dates have not changed
	if have, want := len(ids), 0; have != want {
		t.Errorf("have: %v, want: %v: %v", have, want, ids)
	}

	// "enqueue" our NotUntils command in enq2
	_, err = s.RetrieveStepsToEnqueue(ctx, now)
	if err != nil {
		t.Fatal(err)
	}

	ifBefore = now.Add(time.Second)
	now = ifBefore.Add(time.Second)

	ids, err = s.RetrieveAndMarkRePushed(ctx, ifBefore, now)
	if err != nil {
		t.Fatal(err)
	}

	find := []string{enq.IDs[1], enq2.IDs[0]}
	sort.Strings(ids)
	sort.Strings(find)
	if have, want := ids, find; !reflect.DeepEqual(have, want) {
		t.Errorf("have: %v, want: %v", have, want)
	}

	ids, err = s.RetrieveAndMarkRePushed(ctx, ifBefore, now)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(ids), 0; have != want {
		t.Errorf("have: %v, want: %v: %v", have, want, ids)
	}
}
