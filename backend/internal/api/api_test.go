package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lamcheryl/finance_app/backend/internal/store"
)

func newServer(t *testing.T) http.Handler {
	t.Helper()
	return New(store.NewMemory(), []string{"http://localhost:5173"})
}

// do issues a request and decodes the JSON body into out when out is non-nil.
func do(t *testing.T, h http.Handler, method, path, body string, out any) *httptest.ResponseRecorder {
	t.Helper()

	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if out != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s %s response %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec
}

func createAccountFixture(t *testing.T, h http.Handler, name string, opening int64) string {
	t.Helper()
	var created struct{ ID string }
	body := `{"name":"` + name + `","type":"checking","currency":"USD","openingBalance":` +
		jsonNumber(opening) + `}`
	rec := do(t, h, http.MethodPost, "/api/v1/accounts", body, &created)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create account: got %d, body %s", rec.Code, rec.Body.String())
	}
	return created.ID
}

func jsonNumber(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestHealth(t *testing.T) {
	rec := do(t, newServer(t), http.MethodGet, "/api/v1/health", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestUnknownEndpointReturnsJSON(t *testing.T) {
	var body errorBody
	rec := do(t, newServer(t), http.MethodGet, "/api/v1/nope", "", &body)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
	if body.Error == "" {
		t.Error("expected an error message in the body")
	}
}

func TestCreateAccountValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"valid", `{"name":"Everyday","type":"checking","currency":"USD","openingBalance":1000}`, http.StatusCreated},
		{"defaults applied", `{"name":"Everyday"}`, http.StatusCreated},
		{"blank name", `{"name":"   ","type":"checking"}`, http.StatusBadRequest},
		{"bad type", `{"name":"X","type":"crypto"}`, http.StatusBadRequest},
		{"bad currency", `{"name":"X","currency":"DOLLARS"}`, http.StatusBadRequest},
		{"unknown field", `{"name":"X","colour":"red"}`, http.StatusBadRequest},
		{"malformed json", `{"name":`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, newServer(t), http.MethodPost, "/api/v1/accounts", tc.body, nil)
			if rec.Code != tc.want {
				t.Errorf("got %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestCreateAccountAppliesDefaults(t *testing.T) {
	var got struct {
		Type     string
		Currency string
		Balance  int64
	}
	do(t, newServer(t), http.MethodPost, "/api/v1/accounts", `{"name":"Everyday","openingBalance":500}`, &got)

	if got.Type != "checking" {
		t.Errorf("type = %q, want checking", got.Type)
	}
	if got.Currency != "USD" {
		t.Errorf("currency = %q, want USD", got.Currency)
	}
	if got.Balance != 500 {
		t.Errorf("balance = %d, want 500 (opening balance of a fresh account)", got.Balance)
	}
}

func TestCreateTransactionValidation(t *testing.T) {
	h := newServer(t)
	id := createAccountFixture(t, h, "Everyday", 0)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"valid outflow", `{"accountId":"` + id + `","payee":"Cafe","category":"Food","amount":-450}`, http.StatusCreated},
		{"valid inflow", `{"accountId":"` + id + `","payee":"Payroll","category":"Income","amount":250000}`, http.StatusCreated},
		{"zero amount", `{"accountId":"` + id + `","payee":"Cafe","amount":0}`, http.StatusBadRequest},
		{"missing payee", `{"accountId":"` + id + `","amount":-100}`, http.StatusBadRequest},
		{"missing account", `{"payee":"Cafe","amount":-100}`, http.StatusBadRequest},
		{"unknown account", `{"accountId":"ghost","payee":"Cafe","amount":-100}`, http.StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/api/v1/transactions", tc.body, nil)
			if rec.Code != tc.want {
				t.Errorf("got %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestTransactionWithoutCategoryIsUncategorized(t *testing.T) {
	h := newServer(t)
	id := createAccountFixture(t, h, "Everyday", 0)

	var got struct{ Category string }
	do(t, h, http.MethodPost, "/api/v1/transactions",
		`{"accountId":"`+id+`","payee":"Cafe","amount":-450}`, &got)

	if got.Category != "Uncategorized" {
		t.Errorf("category = %q, want Uncategorized", got.Category)
	}
}

// The balance an account reports must be its opening balance plus every
// transaction against it — the single most important number in the app.
func TestAccountBalanceIncludesTransactions(t *testing.T) {
	h := newServer(t)
	id := createAccountFixture(t, h, "Everyday", 100_000) // $1,000.00

	for _, amount := range []int64{-450, -2_000, 25_000} {
		rec := do(t, h, http.MethodPost, "/api/v1/transactions",
			`{"accountId":"`+id+`","payee":"P","category":"C","amount":`+jsonNumber(amount)+`}`, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create transaction: got %d, body %s", rec.Code, rec.Body.String())
		}
	}

	var got struct {
		Balance          int64
		TransactionCount int
	}
	do(t, h, http.MethodGet, "/api/v1/accounts/"+id, "", &got)

	const want = 100_000 - 450 - 2_000 + 25_000
	if got.Balance != want {
		t.Errorf("balance = %d, want %d", got.Balance, want)
	}
	if got.TransactionCount != 3 {
		t.Errorf("transactionCount = %d, want 3", got.TransactionCount)
	}
}

func TestDeleteAccount(t *testing.T) {
	h := newServer(t)
	id := createAccountFixture(t, h, "Everyday", 0)

	if rec := do(t, h, http.MethodDelete, "/api/v1/accounts/"+id, "", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/v1/accounts/"+id, "", nil); rec.Code != http.StatusNotFound {
		t.Errorf("get after delete: got %d, want 404", rec.Code)
	}
	if rec := do(t, h, http.MethodDelete, "/api/v1/accounts/"+id, "", nil); rec.Code != http.StatusNotFound {
		t.Errorf("double delete: got %d, want 404", rec.Code)
	}
}

func TestSummaryTotals(t *testing.T) {
	h := newServer(t)
	a := createAccountFixture(t, h, "Everyday", 50_000)
	b := createAccountFixture(t, h, "Savings", 200_000)

	mk := func(accountID, category string, amount int64, date string) {
		t.Helper()
		body := `{"accountId":"` + accountID + `","payee":"P","category":"` + category +
			`","amount":` + jsonNumber(amount) + `,"date":"` + date + `"}`
		if rec := do(t, h, http.MethodPost, "/api/v1/transactions", body, nil); rec.Code != http.StatusCreated {
			t.Fatalf("create transaction: got %d, body %s", rec.Code, rec.Body.String())
		}
	}
	mk(a, "Groceries", -12_000, "2026-09-02T10:00:00Z")
	mk(a, "Groceries", -8_000, "2026-09-05T10:00:00Z")
	mk(a, "Rent", -150_000, "2026-09-01T10:00:00Z")
	mk(a, "Income", 300_000, "2026-09-03T10:00:00Z")
	mk(b, "Transfer", 10_000, "2026-08-15T10:00:00Z") // outside the window

	var got summaryResponse
	rec := do(t, h, http.MethodGet, "/api/v1/summary?from=2026-09-01&to=2026-09-30", "", &got)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary: got %d, body %s", rec.Code, rec.Body.String())
	}

	if got.Inflow != 300_000 {
		t.Errorf("inflow = %d, want 300000", got.Inflow)
	}
	if want := int64(12_000 + 8_000 + 150_000); got.Outflow != want {
		t.Errorf("outflow = %d, want %d", got.Outflow, want)
	}
	if want := int64(300_000) - int64(170_000); got.Net != want {
		t.Errorf("net = %d, want %d", got.Net, want)
	}
	if got.TransactionCount != 4 {
		t.Errorf("transactionCount = %d, want 4 (August is outside the window)", got.TransactionCount)
	}

	// Balances are point in time, so the August transfer still counts.
	const wantBalance = 50_000 - 170_000 + 300_000 + 200_000 + 10_000
	if got.TotalBalance != wantBalance {
		t.Errorf("totalBalance = %d, want %d", got.TotalBalance, wantBalance)
	}

	// Categories are ordered by largest outflow.
	if len(got.ByCategory) == 0 || got.ByCategory[0].Category != "Rent" {
		t.Errorf("expected Rent to lead byCategory, got %+v", got.ByCategory)
	}
	for _, c := range got.ByCategory {
		if c.Category == "Groceries" && c.Outflow != 20_000 {
			t.Errorf("Groceries outflow = %d, want 20000", c.Outflow)
		}
	}
	if got.MixedCurrency {
		t.Error("mixedCurrency = true, want false with a single USD book")
	}
}

// Summing minor units across currencies produces a number that means nothing.
// The API has to flag that rather than present it as a total.
func TestSummaryFlagsMixedCurrency(t *testing.T) {
	h := newServer(t)
	createAccountFixture(t, h, "US", 1_000)
	do(t, h, http.MethodPost, "/api/v1/accounts", `{"name":"EU","type":"savings","currency":"EUR"}`, nil)

	var got summaryResponse
	do(t, h, http.MethodGet, "/api/v1/summary", "", &got)

	if !got.MixedCurrency {
		t.Error("mixedCurrency = false, want true")
	}
	if len(got.Currencies) != 2 {
		t.Errorf("currencies = %v, want 2 entries", got.Currencies)
	}
}

func TestSummaryRejectsBadDateRange(t *testing.T) {
	tests := []string{
		"/api/v1/summary?from=not-a-date",
		"/api/v1/summary?to=13/01/2026",
		"/api/v1/summary?from=2026-09-30&to=2026-09-01",
	}
	h := newServer(t)
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			if rec := do(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusBadRequest {
				t.Errorf("got %d, want 400", rec.Code)
			}
		})
	}
}

func TestCORSAllowsConfiguredOriginOnly(t *testing.T) {
	h := newServer(t)

	tests := []struct {
		origin string
		want   string
	}{
		{"http://localhost:5173", "http://localhost:5173"},
		{"http://evil.example", ""},
	}
	for _, tc := range tests {
		t.Run(tc.origin, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
			r.Header.Set("Origin", tc.origin)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRequestBodyIsSizeLimited(t *testing.T) {
	huge := `{"name":"` + strings.Repeat("a", 2<<20) + `"}`
	rec := do(t, newServer(t), http.MethodPost, "/api/v1/accounts", huge, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 for an oversized body", rec.Code)
	}
}
