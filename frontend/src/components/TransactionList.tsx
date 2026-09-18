import { formatCents, formatDate } from "../money";
import type { Account, Transaction } from "../types";

interface Props {
  transactions: Transaction[];
  accounts: Account[];
  onDelete: (id: string) => Promise<void>;
}

export function TransactionList({ transactions, accounts, onDelete }: Props) {
  const currencyFor = new Map(accounts.map((a) => [a.id, a.currency]));
  const nameFor = new Map(accounts.map((a) => [a.id, a.name]));

  return (
    <section className="card">
      <h2>
        Transactions <span className="hint">{transactions.length}</span>
      </h2>

      {transactions.length === 0 ? (
        <p className="empty">Nothing matches the current filters.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Date</th>
              <th>Payee</th>
              <th>Category</th>
              <th>Account</th>
              <th className="num">Amount</th>
              <th aria-label="Actions" />
            </tr>
          </thead>
          <tbody>
            {transactions.map((t) => (
              <tr key={t.id}>
                <td className="muted">{formatDate(t.date)}</td>
                <td title={t.note ?? ""}>{t.payee}</td>
                <td className="muted">{t.category}</td>
                <td className="muted">{nameFor.get(t.accountId) ?? "—"}</td>
                <td className={`num ${t.amount < 0 ? "negative" : "positive"}`}>
                  {formatCents(t.amount, currencyFor.get(t.accountId) ?? "USD")}
                </td>
                <td>
                  <button
                    type="button"
                    className="link danger"
                    onClick={() => void onDelete(t.id)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
