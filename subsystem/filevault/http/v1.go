package http

import (
	"net/http"
)

// Mux can register HTTP handlers.
// Ostensibly this supports flow router.
type Mux interface {
	// Handle registers the handler for the given pattern.
	Handle(pattern string, handler http.Handler, methods ...string)
}

// HandleAPIv1 registers the various API handlers into mux.
// API endpoint paths are prepended with prefix.
// Authentication or any other layered handlers are not present.
// They are assumed to be layered with mux, possibly at the Handle call.
// If prefix is empty and these handlers are used in sub-paths then
// handlers should have that sub-path stripped from the request.
func HandleAPIv1(prefix string, mux Mux) {
	mux.Handle(prefix+"/fvenable/profiletemplate", GetProfileTemplate(), "GET")
}
