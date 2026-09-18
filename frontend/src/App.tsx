import { useCallback, useEffect, useMemo, useState } from "react";

import { api, type TransactionQuery } from "./api";
import { AccountForm } from "./components/AccountForm";
import { AccountList } from "./components/AccountList";
import { Filters, type FilterState } from "./components/Filters";
import { SummaryPanel } from "./components/SummaryPanel";
import { TransactionForm } from "./components/TransactionForm";
import { TransactionList } from "./components/TransactionList";
import type {
  Account,
  NewAccount,
  NewTransaction,
  Summary,
  Transaction,
} from "./types";

const EMPTY_FILTERS: FilterState = {
  accountId: "",
  category: "",
  from: "",
  to: "",
};

export function App() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [filters, setFilters] = useState<FilterState>(EMPTY_FILTERS);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const query = useMemo<TransactionQuery>(
    () => ({
      accountId: filters.accountId || undefined,
      category: filters.category || undefined,
      from: filters.from || undefined,
      to: filters.to || undefined,
    }),
    [filters],
  );

  // One reload path for everything: any mutation refetches the whole view, so
  // derived figures like balances always come from the server rather than being
  // patched up locally and drifting out of step.
  const reload = useCallback(async () => {
    setError(null);
    try {
      const [nextAccounts, nextTransactions, nextSummary] = await Promise.all([
        api.listAccounts(),
        api.listTransactions(query),
        api.summary(query),
      ]);
      setAccounts(nextAccounts);
      setTransactions(nextTransactions);
      setSummary(nextSummary);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => {
    void reload();
  }, [reload]);

  // Categories come from the data itself rather than a fixed list, so the filter
  // only ever offers categories that actually exist.
  const categories = useMemo(() => {
    const seen = new Set(transactions.map((t) => t.category));
    for (const c of summary?.byCategory ?? []) seen.add(c.category);
    return [...seen].sort((a, b) => a.localeCompare(b));
  }, [transactions, summary]);

  const addAccount = async (a: NewAccount) => {
    await api.createAccount(a);
    await reload();
  };

  const addTransaction = async (t: NewTransaction) => {
    await api.createTransaction(t);
    await reload();
  };

  const removeAccount = async (id: string) => {
    try {
      await api.deleteAccount(id);
      // A filter pinned to the account just deleted would show an empty list
      // with no obvious cause, so clear it.
      if (filters.accountId === id) {
        setFilters((f) => ({ ...f, accountId: "" }));
      }
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not delete account.");
    }
  };

  const removeTransaction = async (id: string) => {
    try {
      await api.deleteTransaction(id);
      await reload();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not delete transaction.",
      );
    }
  };

  return (
    <div className="app">
      <header>
        <h1>finance_app</h1>
        <p className="hint">
          Go API · TypeScript frontend · in-memory store (data resets when the
          server restarts)
        </p>
      </header>

      {error && (
        <div className="banner error" role="alert">
          {error}
          <button type="button" className="link" onClick={() => void reload()}>
            Retry
          </button>
        </div>
      )}

      {loading ? (
        <p className="empty">Loading…</p>
      ) : (
        <main>
          <SummaryPanel summary={summary} />

          <Filters
            value={filters}
            accounts={accounts}
            categories={categories}
            onChange={setFilters}
          />

          <div className="columns">
            <AccountForm onSubmit={addAccount} />
            <TransactionForm accounts={accounts} onSubmit={addTransaction} />
          </div>

          <AccountList accounts={accounts} onDelete={removeAccount} />

          <TransactionList
            transactions={transactions}
            accounts={accounts}
            onDelete={removeTransaction}
          />
        </main>
      )}
    </div>
  );
}
