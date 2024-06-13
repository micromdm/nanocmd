// Package engine implements the NanoCMD workflow engine.
package engine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/mdm"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

var (
	ErrNoSuchWorkflow = errors.New("no such workflow")
	ErrNoIDs          = errors.New("no IDs")
)

func NewErrNoSuchWorkflow(name string) error {
	return fmt.Errorf("%w: %s", ErrNoSuchWorkflow, name)
}

// RawEnqueuer sends raw Plist commands to enrollment IDs.
type RawEnqueuer interface {
	Enqueue(ctx context.Context, ids []string, rawCmd []byte) error
}

// PushEnqueuer sends raw commands and APNs pushes to enrollment IDs.
type PushEnqueuer interface {
	RawEnqueuer
	Push(ctx context.Context, ids []string) error
}

// Enqueuer sends raw Plist commands to enrollment IDs and relays multi-command capability.
type Enqueuer interface {
	RawEnqueuer
	SupportsMultiCommands() bool
}

// DefaultTimeout is the default workflow step timeout.
// A workflow's configured timeout will override this default and a
// step's enqueued timeout will override that.
const DefaultTimeout = time.Hour * 24 * 3

// Engine coordinates workflows with MDM servers.
type Engine struct {
	workflowsMu sync.RWMutex
	workflows   map[string]workflow.Workflow
	allResps    map[string][]string // map of MDM command Request Types to slice of workflow names

	storage      storage.Storage
	enqueuer     Enqueuer
	eventStorage storage.ReadEventSubscriptionStorage

	logger log.Logger
	ider   uuid.IDer

	defaultTimeout time.Duration
}

// Options configure the engine.
type Option func(*Engine)

// WithLogger sets the engine logger.
func WithLogger(logger log.Logger) Option {
	return func(e *Engine) {
		e.logger = logger
	}
}

// WithDefaultTimeout configures the engine for a default workflow step timeout.
func WithDefaultTimeout(timeout time.Duration) Option {
	return func(e *Engine) {
		e.defaultTimeout = timeout
	}
}

// WithEventStorage turns on the event dispatch and configures the storage.
func WithEventStorage(evStorage storage.ReadEventSubscriptionStorage) Option {
	return func(e *Engine) {
		e.eventStorage = evStorage
	}
}

// New creates a new NanoCMD engine with default configurations.
func New(storage storage.Storage, enqueuer Enqueuer, opts ...Option) *Engine {
	engine := &Engine{
		workflows:      make(map[string]workflow.Workflow),
		allResps:       make(map[string][]string),
		storage:        storage,
		enqueuer:       enqueuer,
		logger:         log.NopLogger,
		ider:           uuid.NewUUID(),
		defaultTimeout: DefaultTimeout,
	}
	for _, opt := range opts {
		opt(engine)
	}
	return engine
}

// diff returns the difference between a and b
// That is: the items not in both slices.
func diff(a, b []string) (r []string) {
	seen := make(map[string]int)
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		seen[v]--
	}
	for k, v := range seen {
		if v != 0 {
			r = append(r, k)
		}
	}
	return
}

