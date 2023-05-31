package engine

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

type testContextMarshaler struct{}

func (cm *testContextMarshaler) NewContextValue(stepName string) workflow.ContextMarshaler {
	if stepName == "" {
		return new(workflow.StringContext)
	}
	return nil
}

func TestConvertWorkflowStepStartFromEngine(t *testing.T) {
	instID := "AAABBB111222"

	newCtxMarshaler := &testContextMarshaler{}

	inputIDs := []string{"a", "b", "c"}

	ss, err := workflowStepStartFromEngine(instID, newCtxMarshaler, []byte("hello"), inputIDs, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	stepCtx, ok := ss.Context.(*workflow.StringContext)
	if !ok {
		t.Fatal("incorrect context type")
	}

	if have, want := string(*stepCtx), "hello"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if have, want := ss.InstanceID, instID; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if have, want := ss.Name, ""; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if !reflect.DeepEqual(ss.IDs, inputIDs) {
		t.Error("IDs do not match")
	}
}

func TestStorageStepCommandFromRawResponse(t *testing.T) {
	pBytes, err := os.ReadFile("testdata/devinfo.plist")
	if err != nil {
		t.Fatal(err)
	}
	sc, response, err := storageStepCommandFromRawResponse("DeviceInformation", pBytes)
	if err != nil {
		t.Fatal(err)
	}
	if sc == nil {
		t.Fatal("nil StepCommand")
	}
	if response == nil {
		t.Fatal("nil response")
	}
	devInfo, ok := response.(*mdmcommands.DeviceInformationResponse)
	if !ok {
		t.Fatal("incorrect command type")
	}
	if have, want := sc.CommandUUID, "DevInfo001"; have != want {
		t.Fatalf("have=%v, want=%v", have, want)
	}
	var udid string
	if devInfo.UDID != nil {
		udid = *devInfo.UDID
	}
	if have, want := udid, "UDID001"; have != want {
		t.Fatalf("have=%v, want=%v", have, want)
	}
}

// TestConvertNilStorageCommandRawFromWorkflowCommand tests for a
// regression of a nil check/fix in the mdmcommands module.
func TestConvertNilStorageCommandRawFromWorkflowCommand(t *testing.T) {
	var c *mdmcommands.InstallProfileCommand
	_, err := storageCommandRawFromWorkflowCommand(c)
	if err == nil {
		t.Fatal("want error, have nil")
	}
}

func TestConvertStorageCommandRawFromWorkflowCommand(t *testing.T) {
	_, err := storageCommandRawFromWorkflowCommand(&struct{}{})
	if err == nil {
		t.Fatal("want error, have nil")
	}

	ider := uuid.NewStaticIDs("ABCUUID")
	cmd := mdmcommands.NewSecurityInfoCommand(ider.ID())
	sc, err := storageCommandRawFromWorkflowCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := sc.RequestType, cmd.Command.RequestType; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}
	if have, want := sc.CommandUUID, cmd.CommandUUID; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}
	if have, want := cmd.CommandUUID, "ABCUUID"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}
	secInfoGenPlist, err := os.ReadFile("testdata/secinfo.gen.plist")
	if err != nil {
		t.Fatal(err)
	}
	if have, want := bytes.TrimSpace(sc.Command), bytes.TrimSpace(secInfoGenPlist); !bytes.Equal(have, want) {
		t.Errorf("have: %v, want: %v", string(have), string(want))
	}
}
