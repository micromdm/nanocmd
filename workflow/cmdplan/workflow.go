// Package cmdplan implements a NanoCMD Workflow for sending pre-configured commands to enrollments.
package cmdplan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/mdm"
	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
	profstorage "github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

const WorkflowName = "io.micromdm.wf.cmdplan.v1"

type Workflow struct {
	enq       workflow.StepEnqueuer
	ider      uuid.IDer
	logger    log.Logger
	store     storage.ReadStorage
	profStore profstorage.ReadStorage
}

type Option func(*Workflow)

func WithLogger(logger log.Logger) Option {
	return func(w *Workflow) {
		w.logger = logger
	}
}

func New(enq workflow.StepEnqueuer, store storage.ReadStorage, profStorage profstorage.ReadStorage, opts ...Option) (*Workflow, error) {
	w := &Workflow{
		enq:       enq,
		ider:      uuid.NewUUID(),
		logger:    log.NopLogger,
		store:     store,
		profStore: profStorage,
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
	return new(workflow.StringContext)
}

// TODO: create a map of command UUID to more useful types for logging
func (w *Workflow) commandsFromCMDPlan(ctx context.Context, cmdPlan *storage.CMDPlan, name string, e *workflow.Event) ([]interface{}, error) {
	// bail if invalid
	if cmdPlan == nil {
		return nil, errors.New("invalid cmdplan")
	}

	var commands []interface{}

	if len(cmdPlan.ProfileNames) > 0 {
		// get our raw profiles
		rawProfiles, err := w.profStore.RetrieveRawProfiles(ctx, cmdPlan.ProfileNames)
		if err != nil {
			return nil, fmt.Errorf("retrieving profiles: %w", err)
		}

		// build the profile MDM commands
		for _, name := range cmdPlan.ProfileNames {
			rawProfile, ok := rawProfiles[name]
			if !ok {
				return commands, fmt.Errorf("raw profile not found: %s", name)
			}

			c := mdmcommands.NewInstallProfileCommand(w.ider.ID())
			c.Command.Payload = rawProfile
			commands = append(commands, c)
		}
	}

	// build the install application MDM commands
	for _, url := range cmdPlan.ManifestURLs {
		c := mdmcommands.NewInstallApplicationCommand(w.ider.ID())
		mgmtFlag := 1
		c.Command.ManagementFlags = &mgmtFlag
		c.Command.ManifestURL = &url
		commands = append(commands, c)
	}

	// determine if we need to send the device configured command
	// TODO: this may require using a separate step if we can't guarantee
	// ordered queueing of MDM commands in step enqueuings.
	if cmdPlan.DeviceConfigured != nil && *cmdPlan.DeviceConfigured {
		devConfErrMsg := ""
		if e == nil {
			devConfErrMsg = "empty event"
		} else if e.EventFlag != workflow.EventEnrollment {
			devConfErrMsg = fmt.Sprintf("event type mismatch: %s", e.EventFlag)
		} else if tu, ok := e.EventData.(*mdm.TokenUpdate); !ok {
			devConfErrMsg = "event data type mismatch"
		} else if !tu.AwaitingConfiguration {
			devConfErrMsg = "not awaiting configuration"
		}
		if devConfErrMsg == "" {
			commands = append(commands, mdmcommands.NewDeviceConfiguredCommand(w.ider.ID()))
		} else {
			ctxlog.Logger(ctx, w.logger).Info(
				logkeys.Message, "device configured",
				"name", name,
				logkeys.Error, devConfErrMsg,
			)
		}
	}

	return commands, nil
}

// expandParams perform shell-like ${var} expansion on s and replaces values from p.
// An optional colon-separated "default" value can be provided as well.
func expandParams(s string, p map[string]string) string {
	return os.Expand(s, func(v string) string {
		vs := strings.SplitN(v, ":", 2)
		var r string
		var ok bool
		if p != nil {
			r, ok = p[vs[0]]
		}
		if !ok && len(vs) > 1 {
			return vs[1]
		}
		return r
	})
}

func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	// convert context into a string
	ctxVal, ok := step.Context.(*workflow.StringContext)
	if !ok {
		return workflow.ErrIncorrectContextType
	}

	if *ctxVal == "" {
		return errors.New("missing cmdplan name")
	}

	// expand parameter values
	name := expandParams(string(*ctxVal), step.Params)

	// fetch the cmdplan
	cmdplan, err := w.store.RetrieveCMDPlan(ctx, name)
	if err != nil {
		return fmt.Errorf("retrieving cmdplan: %w", err)
	}

	// gather commands from cmdplan
	commands, err := w.commandsFromCMDPlan(ctx, cmdplan, name, step.Event)
	if err != nil {
		return fmt.Errorf("creating commands from cmdplan: %w", err)
	}
	if len(commands) < 1 {
		return errors.New("no commands to queue")
	}

	// assemble our StepEnqueuing
	se := step.NewStepEnqueueing()
	se.Commands = commands // assign all the commands to the step

	// enqueue our step!
	return w.enq.EnqueueStep(ctx, w, se)
}

func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	// TODO: implement a logger and iter over the commands to see if there's any errors
	// TODO2: implement a map struct so we can log even better errors (i.e. which specific profile, etc.)
	logger := ctxlog.Logger(ctx, w.logger).With(logkeys.InstanceID, stepResult.InstanceID)
	statuses := make(map[string]int)
	for _, resp := range stepResult.CommandResults {
		genResper, ok := resp.(mdmcommands.GenericResponser)
		if !ok {
			continue
		}
		genResp := genResper.GetGenericResponse()
		statuses[genResp.Status] += 1
		// TODO: log the association from command UUID to command details in context struct
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
}

func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

func (w *Workflow) Event(ctx context.Context, e *workflow.Event, id string, mdmCtx *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
