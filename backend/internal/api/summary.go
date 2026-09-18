package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/lamcheryl/finance_app/backend/internal/model"
	"github.com/lamcheryl/finance_app/backend/internal/store"
)

// categoryTotal is spending and income rolled up under one category.
type categoryTotal struct {
	Category string `json:"category"`
	Inflow   int64  `json:"inflow"`
	Outflow  int64  `json:"outflow"` // positive magnitude
	Net      int64  `json:"net"`
	Count    int    `json:"count"`
}

// accountTotal is one account's closing balance plus its movement in the period.
type accountTotal struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	Net       int64  `json:"net"`
}

// summaryResponse reports period flows and point-in-time balances.
//
// Inflow, Outflow, Net and ByCategory cover only transactions inside the
// requested window. Balance figures are point-in-time: they include every
// transaction up to `to`, ignoring `from`, because a balance is not a flow.
type summaryResponse struct {
	From             *time.Time      `json:"from"`
	To               *time.Time      `json:"to"`
	Currencies       []string        `json:"currencies"`
	MixedCurrency    bool            `json:"mixedCurrency"`
	TotalBalance     int64           `json:"totalBalance"`
	Inflow           int64           `json:"inflow"`
	Outflow          int64           `json:"outflow"` // positive magnitude
	Net              int64           `json:"net"`
	TransactionCount int             `json:"transactionCount"`
	ByCategory       []categoryTotal `json:"byCategory"`
	ByAccount        []accountTotal  `json:"byAccount"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	f, err := parseTransactionFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	accounts, err := h.store.ListAccounts(ctx)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	// Flows inside the window.
	period, err := h.store.ListTransactions(ctx, f)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	// Everything up to the end of the window, for balances.
	toDate, err := h.store.ListTransactions(ctx, store.TransactionFilter{
		AccountID: f.AccountID,
		Category:  f.Category,
		To:        f.To,
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}

	out := summaryResponse{
		From:             f.From,
		To:               f.To,
		TransactionCount: len(period),
		ByCategory:       rollUpCategories(period),
	}

	periodSums, _ := aggregateByAccount(period)
	balanceSums, _ := aggregateByAccount(toDate)

	seen := make(map[string]bool)
	for _, a := range accounts {
		// An account filter narrows the summary to that one account.
		if f.AccountID != "" && a.ID != f.AccountID {
			continue
		}
		balance := a.OpeningBalance + balanceSums[a.ID]
		out.TotalBalance += balance
		out.ByAccount = append(out.ByAccount, accountTotal{
			AccountID: a.ID,
			Name:      a.Name,
			Currency:  a.Currency,
			Balance:   balance,
			Net:       periodSums[a.ID],
		})
		if !seen[a.Currency] {
			seen[a.Currency] = true
			out.Currencies = append(out.Currencies, a.Currency)
		}
	}
	sort.Strings(out.Currencies)
	// Totals are a plain sum of minor units. With more than one currency in play
	// that sum is meaningless, so say so rather than presenting a fake number.
	out.MixedCurrency = len(out.Currencies) > 1

	for _, t := range period {
		if t.IsOutflow() {
			out.Outflow += -t.Amount
		} else {
			out.Inflow += t.Amount
		}
	}
	out.Net = out.Inflow - out.Outflow

	if out.ByAccount == nil {
		out.ByAccount = []accountTotal{}
	}
	if out.Currencies == nil {
		out.Currencies = []string{}
	}
	writeJSON(w, http.StatusOK, out)
}

// rollUpCategories groups transactions by category, largest outflow first, so
// the biggest spending category leads the list.
func rollUpCategories(txs []model.Transaction) []categoryTotal {
	byCat := make(map[string]*categoryTotal)
	for _, t := range txs {
		c, ok := byCat[t.Category]
		if !ok {
			c = &categoryTotal{Category: t.Category}
			byCat[t.Category] = c
		}
		if t.IsOutflow() {
			c.Outflow += -t.Amount
		} else {
			c.Inflow += t.Amount
		}
		c.Net += t.Amount
		c.Count++
	}

	out := make([]categoryTotal, 0, len(byCat))
	for _, c := range byCat {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Outflow == out[j].Outflow {
			return out[i].Category < out[j].Category
		}
		return out[i].Outflow > out[j].Outflow
	})
	return out
}