// StartWorkflow starts a new workflow instance for workflow name.
func (e *Engine) StartWorkflow(ctx context.Context, name string, context []byte, ids []string, ev *workflow.Event, mdmCtx *workflow.MDMContext) (string, error) {
	// retrieve our workflow and check validity
	w := e.Workflow(name)
	if w == nil {
		return "", NewErrNoSuchWorkflow(name)
	}

	logger := ctxlog.Logger(ctx, e.logger).With(logkeys.WorkflowName, name)

	if cfg := w.Config(); cfg == nil || cfg.Exclusivity == workflow.Exclusive {
		// check if our ids have any outstanding workflows running
		wRunningIDs, err := e.storage.RetrieveOutstandingWorkflowStatus(ctx, name, ids)
		if err != nil {
			return "", fmt.Errorf("retrieving outstanding status: %w", err)
		}
		if len(wRunningIDs) > 0 {
			ct := len(ids)
			ids = diff(ids, wRunningIDs) // replace our ids with the set of NON-outstanding ids
			if len(ids) < 1 {
				// if all IDs are already running, then return an error
				return "", fmt.Errorf("workflow already started on %d (of %d) ids", len(wRunningIDs), ct)
			} else {
				logger.Debug(
					logkeys.Message, fmt.Sprintf("workflow already started on %d (of %d) ids", len(wRunningIDs), ct),
					logkeys.GenericCount, len(ids),
				)
			}
		}
	}

	var startIDs [][]string
	if e.enqueuer.SupportsMultiCommands() {
		startIDs = [][]string{ids}
	} else {
		// if we do not support multi-targeted commands then we need
		// to break apart the initial multi-target IDs. the primary
		// reason for this is workflows should be agnostic about
		// whether they can target multi-ids or not. in this way a
		// workflow can simply generate a unique UUID for each of its
		// commands regardless of the underlying support.
		for _, id := range ids {
			startIDs = append(startIDs, []string{id})
		}
	}

	// create a new instance ID
	instanceID := e.ider.ID()

	var retErr error // accumulate and return the last start error
	for _, startID := range startIDs {
		// check that we have enrollment IDs to start in our IDs
		if len(startID) < 1 {
			logger.Info(logkeys.Error, ErrNoIDs)
			continue
		}

		// create a workflow start step
		ss, err := workflowStepStartFromEngine(instanceID, w, context, ids, ev, mdmCtx)
		if err != nil {
			return instanceID, fmt.Errorf("converting step start: %w", err)
		}
		if err = w.Start(ctx, ss); err != nil {
			return instanceID, fmt.Errorf("staring workflow: %w", err)
		}
		if err = e.storage.RecordWorkflowStarted(ctx, startID, name, time.Now()); err != nil {
			return instanceID, fmt.Errorf("recording workflow status: %w", err)
		}
		logger.Debug(
			logkeys.InstanceID, instanceID,
			logkeys.Message, "starting workflow",
			logkeys.FirstEnrollmentID, startID[0],
			logkeys.GenericCount, len(startID),
		)
	}

	return instanceID, retErr
}

// stepDefaultTimeout returns either the engine or workflow default step timeout.
func (e *Engine) stepDefaultTimeout(workflowName string) (defaultTimeout time.Time) {
	if e.defaultTimeout > 0 {
		defaultTimeout = time.Now().Add(e.defaultTimeout)
	}
	w := e.Workflow(workflowName)
	if w == nil {
		return
	}
	cfg := w.Config()
	if cfg == nil {
		return
	}
	if cfg.Timeout > 0 {
		defaultTimeout = time.Now().Add(cfg.Timeout)
	}
	return
}

// EnqueueStep stores the step and enqueues the commands to the MDM server.
func (e *Engine) EnqueueStep(ctx context.Context, n workflow.Namer, se *workflow.StepEnqueueing) error {
	ss, err := storageStepEnqueuingWithConfigFromWorkflowStepEnqueueing(n, e.stepDefaultTimeout(n.Name()), se)
	if err != nil {
		return fmt.Errorf("converting workflow step: %w", err)
	}

	if err = e.storage.StoreStep(ctx, ss, time.Now()); err != nil {
		return fmt.Errorf("storing step: %w", err)
	}

	if ss.NotUntil.IsZero() {
		// if we are not delaying the steps, then send them now
		for _, cmd := range ss.Commands {
			if err = e.enqueuer.Enqueue(ctx, ss.IDs, cmd.Command); err != nil {
				return fmt.Errorf("enqueueing step: %w", err)
			}
		}
	}

	ctxlog.Logger(ctx, e.logger).Debug(
		logkeys.Message, "enqueued step",
		logkeys.InstanceID, ss.InstanceID,
		logkeys.GenericCount, len(ss.IDs),
		logkeys.FirstEnrollmentID, ss.IDs[0],
		logkeys.WorkflowName, ss.WorkflowName,
		logkeys.StepName, ss.Name,
		"command_count", len(ss.Commands),
	)

	return nil
}

