// Package store defines persistence for finance_app and ships an in-memory
// implementation. Everything the API needs goes through the Store interface so
// a SQL-backed implementation can replace the in-memory one without touching
// the handlers.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/infomcinspirations/finance_app/backend/internal/model"
)

var (
	// ErrNotFound means the requested record does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the write would violate a referential rule, such as a
	// transaction pointing at an account that is not there.
	ErrConflict = errors.New("conflict")
)

// TransactionFilter narrows a transaction listing. Zero values mean "no
// constraint on this field".
type TransactionFilter struct {
	AccountID string
	Category  string
	From      *time.Time
	To        *time.Time
}

// matches reports whether t satisfies every constraint set on f.
func (f TransactionFilter) matches(t model.Transaction) bool {
	if f.AccountID != "" && t.AccountID != f.AccountID {
		return false
	}
	if f.Category != "" && t.Category != f.Category {
		return false
	}
	if f.From != nil && t.Date.Before(*f.From) {
		return false
	}
	// To is inclusive of the instant itself.
	if f.To != nil && t.Date.After(*f.To) {
		return false
	}
	return true
}

// Store is the persistence contract for the whole application.
type Store interface {
	CreateAccount(ctx context.Context, a model.Account) (model.Account, error)
	GetAccount(ctx context.Context, id string) (model.Account, error)
	ListAccounts(ctx context.Context) ([]model.Account, error)
	DeleteAccount(ctx context.Context, id string) error

	CreateTransaction(ctx context.Context, t model.Transaction) (model.Transaction, error)
	GetTransaction(ctx context.Context, id string) (model.Transaction, error)
	ListTransactions(ctx context.Context, f TransactionFilter) ([]model.Transaction, error)
	DeleteTransaction(ctx context.Context, id string) error
}
