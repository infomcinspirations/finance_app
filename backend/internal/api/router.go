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

// New returns an http.Handler serving the whole API. allowedOrigins is the exact
// list of browser origins permitted by CORS.
func New(s store.Store, allowedOrigins []string) http.Handler {
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

	// Anything not matched above is a 404 in our JSON error shape rather than
	// net/http's plain-text default.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "no such endpoint: "+r.Method+" "+r.URL.Path)
	})

	// Outermost first: panics are recovered, then logged, then CORS is applied.
	return recoverPanics(logRequests(withCORS(h.allowedOrigins, mux)))
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
