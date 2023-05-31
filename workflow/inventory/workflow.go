// Package inventory implements a NanoCMD Workflow that updates an inventory system.
package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

const WorkflowName = "io.micromdm.wf.inventory.v1"

var WorkflowConfig = &workflow.Config{
	// we want all SecurityInfo commands, regardless of whether this workflow sent them.
	AllCommandResponseRequestTypes: []string{"SecurityInfo"},
}

// Workflow is a workflow that updates inventory storage.
type Workflow struct {
	enq    workflow.StepEnqueuer
	ider   uuid.IDer
	store  storage.Storage
	logger log.Logger
}

type Option func(*Workflow)

func New(enq workflow.StepEnqueuer, store storage.Storage, opts ...Option) (*Workflow, error) {
	return &Workflow{
		enq:    enq,
		ider:   uuid.NewUUID(),
		store:  store,
		logger: log.NopLogger,
	}, nil
}

func (w *Workflow) Name() string {
	return WorkflowName
}

func (w *Workflow) Config() *workflow.Config {
	return WorkflowConfig
}

func (w *Workflow) NewContextValue(_ string) workflow.ContextMarshaler {
	return nil
}

func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	// build a DeviceInformation command
	cmd := mdmcommands.NewDeviceInformationCommand(w.ider.ID())
	cmd.Command.Queries = []string{
		"Model",
		"SerialNumber",
		"Model",
		"ModelName",
		"DeviceName",
		"BuildVersion",
		"OSVersion",
		"EthernetMAC",
		"IsAppleSilicon",
		"HasBattery",
		"IsMultiUser",
		"SupportsLOMDevice",
	}

	// build a SecurityInfo command
	cmd2 := mdmcommands.NewSecurityInfoCommand(w.ider.ID())

	// assemble our StepEnqueuing
	se := step.NewStepEnqueueing()
	se.Commands = []interface{}{cmd, cmd2}

	// enqueue our step!
	return w.enq.EnqueueStep(ctx, w, se)
}

func storeIfPresent[T any](v storage.Values, k string, p *T) {
	if p == nil {
		return
	}
	v[k] = *p
}

func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	if len(stepResult.CommandResults) != 2 {
		return workflow.ErrStepResultCommandLenMismatch
	}

	for _, response := range stepResult.CommandResults {
		switch r := response.(type) {
		case *mdmcommands.DeviceInformationResponse:
			// we did send a SecurityInfo command, too, but we only care about our
			// DeviceInfo in the step completeion. we process the SecurityInfo
			// command as an event.

			if err := r.Validate(); err != nil {
				return fmt.Errorf("device info response: %w", err)
			}

			v := make(storage.Values)
			qr := r.QueryResponses
			storeIfPresent(v, storage.KeySerialNumber, qr.SerialNumber)
			storeIfPresent(v, storage.KeyModel, qr.Model)
			storeIfPresent(v, storage.KeyModelName, qr.ModelName)
			storeIfPresent(v, storage.KeyDeviceName, qr.DeviceName)
			storeIfPresent(v, storage.KeyBuildVersion, qr.BuildVersion)
			storeIfPresent(v, storage.KeyOSVersion, qr.OSVersion)
			storeIfPresent(v, storage.KeyEthernetMAC, qr.EthernetMAC)
			storeIfPresent(v, storage.KeySupervised, qr.IsSupervised)
			storeIfPresent(v, storage.KeyAppleSilicon, qr.IsAppleSilicon)
			storeIfPresent(v, storage.KeyHasBattery, qr.HasBattery)
			storeIfPresent(v, storage.KeySupportsLOM, qr.SupportsLOMDevice)
			storeIfPresent(v, storage.KeyIsMultiUser, qr.IsMultiUser)
			if len(v) > 0 {
				v[storage.KeyLastSource] = mdmcommands.DeviceInformationRequestType
				v[storage.KeyModified] = time.Now()
				return w.store.StoreInventoryValues(ctx, stepResult.ID, v)
			}
		}
	}
	return nil
}

func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

func (w *Workflow) Event(ctx context.Context, e *workflow.Event, id string, mdmCtx *workflow.MDMContext) error {
	switch evData := e.EventData.(type) {
	case *mdmcommands.SecurityInfoResponse:
		if err := evData.Validate(); err != nil {
			return fmt.Errorf("security info response: %w", err)
		}

		v := make(storage.Values)
		si := evData.SecurityInfo
		storeIfPresent(v, storage.KeySIPEnabled, si.SystemIntegrityProtectionEnabled)
		storeIfPresent(v, storage.KeyFDEEnabled, si.FDEEnabled)
		if len(v) > 0 {
			v[storage.KeyLastSource] = mdmcommands.SecurityInfoRequestType
			v[storage.KeyModified] = time.Now()
			return w.store.StoreInventoryValues(ctx, id, v)
		}
	default:
		return fmt.Errorf("unknown event data type for event: %s", e.EventFlag)
	}
	return nil
}