// dispatchAllCommandResponseRequestTypes sends the "AllCommandResponse" to subscribed workflows.
func (e *Engine) dispatchAllCommandResponseRequestTypes(ctx context.Context, reqType string, id string, response interface{}, mdmCtx *workflow.MDMContext) error {
	logger := ctxlog.Logger(ctx, e.logger).With(
		"request_type", reqType,
		logkeys.EnrollmentID, id,
	)
	ev := &workflow.Event{
		EventFlag: workflow.EventAllCommandResponse,
		EventData: response,
	}
	var wg sync.WaitGroup
	for _, w := range e.allRespWorkflows(reqType) {
		wg.Add(1)
		go func(w workflow.Workflow) {
			defer wg.Done()
			err := w.Event(ctx, ev, id, mdmCtx)
			if err != nil {
				logger.Info(
					logkeys.Message, "workflow all command response",
					logkeys.WorkflowName, w.Name(),
					logkeys.Error, err,
				)
			}
		}(w)
	}
	wg.Wait()
	return nil
}

func logAndError(err error, logger log.Logger, msg string) error {
	logger.Info(
		logkeys.Message, msg,
		logkeys.Error, err,
	)
	return fmt.Errorf("%s: %w", msg, err)
}

// MDMIdleEvent is called when an MDM Report Results has an "Idle" status.
// MDMIdleEvent will dispatch workflow "Idle" events (for workflows that are
// configured for it) and will also start workflows for the "IdleNotStartedSince"
// event subscription type.
// Note: any other event subscription type starting workflows is not supported.
func (e *Engine) MDMIdleEvent(ctx context.Context, id string, raw []byte, mdmContext *workflow.MDMContext, eventAt time.Time) error {
	logger := ctxlog.Logger(ctx, e.logger).With(logkeys.EnrollmentID, id)

	// dispatch the events to (only) the workflow events
	event := &workflow.Event{EventFlag: workflow.EventIdle}
	if err := e.dispatchEvents(ctx, id, event, mdmContext, false, true); err != nil {
		logger.Info(
			logkeys.Message, "idle event: dispatch workflow events",
			logkeys.Event, event.EventFlag,
			logkeys.Error, err,
		)
	}

	if e.eventStorage == nil {
		return nil
	}

	subs, err := e.eventStorage.RetrieveEventSubscriptionsByEvent(ctx, workflow.EventIdleNotStartedSince)
	if err != nil {
		logger.Info(
			logkeys.Message, "retrieving event subscriptions",
			logkeys.Event, workflow.EventIdleNotStartedSince,
			logkeys.Error, err,
		)
	}

	if len(subs) < 1 {
		return nil
	}

	var wg sync.WaitGroup
	event = &workflow.Event{EventFlag: workflow.EventIdleNotStartedSince}
	for _, sub := range subs {
		wg.Add(1)
		go func(es *storage.EventSubscription) {
			defer wg.Done()

			if es == nil {
				return
			}

			subLogger := logger.With(
				logkeys.Event, workflow.EventIdleNotStartedSince,
				logkeys.WorkflowName, es.Workflow,
			)

			// get the last time this workflow started for this
			// enrollment ID for the workflow was subscribed to.
			started, err := e.storage.RetrieveWorkflowStarted(ctx, id, es.Workflow)
			if err != nil {
				subLogger.Info(
					logkeys.Message, "retrieving workflow status",
					logkeys.Error, err,
				)
				return
			}

			// make sure we have a valid event context (time between runs)
			if es.EventContext == "" {
				subLogger.Info(
					logkeys.Error, "event context is empty",
				)
				return
			}
			sinceSeconds, err := strconv.Atoi(es.EventContext)
			if err != nil {
				subLogger.Info(
					logkeys.Message, "converting event context to integer",
					logkeys.Error, err,
				)
				return
			} else if sinceSeconds < 1 {
				subLogger.Info(
					logkeys.Error, "event context less than 1 second",
				)
				return
			}

			// check if we've run this workflow "recently"
			if !eventAt.After(started.Add(time.Second * time.Duration(sinceSeconds))) {
				// TODO: hide behind an "extra" debug flag?
				// subLogger.Debug(logkeys.Message, "workflow not due yet")
				return
			}

			if instanceID, err := e.StartWorkflow(ctx, es.Workflow, []byte(es.Context), []string{id}, event, mdmContext); err != nil {
				subLogger.Info(
					logkeys.Message, "start workflow",
					logkeys.InstanceID, instanceID,
					logkeys.Error, err,
				)
			} else {
				subLogger.Debug(
					logkeys.Message, "started workflow",
					logkeys.InstanceID, instanceID,
				)
			}
		}(sub)
	}
	wg.Wait()
	return nil
}

