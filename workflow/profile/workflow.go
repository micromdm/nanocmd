// Package profile implements a NanoCMD Workflow for "statefully" installing and removing profiles.
package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/ctxlog"
	"github.com/micromdm/nanocmd/log/logkeys"
	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

const WorkflowName = "io.micromdm.wf.profile.v1"

// Workflow "statefully" installs and removes profiles.
type Workflow struct {
	enq    workflow.StepEnqueuer
	store  storage.ReadStorage
	ider   uuid.IDer
	logger log.Logger
}

type Option func(*Workflow)

func WithLogger(logger log.Logger) Option {
	return func(w *Workflow) {
		w.logger = logger
	}
}

func New(enq workflow.StepEnqueuer, store storage.ReadStorage, opts ...Option) (*Workflow, error) {
	w := &Workflow{
		enq:    enq,
		store:  store,
		ider:   uuid.NewUUID(),
		logger: log.NopLogger,
	}
	for _, opt := range opts {
		opt(w)
	}
	w.logger = w.logger.With(logkeys.WorkflowName, w.Name())
	return w, nil
}

func (w *Workflow) Name() string {
	return WorkflowName
}

func (w *Workflow) Config() *workflow.Config {
	return nil
}

func (w *Workflow) NewContextValue(name string) workflow.ContextMarshaler {
	switch name {
	case "", "list":
		// for the start and list steps, use the comma context type
		return new(CommaStringSliceContext)
	default:
		return nil
	}
}

func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	// make sure our context is of the correct type
	manageList, ok := step.Context.(*CommaStringSliceContext)
	if !ok {
		return workflow.ErrIncorrectContextType
	}

	// sanity check
	if len(*manageList) < 1 {
		return errors.New("no managed profiles supplied in context")
	}

	// parse and get our profiles list.
	// TODO: we could cache these results in the context and re-use them later.
	all, _ := splitInstallRemove(*manageList)

	// retrive the infos to make sure they exist. this avoids starting
	// the workflow if we supplied an invalid set of profile names.
	if _, err := w.store.RetrieveProfileInfos(ctx, all); err != nil {
		return fmt.Errorf("retrieving profile info: %w", err)
	}

	ctxlog.Logger(ctx, w.logger).Debug(
		logkeys.InstanceID, step.InstanceID,
		logkeys.FirstEnrollmentID, step.IDs[0],
		logkeys.GenericCount, len(step.IDs),
		logkeys.Message, "enqueuing step",
		"profile_count", len(all),
		"profile_first", all[0],
	)

	// build a ProfileList command
	cmd := mdmcommands.NewProfileListCommand(w.ider.ID())
	managedOnly := true
	cmd.Command.ManagedOnly = &managedOnly

	// assemble our StepEnqueuing
	se := step.NewStepEnqueueing()
	se.Commands = []interface{}{cmd}
	se.Context = manageList // re-use our passed-in context (i.e. the list of profiles to manage)
	se.Name = "list"        // will get handed back to us in StepCompleted

	// enqueue our step!
	return w.enq.EnqueueStep(ctx, w, se)
}

const (
	manageInstallReq int = iota + 1 // install "requested"
	manageRemoveReq                 // removal "requested"
	manageToInstall                 // confirmed to install
	manageToRemove                  // confirmed to remove
)

// split the context profiles into a slice of all of them and
// a map-to-management style (i.e. install or remove)
func splitInstallRemove(s CommaStringSliceContext) (all []string, ret map[string]int) {
	ret = make(map[string]int)
	for _, name := range s {
		if strings.HasPrefix(name, "-") {
			// if one of the entries is prefixed with a "-" dash (minus)
			// then it is for removal.
			ret[name[1:]] = manageRemoveReq
			all = append(all, name[1:])
		} else {
			ret[name] = manageToInstall
			all = append(all, name)
		}
	}
	return
}

