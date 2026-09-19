package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/infomcinspirations/finance_app/backend/internal/model"
)

// Memory is a goroutine-safe in-memory Store. It is the default for local
// development; data is lost when the process exits.
type Memory struct {
	mu           sync.RWMutex
	accounts     map[string]model.Account
	transactions map[string]model.Transaction
	now          func() time.Time
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		accounts:     make(map[string]model.Account),
		transactions: make(map[string]model.Transaction),
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// newID returns a random 128-bit hex identifier.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (m *Memory) CreateAccount(_ context.Context, a model.Account) (model.Account, error) {
	id, err := newID()
	if err != nil {
		return model.Account{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	a.ID = id
	a.CreatedAt = m.now()
	m.accounts[a.ID] = a
	return a, nil
}

func (m *Memory) GetAccount(_ context.Context, id string) (model.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	a, ok := m.accounts[id]
	if !ok {
		return model.Account{}, fmt.Errorf("account %s: %w", id, ErrNotFound)
	}
	return a, nil
}

// ListAccounts returns every account, oldest first.
func (m *Memory) ListAccounts(_ context.Context) ([]model.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]model.Account, 0, len(m.accounts))
	for _, a := range m.accounts {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// DeleteAccount removes an account and every transaction booked against it.
// Deleting the account without its transactions would leave orphans that no
// balance calculation could attribute.
func (m *Memory) DeleteAccount(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.accounts[id]; !ok {
		return fmt.Errorf("account %s: %w", id, ErrNotFound)
	}
	delete(m.accounts, id)
	for txID, t := range m.transactions {
		if t.AccountID == id {
			delete(m.transactions, txID)
		}
	}
	return nil
}

func (m *Memory) CreateTransaction(_ context.Context, t model.Transaction) (model.Transaction, error) {
	id, err := newID()
	if err != nil {
		return model.Transaction{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.accounts[t.AccountID]; !ok {
		return model.Transaction{}, fmt.Errorf("account %s: %w", t.AccountID, ErrConflict)
	}

	t.ID = id
	t.CreatedAt = m.now()
	m.transactions[t.ID] = t
	return t, nil
}

func (m *Memory) GetTransaction(_ context.Context, id string) (model.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.transactions[id]
	if !ok {
		return model.Transaction{}, fmt.Errorf("transaction %s: %w", id, ErrNotFound)
	}
	return t, nil
}

// ListTransactions returns matching transactions, newest first.
func (m *Memory) ListTransactions(_ context.Context, f TransactionFilter) ([]model.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]model.Transaction, 0, len(m.transactions))
	for _, t := range m.transactions {
		if f.matches(t) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date.Equal(out[j].Date) {
			return out[i].ID < out[j].ID
		}
		return out[i].Date.After(out[j].Date)
	})
	return out, nil
}

func (m *Memory) DeleteTransaction(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.transactions[id]; !ok {
		return fmt.Errorf("transaction %s: %w", id, ErrNotFound)
	}
	delete(m.transactions, id)
	return nil
}