// MDMCommandResponseEvent receives MDM command responses.
func (e *Engine) MDMCommandResponseEvent(ctx context.Context, id string, uuid string, raw []byte, mdmContext *workflow.MDMContext) error {
	logger := ctxlog.Logger(ctx, e.logger).With(
		logkeys.CommandUUID, uuid,
		logkeys.EnrollmentID, id,
	)

	// see if this is a engine-"tracked" MDM command and get its metadata if so
	reqType, ok, err := e.storage.RetrieveCommandRequestType(ctx, id, uuid)
	if err != nil {
		return logAndError(err, logger, "retreive command request type")
	}
	logger = logger.With("engine_command", ok)

	if !ok {
		// we didn't find this command UUID
		// probably did not originate with the engine
		logger.Debug()
		return nil
	}

	logger = logger.With(logkeys.RequestType, reqType)

	// convert raw response to a storage raw response
	sc, response, err := storageStepCommandFromRawResponse(reqType, raw)
	if err != nil {
		return logAndError(err, logger, "convert response")
	}
	logger = logger.With("command_completed", sc.Completed)

	var wg sync.WaitGroup
	defer wg.Wait() // we have a context so make sure we block
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := e.dispatchAllCommandResponseRequestTypes(ctx, reqType, id, response, mdmContext); err != nil {
			logger.Info(
				logkeys.Message, "dispatching all command response types",
				logkeys.Error, err,
			)
		}
	}()

	// store our command response and get the completed storage step result
	ssr, err := e.storage.StoreCommandResponseAndRetrieveCompletedStep(ctx, id, sc)
	if err != nil {
		return logAndError(err, logger, "store command retrieve completed")
	}
	logger = logger.With("step_completed", ssr != nil)

	if ssr == nil {
		logger.Debug()
		// return if there was no completed step; nothing more to do.
		return nil
	}

	logger = logger.With(
		logkeys.WorkflowName, ssr.WorkflowName,
		logkeys.InstanceID, ssr.InstanceID,
	)

	w := e.Workflow(ssr.WorkflowName)
	if w == nil {
		return logAndError(NewErrNoSuchWorkflow(ssr.WorkflowName), logger, "retrieving workflow")
	}

	// create a workflow step result for handing off to a workflow
	stepResult, err := workflowStepResultFromStorageStepResult(ssr, w, false, uuid, response)
	if err != nil {
		return logAndError(err, logger, "converting storage step")
	}

	if mdmContext != nil {
		stepResult.MDMContext = *mdmContext
	}

	// let our workflow know that we have completed the step
	if err = w.StepCompleted(ctx, stepResult); err != nil {
		return logAndError(err, logger, "completing workflow step")
	}
	logger.Debug(logkeys.Message, "completed workflow step")
	return nil
}

