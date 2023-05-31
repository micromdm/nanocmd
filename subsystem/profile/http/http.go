// Package http provides HTTP handlers for the Profile subsystem.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanocmd/http/api"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/ctxlog"
	"github.com/micromdm/nanocmd/log/logkeys"
	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/utils/mobileconfig"
)

// GetProfilesHandler returns an HTTP handler that returns profile metadata for all profile names.
func GetProfilesHandler(store storage.ReadStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		profiles, err := store.RetrieveProfileInfos(r.Context(), r.URL.Query()["name"])
		if err != nil {
			logger.Info(logkeys.Message, "retrieve profiles", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}
		logger.Debug(logkeys.Message, "retrieve profiles", "length", len(profiles))
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(profiles)
		if err != nil {
			logger.Info(logkeys.Message, "encoding json", logkeys.Error, err)
			return
		}
	}
}

var (
	ErrEmptyName  = errors.New("empty name")
	ErrNoSuchName = errors.New("no such name")
	ErrEmptyBody  = errors.New("empty body")
)

// DeleteProfileHandler returns an HTTP handler that deletes a named profile.
func DeleteProfileHandler(store storage.Storage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "name check", logkeys.Error, ErrEmptyName)
			api.JSONError(w, ErrEmptyName, http.StatusBadRequest)
			return
		}
		logger = logger.With("name", name)
		err := store.DeleteProfile(r.Context(), name)
		if err != nil {
			logger.Info(logkeys.Message, "delete profile", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetProfileHandler returns an HTTP handler that returns a named raw profile.
func GetProfileHandler(store storage.ReadStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "name check", logkeys.Error, ErrEmptyName)
			api.JSONError(w, ErrEmptyName, http.StatusBadRequest)
			return
		}
		logger = logger.With("name", name)
		profiles, err := store.RetrieveRawProfiles(r.Context(), []string{name})
		if err != nil {
			logger.Info(logkeys.Message, "retrieve profile", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}
		raw, ok := profiles[name]
		if !ok {
			// shouldn't actually happen, but be cautious just in case
			logger.Info(logkeys.Message, "access retrieved profile", logkeys.Error, ErrNoSuchName)
			api.JSONError(w, ErrNoSuchName, 0)

			return
		}
		w.Header().Set("Content-Type", "application/x-apple-aspen-config")
		w.Write(raw)
	}
}

// StoreProfileHandler returns an HTTP handler that uploads a named raw profile.
func StoreProfileHandler(store storage.Storage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		name := flow.Param(r.Context(), "name")
		if name == "" {
			logger.Info(logkeys.Message, "name check", logkeys.Error, ErrEmptyName)
			api.JSONError(w, ErrEmptyName, http.StatusBadRequest)
			return
		}
		logger = logger.With("name", name)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Info(logkeys.Message, "reading body", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}
		if len(raw) < 1 {
			logger.Info(logkeys.Message, "body check", logkeys.Error, ErrEmptyBody)
			api.JSONError(w, ErrEmptyBody, http.StatusBadRequest)
			return
		}
		mc := mobileconfig.Mobileconfig(raw)
		payload, _, err := mc.Parse()
		if err != nil {
			logger.Info(logkeys.Message, "parsing mobileconfig", logkeys.Error, err)
			api.JSONError(w, err, http.StatusBadRequest)
			return
		}
		info := storage.ProfileInfo{
			Identifier: payload.PayloadIdentifier,
			UUID:       payload.PayloadUUID,
		}
		err = store.StoreProfile(r.Context(), name, info, raw)
		if err != nil {
			logger.Info(logkeys.Message, "store profile", logkeys.Error, err)
			api.JSONError(w, err, 0)
			return
		}
		logger.Debug(
			logkeys.Message, "store profile",
			"identifier", info.Identifier,
			"uuid", info.UUID,
		)
		w.WriteHeader(http.StatusNoContent)
	}
}
