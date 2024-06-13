// Package http contains HTTP handlers for working with the inventory subsytem.
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/micromdm/nanocmd/http/api"
	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

var (
	ErrNoIDs     = errors.New("no IDs provided")
	ErrNoStorage = errors.New("no storage backend")
)

// RetrieveInventory returns an HTTP handler that retrieves inventory data for enrollment IDs.
func RetrieveInventory(store storage.ReadStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if store == nil {
			logger.Info(logkeys.Message, "retrieve inventory", logkeys.Error, ErrNoStorage)
			api.JSONError(w, ErrNoStorage, 0)
			return
		}

		ids := r.URL.Query()["id"]
		if len(ids) < 1 {
			logger.Info(logkeys.Message, "parameters", logkeys.Error, ErrNoIDs)
			api.JSONError(w, ErrNoIDs, http.StatusBadRequest)
			return
		}

		logger = logger.With(
			logkeys.FirstEnrollmentID, ids[0],
			logkeys.GenericCount, len(ids),
		)
		opts := &storage.SearchOptions{IDs: ids}
		idValues, err := store.RetrieveInventory(r.Context(), opts)
		if err != nil {
			logger.Info(logkeys.Message, "retrieve inventory", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}
		logger.Debug(
			logkeys.Message, "retrieved inventory",
		)
		w.Header().Set("Content-type", "application/json")
		err = json.NewEncoder(w).Encode(idValues)
		if err != nil {
			logger.Info(logkeys.Message, "encode response", logkeys.Error, err)
			return
		}
	}
}
