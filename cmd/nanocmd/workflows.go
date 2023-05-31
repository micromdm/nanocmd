package main

import (
	"fmt"

	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/workflow"
	"github.com/micromdm/nanocmd/workflow/cmdplan"
	"github.com/micromdm/nanocmd/workflow/fvenable"
	"github.com/micromdm/nanocmd/workflow/fvrotate"
	"github.com/micromdm/nanocmd/workflow/inventory"
	"github.com/micromdm/nanocmd/workflow/lock"
	"github.com/micromdm/nanocmd/workflow/profile"
)

type registerer interface {
	RegisterWorkflow(w workflow.Workflow) error
}

func registerWorkflows(logger log.Logger, r registerer, s *storageConfig, e workflow.StepEnqueuer) error {
	var w workflow.Workflow
	var err error

	if w, err = inventory.New(e, s.inventory); err != nil {
		return fmt.Errorf("creating inventory workflow: %w", err)
	} else if err = r.RegisterWorkflow(w); err != nil {
		return fmt.Errorf("registering inventory workflow: %w", err)
	}

	if w, err = profile.New(e, s.profile, profile.WithLogger(logger)); err != nil {
		return fmt.Errorf("creating profile workflow: %w", err)
	} else if err = r.RegisterWorkflow(w); err != nil {
		return fmt.Errorf("registering profile workflow: %w", err)
	}

	if w, err = fvenable.New(e, s.filevault, s.profile, fvenable.WithLogger(logger)); err != nil {
		return fmt.Errorf("creating fvenable workflow: %w", err)
	} else if err = r.RegisterWorkflow(w); err != nil {
		return fmt.Errorf("registering fvenable workflow: %w", err)
	}

	if w, err = fvrotate.New(e, s.filevault, fvrotate.WithLogger(logger)); err != nil {
		return fmt.Errorf("creating fvrotate workflow: %w", err)
	} else if err = r.RegisterWorkflow(w); err != nil {
		return fmt.Errorf("registering fvrotate workflow: %w", err)
	}

	if w, err = cmdplan.New(e, s.cmdplan, s.profile, cmdplan.WithLogger(logger)); err != nil {
		return fmt.Errorf("creating cmdplan workflow: %w", err)
	} else if err = r.RegisterWorkflow(w); err != nil {
		return fmt.Errorf("registering cmdplan workflow: %w", err)
	}

	if w, err = lock.New(e, s.inventory, lock.WithLogger(logger)); err != nil {
		return fmt.Errorf("creating lock workflow: %w", err)
	} else if err = r.RegisterWorkflow(w); err != nil {
		return fmt.Errorf("registering lock workflow: %w", err)
	}

	return nil
}
