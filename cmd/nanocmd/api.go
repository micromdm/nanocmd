package main

import (
	enginehttp "github.com/micromdm/nanocmd/engine/http"
	cmdplanhttp "github.com/micromdm/nanocmd/subsystem/cmdplan/http"
	fvenablehttp "github.com/micromdm/nanocmd/subsystem/filevault/http"
	invhttp "github.com/micromdm/nanocmd/subsystem/inventory/http"
	profhttp "github.com/micromdm/nanocmd/subsystem/profile/http"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanolib/log"
)

type engineLike interface {
	enginehttp.WorkflowNameChecker
	enginehttp.WorkflowStarter
}

func handlers(mux *flow.Mux, logger log.Logger, e engineLike, s *storageConfig) {
	// engine (workflow)

	mux.Handle(
		"/v1/workflow/:name/start",
		enginehttp.StartWorkflowHandler(e, logger.With("handler", "start workflow")),
		"POST",
	)

	// engine (event subscriptions)

	mux.Handle(
		"/v1/event/:name",
		enginehttp.GetHandler(s.event, logger.With("handler", "get event")),
		"GET",
	)

	mux.Handle(
		"/v1/event/:name",
		enginehttp.PutHandler(s.event, e, logger.With("handler", "put event")),
		"PUT",
	)

	// inventory

	mux.Handle(
		"/v1/inventory",
		invhttp.RetrieveInventory(s.inventory, logger.With("handler", "retrieve enrollments")),
		"GET",
	)

	// profiles

	mux.Handle(
		"/v1/profile/:name",
		profhttp.StoreProfileHandler(s.profile, logger.With("handler", "store profile")),
		"PUT",
	)

	mux.Handle(
		"/v1/profile/:name",
		profhttp.GetProfileHandler(s.profile, logger.With("handler", "get raw profile")),
		"GET",
	)

	mux.Handle(
		"/v1/profile/:name",
		profhttp.DeleteProfileHandler(s.profile, logger.With("handler", "delete profile")),
		"DELETE",
	)

	mux.Handle(
		"/v1/profiles",
		profhttp.GetProfilesHandler(s.profile, logger.With("handler", "get profiles")),
		"GET",
	)

	// fvenable

	mux.Handle("/v1/fvenable/profiletemplate", fvenablehttp.GetProfileTemplate(), "GET")

	// cmdplan

	mux.Handle(
		"/v1/cmdplan/:name",
		cmdplanhttp.GetHandler(s.cmdplan, logger.With("handler", "get cmdplan")),
		"GET",
	)

	mux.Handle(
		"/v1/cmdplan/:name",
		cmdplanhttp.PutHandler(s.cmdplan, logger.With("handler", "put cmdplan")),
		"PUT",
	)
}