// dispatchEvents dispatches MDM check-in events.
// this includes event subscriptions (user configured) and workflow
// configs. The bool subEV as true indicates to run subscription event
// workflows and wfEV indicates to run workflow-configured events.
func (e *Engine) dispatchEvents(ctx context.Context, id string, ev *workflow.Event, mdmCtx *workflow.MDMContext, subEV, wfEV bool) error {
	logger := ctxlog.Logger(ctx, e.logger).With(
		logkeys.Event, ev.EventFlag,
		logkeys.EnrollmentID, id,
	)
	var wg sync.WaitGroup
	if subEV && e.eventStorage != nil {
		subs, err := e.eventStorage.RetrieveEventSubscriptionsByEvent(ctx, ev.EventFlag)
		if err != nil {
			logger.Info(
				logkeys.Message, "retrieving event subscriptions",
				logkeys.Error, err,
			)
		} else {
			for _, sub := range subs {
				wg.Add(1)
				go func(es *storage.EventSubscription) {
					defer wg.Done()
					if instanceID, err := e.StartWorkflow(ctx, es.Workflow, []byte(es.Context), []string{id}, ev, mdmCtx); err != nil {
						logger.Info(
							logkeys.Message, "start workflow",
							logkeys.WorkflowName, es.Workflow,
							logkeys.InstanceID, instanceID,
							logkeys.Error, err,
						)
					} else {
						logger.Debug(
							logkeys.Message, "started workflow",
							logkeys.WorkflowName, es.Workflow,
							logkeys.InstanceID, instanceID,
						)
					}
				}(sub)
			}
		}
	}
	if wfEV {
		for _, w := range e.eventWorkflows(ev.EventFlag) {
			wg.Add(1)
			go func(w workflow.Workflow) {
				defer wg.Done()
				if err := w.Event(ctx, ev, id, mdmCtx); err != nil {
					logger.Info(
						logkeys.Message, "workflow event",
						logkeys.WorkflowName, w.Name(),
						logkeys.Error, err,
					)
				} else {
					logger.Debug(
						logkeys.Message, "workflow event",
						logkeys.WorkflowName, w.Name(),
					)
				}
			}(w)
		}
	}
	wg.Wait()
	return nil
}

// MDMCheckinEvent receives MDM checkin messages.
func (e *Engine) MDMCheckinEvent(ctx context.Context, id string, checkin interface{}, mdmContext *workflow.MDMContext) error {
	logger := ctxlog.Logger(ctx, e.logger).With(logkeys.EnrollmentID, id)
	cancelSteps := false
	var events []*workflow.Event
	switch v := checkin.(type) {
	case *mdm.Authenticate:
		cancelSteps = true
		events = []*workflow.Event{{
			EventFlag: workflow.EventAuthenticate,
			EventData: v,
		}}
	case *mdm.TokenUpdate:
		events = []*workflow.Event{{
			EventFlag: workflow.EventTokenUpdate,
			EventData: v,
		}, {
			// from a pure token update we can't tell if an enrollment
			// happened. so we default to sending that event, too.
			// even if this is a supplementary intra-enrollment token
			// update.
			EventFlag: workflow.EventEnrollment,
			EventData: v,
		}}
	case *mdm.TokenUpdateEnrolling:
		events = []*workflow.Event{{
			EventFlag: workflow.EventTokenUpdate,
			EventData: v.TokenUpdate,
		}}
		if v.Enrolling {
			// with this type we *can* tell if we're enrolling or not.
			// so only dispatch that event if, truly, we're enrolling.
			events = append(events, &workflow.Event{
				EventFlag: workflow.EventEnrollment,
				EventData: v.TokenUpdate,
			})
		}
	case *mdm.CheckOut:
		cancelSteps = true
		events = []*workflow.Event{{
			EventFlag: workflow.EventCheckOut,
			EventData: v,
		}}
	}
	if cancelSteps {
		// we cancel all steps for an enrollment upon re-enrollment
		// or checkout. this will allow us to enqueue workflows again.
		// otherwise any outstanding workflow instances would block
		// new ones being executed due to exclusivity.
		if err := e.storage.CancelSteps(ctx, id, ""); err != nil {
			return logAndError(err, logger, "checkin event: cancel steps")
		}
		// also clear out any workflow status for an id
		if err := e.storage.ClearWorkflowStatus(ctx, id); err != nil {
			return logAndError(err, logger, "checkin event: clearing workflow status")
		}
	}
	for _, event := range events {
		if err := e.dispatchEvents(ctx, id, event, mdmContext, true, true); err != nil {
			logger.Info(
				logkeys.Message, "checkin event: dispatch events",
				logkeys.Event, event.EventFlag,
				logkeys.Error, err,
			)
		}
	}
	return nil
}
