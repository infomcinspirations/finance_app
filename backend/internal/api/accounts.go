package api

import (
	"net/http"

	"github.com/lamcheryl/finance_app/backend/internal/model"
	"github.com/lamcheryl/finance_app/backend/internal/store"
)

// accountRequest is the client-supplied half of an account. ID and CreatedAt are
// assigned by the server, so they are deliberately absent.
type accountRequest struct {
	Name           string            `json:"name"`
	Type           model.AccountType `json:"type"`
	Currency       string            `json:"currency"`
	OpeningBalance int64             `json:"openingBalance"`
}

// accountResponse is an account plus the figures derived from its transactions.
// Balance is the opening balance plus every transaction amount.
type accountResponse struct {
	model.Account
	Balance          int64 `json:"balance"`
	TransactionCount int   `json:"transactionCount"`
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[accountRequest](w, r)
	if !ok {
		return
	}

	a := model.Account{
		Name:           req.Name,
		Type:           req.Type,
		Currency:       req.Currency,
		OpeningBalance: req.OpeningBalance,
	}
	a.Normalize()
	if err := a.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.store.CreateAccount(r.Context(), a)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	// A brand-new account has no transactions, so its balance is its opening
	// balance by definition.
	writeJSON(w, http.StatusCreated, accountResponse{Account: created, Balance: created.OpeningBalance})
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accounts, err := h.store.ListAccounts(ctx)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	txs, err := h.store.ListTransactions(ctx, store.TransactionFilter{})
	if err != nil {
		writeStoreErr(w, err)
		return
	}

	sums, counts := aggregateByAccount(txs)

	out := make([]accountResponse, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, accountResponse{
			Account:          a,
			Balance:          a.OpeningBalance + sums[a.ID],
			TransactionCount: counts[a.ID],
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	a, err := h.store.GetAccount(ctx, id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	txs, err := h.store.ListTransactions(ctx, store.TransactionFilter{AccountID: id})
	if err != nil {
		writeStoreErr(w, err)
		return
	}

	sums, counts := aggregateByAccount(txs)
	writeJSON(w, http.StatusOK, accountResponse{
		Account:          a,
		Balance:          a.OpeningBalance + sums[id],
		TransactionCount: counts[id],
	})
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteAccount(r.Context(), r.PathValue("id")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// aggregateByAccount totals transaction amounts and counts per account ID.
func aggregateByAccount(txs []model.Transaction) (sums map[string]int64, counts map[string]int) {
	sums = make(map[string]int64)
	counts = make(map[string]int)
	for _, t := range txs {
		sums[t.AccountID] += t.Amount
		counts[t.AccountID]++
	}
	return sums, counts
}
