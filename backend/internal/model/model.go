// Package model holds the core domain types for finance_app.
//
// All monetary values are integer minor units (cents for USD). Floating point
// is never used for money: 0.1 + 0.2 != 0.3 in binary floating point, and a
// ledger that cannot add up exactly is not a ledger.
package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// AccountType classifies an account for reporting and sign conventions.
type AccountType string

const (
	AccountChecking   AccountType = "checking"
	AccountSavings    AccountType = "savings"
	AccountCredit     AccountType = "credit"
	AccountInvestment AccountType = "investment"
	AccountCash       AccountType = "cash"
)

// Valid reports whether t is a known account type.
func (t AccountType) Valid() bool {
	switch t {
	case AccountChecking, AccountSavings, AccountCredit, AccountInvestment, AccountCash:
		return true
	}
	return false
}

// Account is a place money sits. OpeningBalance is the balance at the moment
// the account was added, before any recorded transaction.
type Account struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Type           AccountType `json:"type"`
	Currency       string      `json:"currency"`
	OpeningBalance int64       `json:"openingBalance"`
	CreatedAt      time.Time   `json:"createdAt"`
}

// Transaction is a single movement of money against one account.
//
// Amount is signed: negative is an outflow (spending), positive is an inflow
// (income, deposits). A transfer between two accounts is recorded as two
// transactions; the API does not model transfers as a first-class object yet.
type Transaction struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Date      time.Time `json:"date"`
	Payee     string    `json:"payee"`
	Category  string    `json:"category"`
	Amount    int64     `json:"amount"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// ErrValidation wraps every field-level rejection so the API layer can map it
// to 400 without inspecting message text.
var ErrValidation = errors.New("validation failed")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
}

// Normalize trims incoming string fields and applies defaults in place.
func (a *Account) Normalize() {
	a.Name = strings.TrimSpace(a.Name)
	a.Currency = strings.ToUpper(strings.TrimSpace(a.Currency))
	if a.Currency == "" {
		a.Currency = "USD"
	}
	if a.Type == "" {
		a.Type = AccountChecking
	}
}

// Validate reports the first problem that would make a an unusable record.
func (a Account) Validate() error {
	switch {
	case a.Name == "":
		return invalid("name is required")
	case len(a.Name) > 120:
		return invalid("name must be 120 characters or fewer")
	case !a.Type.Valid():
		return invalid("type %q is not one of checking, savings, credit, investment, cash", a.Type)
	case len(a.Currency) != 3:
		return invalid("currency must be a 3-letter ISO 4217 code, got %q", a.Currency)
	}
	return nil
}

// Normalize trims incoming string fields and applies defaults in place.
func (t *Transaction) Normalize() {
	t.AccountID = strings.TrimSpace(t.AccountID)
	t.Payee = strings.TrimSpace(t.Payee)
	t.Category = strings.TrimSpace(t.Category)
	t.Note = strings.TrimSpace(t.Note)
	if t.Category == "" {
		t.Category = "Uncategorized"
	}
	if t.Date.IsZero() {
		t.Date = time.Now().UTC()
	}
	t.Date = t.Date.UTC()
}

// Validate reports the first problem that would make t an unusable record.
func (t Transaction) Validate() error {
	switch {
	case t.AccountID == "":
		return invalid("accountId is required")
	case t.Payee == "":
		return invalid("payee is required")
	case len(t.Payee) > 200:
		return invalid("payee must be 200 characters or fewer")
	case len(t.Category) > 80:
		return invalid("category must be 80 characters or fewer")
	case len(t.Note) > 1000:
		return invalid("note must be 1000 characters or fewer")
	case t.Amount == 0:
		return invalid("amount must be non-zero (negative for outflow, positive for inflow)")
	}
	return nil
}

// IsOutflow reports whether the transaction moves money out of the account.
func (t Transaction) IsOutflow() bool { return t.Amount < 0 }
