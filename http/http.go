// Package http includes handlers and utilties.
package http

import (
	"bytes"
	"io"
	"net/http"
)

// ReadAllAndReplaceBody reads all of r.Body and replaces it with a new byte buffer.
func ReadAllAndReplaceBody(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return b, err
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}

// DumpHandler outputs the body of the request to output.
func DumpHandler(next http.Handler, output io.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := ReadAllAndReplaceBody(r)
		output.Write(append(body, '\n'))
		next.ServeHTTP(w, r)
	}
}
