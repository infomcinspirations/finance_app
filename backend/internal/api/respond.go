// Package api wires HTTP routes onto a store.Store.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/infomcinspirations/finance_app/backend/internal/model"
	"github.com/infomcinspirations/finance_app/backend/internal/store"
)

// maxBodyBytes caps request bodies so a malformed or hostile client cannot make
// the server allocate without bound.
const maxBodyBytes = 1 << 20 // 1 MiB

// errorBody is the single error shape every failing endpoint returns.
type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line is already sent, so there is nothing to tell the
		// client. Log it and move on.
		slog.Error("encode response", "error", err)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

// writeStoreErr maps a store error to the matching HTTP status. Unknown errors
// become 500 and are logged, never echoed back to the client.
func writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, model.ErrValidation):
		writeErr(w, http.StatusBadRequest, err.Error())
	default:
		slog.Error("unhandled store error", "error", err)
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}

// decodeJSON reads exactly one JSON value of type T from the request body and
// rejects unknown fields, so a typo in a client payload fails loudly instead of
// being silently dropped.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&v); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return v, false
	}
	// Reject trailing content: two concatenated objects are a client bug.
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		writeErr(w, http.StatusBadRequest, "body must contain exactly one JSON object")
		return v, false
	}
	return v, true
}
