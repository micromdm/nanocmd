package certprof

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/engine"
	enginestorage "github.com/micromdm/nanocmd/engine/storage/inmem"
	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/subsystem/profile/storage/inmem"
	"github.com/micromdm/nanocmd/utils/mobileconfig"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow/test"
)

func loadTestProfile(ctx context.Context, s storage.Storage, name string) error {
	var profile mobileconfig.Mobileconfig
	var err error

	profile, err = os.ReadFile("testdata/scep.mobileconfig")
	if err != nil {
		return err
	}

	payload, _, err := profile.Parse()
	if err != nil {
		return err
	}

	return s.StoreProfile(
		ctx,
		name,
		storage.ProfileInfo{Identifier: payload.PayloadIdentifier, UUID: payload.PayloadUUID},
		profile,
	)
}

func TestWorkflow(t *testing.T) {
	e := engine.New(enginestorage.New(), &test.NullEnqueuer{})

	c := test.NewCollectingStepEnqueur(e)

	s := inmem.New()

	ctx := context.Background()

	err := loadTestProfile(ctx, s, "scep")
	if err != nil {
		t.Fatal()
	}

	w, err := New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	w.ider = uuid.NewStaticIDs(
		// note: order is important and depends on values in plist testdata
		"CERT-LIST-01",
		"D053EE3A-EE0E-4E82-87D2-BB3624186694",
	)

	// enrollment id
	id := "6362F867-FFF2-4EA6-905C-3C796DF4EF68"

	e.RegisterWorkflow(w)

	// read it back out of the engine
	w2 := e.Workflow(w.Name()).(*Workflow)

	if w.Name() != w2.Name() {
		t.Fatal("workflow name not equal after registration")
	}

	const testCtx = `{
    "criteria": {
        "always_replace": true
    },
    "filter": {
        "cn_prefix": "?Error"
    },
    "profile": "scep",
    "text_replacements": {
        "%CN%": "324429DE-46B4-4209-AF95-82D9D8942A0E",
        "%CHAL%": "D6E71C89-4831-4B88-9F1F-83714D526C63"
    }
}`

	_, err = e.StartWorkflow(ctx, w.Name(), []byte(testCtx), []string{id}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	steps := c.Steps()

	// make sure at least one step has been accumulated. the cert list.
	if want, have := 1, len(steps); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	step := steps[0]

	// make sure that step has ids
	if want, have := 1, len(step.StepEnqueueing.IDs); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}
	// make sure the first id is the id the workflow was started for
	if want, have := id, step.StepEnqueueing.IDs[0]; want != have {
		t.Errorf("wanted: %s; have: %s", want, have)
	}
	// make sure the enqueued step had one command
	if want, have := 1, len(step.Commands); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	err = test.SendCommandEvent(ctx, e, "testdata/certlist.plist", id, "CERT-LIST-01")
	if err != nil {
		t.Fatal(err)
	}

	steps = c.Steps()

	// should have acculated two steps now. the install profile.
	if want, have := 2, len(steps); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	step = steps[1]

	// make sure that step has ids
	if want, have := 1, len(step.StepEnqueueing.IDs); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}
	// make sure the first id is the id the workflow was started for
	if want, have := id, step.StepEnqueueing.IDs[0]; want != have {
		t.Errorf("wanted: %s; have: %s", want, have)
	}
	// make sure the enqueued step had one command
	if want, have := 1, len(step.Commands); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	cmd := step.Commands[0].(*mdmcommands.InstallProfileCommand)
	installProfile := mobileconfig.Mobileconfig(cmd.Command.Payload)
	_, _, err = installProfile.Parse()
	if err != nil {
		t.Error(err)
	}

	// validate UUID (second in static list, above)
	if want, have := "D053EE3A-EE0E-4E82-87D2-BB3624186694", cmd.CommandUUID; want != have {
		t.Errorf("wanted: %s; have: %s", want, have)
	}

	// verify profile has the replaced values
	if !bytes.Contains(installProfile, []byte("<string>324429DE-46B4-4209-AF95-82D9D8942A0E</string>")) {
		t.Error("profile does not contain magic value 1")
	}
	if !bytes.Contains(installProfile, []byte("<string>D6E71C89-4831-4B88-9F1F-83714D526C63</string>")) {
		t.Error("profile does not contain magic value 1")
	}

	err = test.SendCommandEvent(ctx, e, "testdata/resp-ack.plist", id, "D053EE3A-EE0E-4E82-87D2-BB3624186694")
	if err != nil {
		t.Fatal(err)
	}
}
