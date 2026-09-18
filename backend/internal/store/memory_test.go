package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lamcheryl/finance_app/backend/internal/model"
)

func newAccount(t *testing.T, m *Memory, name string, opening int64) model.Account {
	t.Helper()
	a, err := m.CreateAccount(context.Background(), model.Account{
		Name: name, Type: model.AccountChecking, Currency: "USD", OpeningBalance: opening,
	})
	if err != nil {
		t.Fatalf("CreateAccount(%q): %v", name, err)
	}
	return a
}

func TestCreateAccountAssignsID(t *testing.T) {
	m := NewMemory()
	a := newAccount(t, m, "Everyday", 10_000)

	if a.ID == "" {
		t.Fatal("expected a generated ID, got empty string")
	}
	if a.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	got, err := m.GetAccount(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.Name != "Everyday" || got.OpeningBalance != 10_000 {
		t.Errorf("round trip mismatch: got %+v", got)
	}
}

func TestGetAccountNotFound(t *testing.T) {
	m := NewMemory()
	_, err := m.GetAccount(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestCreateTransactionRequiresExistingAccount(t *testing.T) {
	m := NewMemory()
	_, err := m.CreateTransaction(context.Background(), model.Transaction{
		AccountID: "ghost", Payee: "Cafe", Amount: -500, Date: time.Now().UTC(),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("got %v, want ErrConflict", err)
	}
}

// Deleting an account must take its transactions with it; an orphaned
// transaction would silently distort every later total.
func TestDeleteAccountCascadesToTransactions(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	keep := newAccount(t, m, "Keep", 0)
	drop := newAccount(t, m, "Drop", 0)

	for _, accountID := range []string{keep.ID, drop.ID, drop.ID} {
		if _, err := m.CreateTransaction(ctx, model.Transaction{
			AccountID: accountID, Payee: "Shop", Category: "Misc",
			Amount: -100, Date: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("CreateTransaction: %v", err)
		}
	}

	if err := m.DeleteAccount(ctx, drop.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	remaining, err := m.ListTransactions(ctx, TransactionFilter{})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("got %d transactions, want 1", len(remaining))
	}
	if remaining[0].AccountID != keep.ID {
		t.Errorf("surviving transaction belongs to %s, want %s", remaining[0].AccountID, keep.ID)
	}
}

func TestListTransactionsFilters(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	a := newAccount(t, m, "Everyday", 0)
	b := newAccount(t, m, "Savings", 0)

	mk := func(accountID, category string, day int) {
		t.Helper()
		if _, err := m.CreateTransaction(ctx, model.Transaction{
			AccountID: accountID, Payee: "P", Category: category, Amount: -100,
			Date: time.Date(2026, 9, day, 12, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("CreateTransaction: %v", err)
		}
	}
	mk(a.ID, "Groceries", 1)
	mk(a.ID, "Rent", 10)
	mk(b.ID, "Groceries", 20)

	day := func(d int) *time.Time {
		v := time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC)
		return &v
	}

	tests := []struct {
		name string
		f    TransactionFilter
		want int
	}{
		{"no filter", TransactionFilter{}, 3},
		{"by account", TransactionFilter{AccountID: a.ID}, 2},
		{"by category", TransactionFilter{Category: "Groceries"}, 2},
		{"account and category", TransactionFilter{AccountID: a.ID, Category: "Groceries"}, 1},
		{"from bound", TransactionFilter{From: day(5)}, 2},
		{"to bound", TransactionFilter{To: day(15)}, 2},
		{"window", TransactionFilter{From: day(5), To: day(15)}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.ListTransactions(ctx, tc.f)
			if err != nil {
				t.Fatalf("ListTransactions: %v", err)
			}
			if len(got) != tc.want {
				t.Errorf("got %d transactions, want %d", len(got), tc.want)
			}
		})
	}
}

// Listings drive the UI ordering, so the contract is worth pinning down.
func TestListTransactionsOrdersNewestFirst(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	a := newAccount(t, m, "Everyday", 0)

	for _, day := range []int{5, 1, 10} {
		if _, err := m.CreateTransaction(ctx, model.Transaction{
			AccountID: a.ID, Payee: "P", Category: "C", Amount: -1,
			Date: time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("CreateTransaction: %v", err)
		}
	}

	got, err := m.ListTransactions(ctx, TransactionFilter{})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Date.Before(got[i].Date) {
			t.Fatalf("not newest-first at index %d: %v then %v", i, got[i-1].Date, got[i].Date)
		}
	}
}
