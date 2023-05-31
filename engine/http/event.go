package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/http/api"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/ctxlog"
	"github.com/micromdm/nanocmd/log/logkeys"
)

var (
	ErrMissingStore          = errors.New("missing store")
	ErrNoName                = errors.New("missing name parameter")
	ErrWorkflowNotRegistered = errors.New("workflow not registered")
)

// GetHandler retrieves and returns JSON of the named event subscription.
func GetHandler(store storage.ReadEventSubscriptionStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if store == nil {
			logger.Info(logkeys.Error, ErrMissingStore)
			api.JSONError(w, ErrMissingStore, 0)
			return
		}

		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "parameters", logkeys.Error, ErrNoName)
			api.JSONError(w, ErrNoName, http.StatusBadRequest)
			return
		}

		logger = logger.With("name", name)
		es, err := store.RetrieveEventSubscriptions(r.Context(), []string{name})
		if err != nil {
			logger.Info(logkeys.Message, "retrieve event subscription", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		logger.Debug(
			logkeys.Message, "retrieved event subscription",
			logkeys.GenericCount, len(es),
		)
		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(es[name]); err != nil {
			logger.Info(logkeys.Message, "encoding json to body", logkeys.Error, err)
			return
		}
	}
}

type WorkflowNameChecker interface {
	WorkflowRegistered(name string) bool
}

// PutHandler stores JSON of the named event subscription.
func PutHandler(store storage.EventSubscriptionStorage, chk WorkflowNameChecker, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if store == nil {
			logger.Info(logkeys.Error, ErrMissingStore)
			api.JSONError(w, ErrMissingStore, 0)
			return
		}

		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "parameters", logkeys.Error, ErrNoName)
			api.JSONError(w, ErrNoName, http.StatusBadRequest)
			return
		}

		logger = logger.With("name", name)
		es := new(storage.EventSubscription)
		err := json.NewDecoder(r.Body).Decode(es)
		if err != nil {
			logger.Info(logkeys.Message, "decoding body", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		logger = logger.With(logkeys.WorkflowName, es.Workflow)

		if err = es.Validate(); err != nil {
			logger.Info(logkeys.Message, "validating event subscription", logkeys.Error, err)
			api.JSONError(w, err, http.StatusBadRequest)
			return
		}

		if !chk.WorkflowRegistered(es.Workflow) {
			logger.Info(logkeys.Message, "checking workflow name", logkeys.Error, ErrWorkflowNotRegistered)
			api.JSONError(w, ErrWorkflowNotRegistered, http.StatusBadRequest)
			return
		}

		if err = store.StoreEventSubscription(r.Context(), name, es); err != nil {
			logger.Info(logkeys.Message, "storing event subscription", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}

		logger.Debug(logkeys.Message, "stored event subscription")
		w.WriteHeader(http.StatusNoContent)
	}
}
