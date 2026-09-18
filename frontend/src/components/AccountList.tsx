import { formatCents } from "../money";
import type { Account } from "../types";

interface Props {
  accounts: Account[];
  onDelete: (id: string) => Promise<void>;
}

export function AccountList({ accounts, onDelete }: Props) {
  if (accounts.length === 0) {
    return (
      <section className="card">
        <h2>Accounts</h2>
        <p className="empty">No accounts yet. Add one to get started.</p>
      </section>
    );
  }

  return (
    <section className="card">
      <h2>Accounts</h2>
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th className="num">Transactions</th>
            <th className="num">Balance</th>
            <th aria-label="Actions" />
          </tr>
        </thead>
        <tbody>
          {accounts.map((a) => (
            <tr key={a.id}>
              <td>{a.name}</td>
              <td className="muted">{a.type}</td>
              <td className="num muted">{a.transactionCount}</td>
              <td className={`num ${a.balance < 0 ? "negative" : ""}`}>
                {formatCents(a.balance, a.currency)}
              </td>
              <td>
                <button
                  type="button"
                  className="link danger"
                  onClick={() => {
                    // Deleting an account takes its transactions with it, so
                    // this one is worth confirming.
                    const ok = window.confirm(
                      `Delete "${a.name}" and its ${a.transactionCount} transaction(s)?`,
                    );
                    if (ok) void onDelete(a.id);
                  }}
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
