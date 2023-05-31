// Package http contains HTTP handlers that work with the NanoCMD engine.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanocmd/http/api"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/ctxlog"
	"github.com/micromdm/nanocmd/log/logkeys"
	"github.com/micromdm/nanocmd/workflow"
)

var (
	ErrNoIDs     = errors.New("no IDs provided")
	ErrNoStarter = errors.New("missing workflow starter")
)

type WorkflowStarter interface {
	StartWorkflow(ctx context.Context, name string, context []byte, ids []string, e *workflow.Event, mdmCtx *workflow.MDMContext) (string, error)
}

// StartWorkflowHandler creates a HandlerFunc that starts a workflow.
func StartWorkflowHandler(starter WorkflowStarter, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		ids := r.URL.Query()["id"]
		if len(ids) < 1 {
			logger.Info(logkeys.Message, "parameters", logkeys.Error, ErrNoIDs)
			api.JSONError(w, ErrNoIDs, http.StatusBadRequest)
			return
		}

		name := flow.Param(r.Context(), "name")
		logger = logger.With(
			logkeys.FirstEnrollmentID, ids[0],
			logkeys.WorkflowName, name,
		)
		if starter == nil {
			logger.Info(logkeys.Message, "starting workflow", logkeys.Error, ErrNoStarter)
			api.JSONError(w, ErrNoStarter, 0)
			return
		}

		logger.Debug(logkeys.Message, "starting workflow")
		instanceID, err := starter.StartWorkflow(
			r.Context(),
			name,
			[]byte(r.URL.Query().Get("context")),
			ids,
			nil,
			nil,
		)
		if err != nil {
			logger.Info(logkeys.Message, "starting workflow", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		jsonResp := &struct {
			InstanceID string `json:"instance_id"`
		}{InstanceID: instanceID}
		if err = json.NewEncoder(w).Encode(jsonResp); err != nil {
			logger.Info(logkeys.Message, "encoding json response", logkeys.Error, err)
		}
	}
}
