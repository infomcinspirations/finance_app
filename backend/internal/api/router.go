package api

import (
	"net/http"

	"github.com/infomcinspirations/finance_app/backend/internal/store"
)

// Handler holds the dependencies every route needs.
type Handler struct {
	store          store.Store
	allowedOrigins []string
}

// New returns an http.Handler serving the whole application. allowedOrigins is
// the exact list of browser origins permitted by CORS.
//
// static serves the frontend on every path the API does not claim. Passing nil
// makes those paths return a JSON 404, which is what the tests want and what a
// binary built without a frontend should do.
func New(s store.Store, allowedOrigins []string, static http.Handler) http.Handler {
	h := &Handler{store: s, allowedOrigins: allowedOrigins}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", h.health)

	mux.HandleFunc("GET /api/v1/accounts", h.listAccounts)
	mux.HandleFunc("POST /api/v1/accounts", h.createAccount)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.getAccount)
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", h.deleteAccount)

	mux.HandleFunc("GET /api/v1/transactions", h.listTransactions)
	mux.HandleFunc("POST /api/v1/transactions", h.createTransaction)
	mux.HandleFunc("GET /api/v1/transactions/{id}", h.getTransaction)
	mux.HandleFunc("DELETE /api/v1/transactions/{id}", h.deleteTransaction)

	mux.HandleFunc("GET /api/v1/summary", h.summary)

	// An unrecognised path under /api/ is a client calling an endpoint that does
	// not exist, so it gets a JSON 404 rather than being handed the frontend.
	// This pattern is more specific than "/" below, so it wins for those paths.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "no such endpoint: "+r.Method+" "+r.URL.Path)
	})

	// Everything else is the frontend's, including client-side routes that must
	// survive a hard refresh.
	if static != nil {
		mux.Handle("/", static)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			writeErr(w, http.StatusNotFound, "no such endpoint: "+r.Method+" "+r.URL.Path)
		})
	}

	// Outermost first: panics are recovered, then logged, then CORS is applied.
	return recoverPanics(logRequests(withCORS(h.allowedOrigins, mux)))
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
