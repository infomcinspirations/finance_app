import { useState, type FormEvent } from "react";

import { parseAmountToCents } from "../money";
import { ACCOUNT_TYPES, type AccountType, type NewAccount } from "../types";

interface Props {
  onSubmit: (a: NewAccount) => Promise<void>;
}

export function AccountForm({ onSubmit }: Props) {
  const [name, setName] = useState("");
  const [type, setType] = useState<AccountType>("checking");
  const [currency, setCurrency] = useState("USD");
  const [opening, setOpening] = useState("0.00");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    if (!name.trim()) {
      setError("Give the account a name.");
      return;
    }
    // An opening balance of zero is legitimate, so only reject unparseable text.
    const cents = parseAmountToCents(opening);
    if (cents === null) {
      setError("Opening balance must be a number with at most two decimals.");
      return;
    }

    setBusy(true);
    try {
      await onSubmit({
        name: name.trim(),
        type,
        currency: currency.trim().toUpperCase(),
        openingBalance: cents,
      });
      setName("");
      setOpening("0.00");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add the account.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="card form" onSubmit={handleSubmit}>
      <h2>Add account</h2>

      <label>
        Name
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Everyday checking"
          maxLength={120}
        />
      </label>

      <div className="row">
        <label>
          Type
          <select
            value={type}
            onChange={(e) => setType(e.target.value as AccountType)}
          >
            {ACCOUNT_TYPES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </label>

        <label>
          Currency
          <input
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
            maxLength={3}
            size={4}
          />
        </label>

        <label>
          Opening balance
          <input
            value={opening}
            onChange={(e) => setOpening(e.target.value)}
            inputMode="decimal"
          />
        </label>
      </div>

      {error && <p className="error">{error}</p>}

      <button type="submit" disabled={busy}>
        {busy ? "Adding…" : "Add account"}
      </button>
    </form>
  );
}
