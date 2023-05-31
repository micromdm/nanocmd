package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/logkeys"
	"github.com/micromdm/nanocmd/workflow"
)

const DefaultDuration = time.Minute * 5
const DefaultRePushDuration = time.Hour * 24

type WorkflowFinder interface {
	Workflow(name string) workflow.Workflow
}

// Worker polls storage backends for timed events on an interval.
// Examples include step timeouts, delayed steps (NotUntil), and
// re-pushes.
type Worker struct {
	wff      WorkflowFinder
	storage  storage.WorkerStorage
	enqueuer PushEnqueuer
	logger   log.Logger

	// duration is the interval at which the worker will wake up to
	// continue polling the storage backend for data to take action on.
	duration time.Duration

	// repushDuration is how long MDM commands should go without any
	// response seen before we send an APNs to the enrollment ID.
	repushDuration time.Duration
}

type WorkerOption func(w *Worker)

func WithWorkerLogger(logger log.Logger) WorkerOption {
	return func(w *Worker) {
		w.logger = logger
	}
}

// WithWorkerDuration configures the polling interval for the worker.
func WithWorkerDuration(d time.Duration) WorkerOption {
	return func(w *Worker) {
		w.duration = d
	}
}

// WithWorkerRePushDuration configures when enrollments should be sent APNs pushes.
// This is the duration an enrollment ID has not received a response for
// an MDM command.
func WithWorkerRePushDuration(d time.Duration) WorkerOption {
	return func(w *Worker) {
		w.repushDuration = d
	}
}

func NewWorker(wff WorkflowFinder, storage storage.WorkerStorage, enqueuer PushEnqueuer, opts ...WorkerOption) *Worker {
	w := &Worker{
		wff:      wff,
		storage:  storage,
		enqueuer: enqueuer,
		logger:   log.NopLogger,
		duration: DefaultDuration,

		repushDuration: DefaultRePushDuration,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// RunOnce runs the processes of the worker and logs errors.
func (w *Worker) RunOnce(ctx context.Context) error {
	err := w.processEnqueuings(ctx)
	if err != nil {
		return logAndError(err, w.logger, "processing enqueueings")
	}
	if err = w.processTimeouts(ctx); err != nil {
		return logAndError(err, w.logger, "processing timeouts")
	}
	if w.repushDuration > 0 {
		if err = w.processRePushes(ctx); err != nil {
			return logAndError(err, w.logger, "processing repushes")
		}
	}
	return nil
}

// Run starts and runs the worker forever on an interval.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Debug(logkeys.Message, "starting worker", "duration", w.duration)

	ticker := time.NewTicker(w.duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.RunOnce(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) processEnqueuings(ctx context.Context) error {
	steps, err := w.storage.RetrieveStepsToEnqueue(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("retrieving steps to enqueue: %w", err)
	}

	for _, step := range steps {
		stepLogger := w.logger.With(
			logkeys.Message, "enqueueing command",
			logkeys.InstanceID, step.InstanceID,
			logkeys.WorkflowName, step.WorkflowName,
			logkeys.StepName, step.Name,
			logkeys.GenericCount, len(step.IDs),
			logkeys.FirstEnrollmentID, step.IDs[0],
		)
		if len(step.Commands) < 1 {
			stepLogger.Info()
		}

		for _, cmd := range step.Commands {
			logger := stepLogger.With(
				logkeys.CommandUUID, cmd.CommandUUID,
				logkeys.RequestType, cmd.RequestType,
			)
			err := w.enqueuer.Enqueue(ctx, step.IDs, cmd.Command)
			if err != nil {
				logger.Info(logkeys.Error, err)
			} else {
				logger.Debug()
			}
		}
	}
	return nil
}

func (w *Worker) processTimeouts(ctx context.Context) error {
	steps, err := w.storage.RetrieveTimedOutSteps(ctx)
	if err != nil {
		return fmt.Errorf("retrieving timed-out steps: %w", err)
	}

	for _, step := range steps {
		stepLogger := w.logger.With(
			logkeys.Message, "step timeout",
			logkeys.InstanceID, step.InstanceID,
			logkeys.WorkflowName, step.WorkflowName,
			logkeys.StepName, step.Name,
		)
		if len(step.IDs) != 1 {
			// step timeouts need to be per-enrollment ID
			stepLogger.Info(logkeys.Error, "invalid count of step IDs")
			continue
		}
		stepLogger = stepLogger.With(logkeys.EnrollmentID, step.IDs[0])
		w := w.wff.Workflow(step.WorkflowName)
		if w == nil {
			stepLogger.Info(logkeys.Error, NewErrNoSuchWorkflow(step.WorkflowName))
			continue
		}

		// convert the storage step result to a workflow step result
		stepResult, err := workflowStepResultFromStorageStepResult(step, w, true, "", nil)
		if err != nil {
			stepLogger.Info(logkeys.Error, err)
			continue
		}

		// send the timeout notification
		if err = w.StepTimeout(ctx, stepResult); err != nil {
			stepLogger.Info(logkeys.Error, err)
		} else {
			stepLogger.Debug()
		}
	}
	return nil
}

func (w *Worker) processRePushes(ctx context.Context) error {
	ids, err := w.storage.RetrieveAndMarkRePushed(ctx, time.Now().Add(-w.repushDuration), time.Now())
	if err != nil {
		return fmt.Errorf("retrieving repush ids: %w", err)
	}
	if len(ids) < 1 {
		return nil
	}
	logger := w.logger.With(
		logkeys.FirstEnrollmentID, ids[0],
		logkeys.GenericCount, len(ids),
	)
	if err = w.enqueuer.Push(ctx, ids); err != nil {
		return logAndError(err, logger, "sending push")
	}
	logger.Debug(
		logkeys.Message, "processed repushes",
	)
	return nil
}
