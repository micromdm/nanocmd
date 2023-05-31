package api

import (
	"encoding/json"
	"net/http"
)

// JSONError encodes err as JSON to w.
func JSONError(w http.ResponseWriter, err error, statusCode int) {
	jsonErr := &struct {
		Err string `json:"error"`
	}{Err: err.Error()}
	w.Header().Set("Content-type", "application/json")
	if statusCode < 1 {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(jsonErr)
}
