// Package http contains HTTP handlers for working with Command Plans.
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/micromdm/nanocmd/http/api"
	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

var (
	ErrNoName = errors.New("no name provided")
)

// GetHandler returns an HTTP handler that fetches a command plan.
func GetHandler(store storage.ReadStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "name parameter", logkeys.Error, ErrNoName)
			api.JSONError(w, ErrNoName, http.StatusBadRequest)
			return
		}

		logger = logger.With("name", name)
		cmdPlan, err := store.RetrieveCMDPlan(r.Context(), name)
		if err != nil {
			logger.Info(logkeys.Message, "retrieve cmdplan", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		logger.Debug(
			logkeys.Message, "retrieved cmdplan",
			logkeys.GenericCount, len(cmdPlan.ProfileNames),
		)
		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(cmdPlan); err != nil {
			logger.Info(logkeys.Message, "encoding json to body", logkeys.Error, err)
			return
		}
	}
}

// PutHandler returns an HTTP handler for uploading a command plan.
func PutHandler(store storage.Storage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "name parameter", logkeys.Error, ErrNoName)
			api.JSONError(w, ErrNoName, http.StatusBadRequest)
			return
		}

		logger = logger.With("name", name)
		cmdplan := new(storage.CMDPlan)
		err := json.NewDecoder(r.Body).Decode(cmdplan)
		if err != nil {
			logger.Info(logkeys.Message, "decoding body", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		if err = store.StoreCMDPlan(r.Context(), name, cmdplan); err != nil {
			logger.Info(logkeys.Message, "storing cmdplan", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		logger.Debug(logkeys.Message, "stored cmdplan")
		w.WriteHeader(http.StatusNoContent)
	}
}
