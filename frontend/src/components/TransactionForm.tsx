import { useState, type FormEvent } from "react";

import { parseAmountToCents, todayISODate } from "../money";
import type { Account, NewTransaction } from "../types";

interface Props {
  accounts: Account[];
  onSubmit: (t: NewTransaction) => Promise<void>;
}

type Direction = "out" | "in";

export function TransactionForm({ accounts, onSubmit }: Props) {
  const [accountId, setAccountId] = useState("");
  const [direction, setDirection] = useState<Direction>("out");
  const [amount, setAmount] = useState("");
  const [payee, setPayee] = useState("");
  const [category, setCategory] = useState("");
  const [date, setDate] = useState(todayISODate);
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Fall back to the first account so the common case needs no extra click.
  const effectiveAccountId = accountId || accounts[0]?.id || "";

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    if (!effectiveAccountId) {
      setError("Add an account first.");
      return;
    }
    if (!payee.trim()) {
      setError("Who was it paid to, or received from?");
      return;
    }

    const cents = parseAmountToCents(amount);
    if (cents === null || cents === 0) {
      setError("Enter a non-zero amount, e.g. 12.50.");
      return;
    }
    // The form takes a positive figure plus a direction; the API wants a signed
    // amount. Deriving the sign here keeps the ambiguity out of the store.
    const magnitude = Math.abs(cents);
    const signed = direction === "out" ? -magnitude : magnitude;

    // A date input can be cleared, and new Date("") throws on toISOString.
    // Midday is used rather than midnight so a timezone shift cannot move the
    // transaction onto the previous day.
    const timestamp = new Date(`${date}T12:00:00`);
    if (Number.isNaN(timestamp.getTime())) {
      setError("Pick a valid date.");
      return;
    }

    setBusy(true);
    try {
      await onSubmit({
        accountId: effectiveAccountId,
        payee: payee.trim(),
        category: category.trim(),
        amount: signed,
        date: timestamp.toISOString(),
        note: note.trim(),
      });
      setAmount("");
      setPayee("");
      setNote("");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not add the transaction.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="card form" onSubmit={handleSubmit}>
      <h2>Add transaction</h2>

      <div className="row">
        <label>
          Account
          <select
            value={effectiveAccountId}
            onChange={(e) => setAccountId(e.target.value)}
            disabled={accounts.length === 0}
          >
            {accounts.length === 0 && <option value="">No accounts yet</option>}
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </label>

        <label>
          Direction
          <select
            value={direction}
            onChange={(e) => setDirection(e.target.value as Direction)}
          >
            <option value="out">Money out</option>
            <option value="in">Money in</option>
          </select>
        </label>

        <label>
          Amount
          <input
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="12.50"
            inputMode="decimal"
          />
        </label>
      </div>

      <div className="row">
        <label>
          Payee
          <input
            value={payee}
            onChange={(e) => setPayee(e.target.value)}
            placeholder="Corner Store"
            maxLength={200}
          />
        </label>

        <label>
          Category
          <input
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            placeholder="Groceries"
            maxLength={80}
          />
        </label>

        <label>
          Date
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
          />
        </label>
      </div>

      <label>
        Note <span className="hint">optional</span>
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          maxLength={1000}
        />
      </label>

      {error && <p className="error">{error}</p>}

      <button type="submit" disabled={busy || accounts.length === 0}>
        {busy ? "Adding…" : "Add transaction"}
      </button>
    </form>
  );
}
