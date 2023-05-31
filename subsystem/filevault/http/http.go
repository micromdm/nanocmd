// Package http provides HTTP handlers related to the FileVault enable workflow.
package http

import (
	"net/http"

	"github.com/micromdm/nanocmd/workflow/fvenable"
)

// GetProfileTemplate returns an HTTP handler that serves the fvenable profile template.
func GetProfileTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/x-apple-aspen-config")
		w.Write([]byte(fvenable.ProfileTemplate))
	}
}
