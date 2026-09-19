package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/infomcinspirations/finance_app/backend/internal/model"
	"github.com/infomcinspirations/finance_app/backend/internal/store"
)

// transactionRequest is the client-supplied half of a transaction.
//
// Amount is in minor units and signed: -1250 is $12.50 spent, 1250 is $12.50
// received. Date is optional and defaults to now.
type transactionRequest struct {
	AccountID string     `json:"accountId"`
	Date      *time.Time `json:"date"`
	Payee     string     `json:"payee"`
	Category  string     `json:"category"`
	Amount    int64      `json:"amount"`
	Note      string     `json:"note"`
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[transactionRequest](w, r)
	if !ok {
		return
	}

	t := model.Transaction{
		AccountID: req.AccountID,
		Payee:     req.Payee,
		Category:  req.Category,
		Amount:    req.Amount,
		Note:      req.Note,
	}
	if req.Date != nil {
		t.Date = *req.Date
	}
	t.Normalize()
	if err := t.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.store.CreateTransaction(r.Context(), t)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	f, err := parseTransactionFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	txs, err := h.store.ListTransactions(r.Context(), f)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, txs)
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	t, err := h.store.GetTransaction(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteTransaction(r.Context(), r.PathValue("id")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseTransactionFilter reads accountId, category, from and to from the query
// string. Dates accept either a full RFC 3339 timestamp or a plain YYYY-MM-DD.
func parseTransactionFilter(r *http.Request) (store.TransactionFilter, error) {
	q := r.URL.Query()
	f := store.TransactionFilter{
		AccountID: q.Get("accountId"),
		Category:  q.Get("category"),
	}

	from, err := parseQueryDate(q.Get("from"), false)
	if err != nil {
		return f, fmt.Errorf("from: %w", err)
	}
	to, err := parseQueryDate(q.Get("to"), true)
	if err != nil {
		return f, fmt.Errorf("to: %w", err)
	}
	if from != nil && to != nil && to.Before(*from) {
		return f, fmt.Errorf("to must not be earlier than from")
	}
	f.From, f.To = from, to
	return f, nil
}

// parseQueryDate turns a query parameter into a time. A bare date is widened to
// the start of that day, or to the last instant of it when endOfDay is set, so
// that ?to=2026-09-18 includes everything that happened on the 18th.
func parseQueryDate(s string, endOfDay bool) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		t = t.UTC()
		return &t, nil
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, fmt.Errorf("%q is not RFC 3339 or YYYY-MM-DD", s)
	}
	if endOfDay {
		d = d.Add(24*time.Hour - time.Nanosecond)
	}
	d = d.UTC()
	return &d, nil
}