func (w *Workflow) listStepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	profListResp, ok := stepResult.CommandResults[0].(*mdmcommands.ProfileListResponse)
	if !ok {
		return fmt.Errorf("%w: not a profile list", workflow.ErrIncorrectCommandType)
	}
	if err := profListResp.Validate(); err != nil {
		return fmt.Errorf("validating profile list: %w", err)
	}

	// make sure our context is of the correct type
	manageList, ok := stepResult.Context.(*CommaStringSliceContext)
	if !ok {
		return workflow.ErrIncorrectContextType
	}

	all, manageMap := splitInstallRemove(*manageList)

	// retrieve the list of profiles provided to the workflow when started
	allProfsToManage, err := w.store.RetrieveProfileInfos(ctx, all)
	if err != nil {
		return fmt.Errorf("retrieving profile info: %w", err)
	}

	// find out which profiles we need to install of the requested
loop:
	for name, info := range allProfsToManage {
		manageStyle := manageMap[name]
		for _, profListItem := range profListResp.ProfileList {
			if profListItem.PayloadIdentifier == info.Identifier {
				if manageStyle == manageRemoveReq {
					// found profile on system but we need to remove it
					manageMap[name] = manageToRemove
					continue loop
				}
				if manageStyle == manageToInstall && profListItem.PayloadUUID == info.UUID {
					// matching UUID and identifier: don't install
					manageMap[name] = manageInstallReq
					continue loop
				}
			}
		}
	}

	// convert the map to a slice of names for our raw profile retrieval
	var profToInstSlice []string
	for name, manageStyle := range manageMap {
		if manageStyle == manageToInstall {
			profToInstSlice = append(profToInstSlice, name)
		}
	}

	// get our raw profiles (only if we need to)
	var profToInstRaw map[string][]byte
	if len(profToInstSlice) > 0 {
		// retrieve the raw profiles from the store
		profToInstRaw, err = w.store.RetrieveRawProfiles(ctx, profToInstSlice)
		if err != nil {
			return fmt.Errorf("retrieving raw profiles: %w", err)
		}
	}

	// create our step enqueueing
	se := stepResult.NewStepEnqueueing()
	se.Name = "install"

	// assemble our collection of InstallProfile and RemoveProfile MDM
	// commands and append them to the command list
	for name, manageStyle := range manageMap {
		switch manageStyle {
		case manageToInstall:
			cmd := mdmcommands.NewInstallProfileCommand(w.ider.ID())
			cmd.Command.Payload = profToInstRaw[name]
			se.Commands = append(se.Commands, cmd)
		case manageToRemove:
			cmd := mdmcommands.NewRemoveProfileCommand(w.ider.ID())
			cmd.Command.Identifier = allProfsToManage[name].Identifier
			se.Commands = append(se.Commands, cmd)
		}
	}

	if len(se.Commands) > 0 {
		// enqueue our step!
		return w.enq.EnqueueStep(ctx, w, se)
	}
	ctxlog.Logger(ctx, w.logger).Debug(
		logkeys.InstanceID, stepResult.InstanceID,
		logkeys.StepName, stepResult.Name,
		logkeys.EnrollmentID, stepResult.ID,
		logkeys.Message, "no profiles to install or remove after profile list",
	)
	return nil
}

func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	switch stepResult.Name {
	case "list":
		return w.listStepCompleted(ctx, stepResult)
	case "install":
		logger := ctxlog.Logger(ctx, w.logger).With(
			logkeys.InstanceID, stepResult.InstanceID,
			logkeys.StepName, stepResult.Name,
		)
		statuses := make(map[string]int)
		for _, resp := range stepResult.CommandResults {
			genResper, ok := resp.(mdmcommands.GenericResponser)
			if !ok {
				continue
			}
			genResp := genResper.GetGenericResponse()
			statuses[genResp.Status] += 1
			// TODO: log the association from command UUID to profile name
			if err := genResp.Validate(); err != nil {
				logger.Info(
					logkeys.Message, "validate MDM response",
					logkeys.CommandUUID, genResp.CommandUUID,
					logkeys.Error, err,
				)
			}
		}
		logs := []interface{}{logkeys.Message, "workflow complete"}
		for k, v := range statuses {
			logs = append(logs, "count_"+strings.ToLower(k), v)
		}
		logger.Debug(logs...)
		return nil
	default:
		return fmt.Errorf("%w: %s", workflow.ErrUnknownStepName, stepResult.Name)
	}
}

func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

func (w *Workflow) Event(_ context.Context, _ *workflow.Event, _ string, _ *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
