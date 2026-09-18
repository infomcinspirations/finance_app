import { formatCents } from "../money";
import type { Summary } from "../types";

interface Props {
  summary: Summary | null;
}

export function SummaryPanel({ summary }: Props) {
  if (!summary) {
    return (
      <section className="card">
        <h2>Summary</h2>
        <p className="empty">Nothing to summarise yet.</p>
      </section>
    );
  }

  // With several currencies in the book, adding minor units together produces a
  // number that means nothing. Show the parts and say why the total is absent.
  const currency = summary.currencies[0] ?? "USD";
  const showTotals = !summary.mixedCurrency;

  return (
    <section className="card">
      <h2>Summary</h2>

      {summary.mixedCurrency && (
        <p className="warn">
          Accounts span {summary.currencies.join(", ")}. Totals across
          currencies are omitted — no exchange rates are applied anywhere in
          this app.
        </p>
      )}

      <div className="stats">
        <Stat
          label="Net worth"
          value={showTotals ? formatCents(summary.totalBalance, currency) : "—"}
        />
        <Stat
          label="Money in"
          value={showTotals ? formatCents(summary.inflow, currency) : "—"}
          tone="positive"
        />
        <Stat
          label="Money out"
          value={showTotals ? formatCents(summary.outflow, currency) : "—"}
          tone="negative"
        />
        <Stat
          label="Net"
          value={showTotals ? formatCents(summary.net, currency) : "—"}
          tone={summary.net < 0 ? "negative" : "positive"}
        />
      </div>

      {summary.byCategory.length > 0 && (
        <>
          <h3>By category</h3>
          <table>
            <thead>
              <tr>
                <th>Category</th>
                <th className="num">Out</th>
                <th className="num">In</th>
                <th className="num">Net</th>
              </tr>
            </thead>
            <tbody>
              {summary.byCategory.map((c) => (
                <tr key={c.category}>
                  <td>{c.category}</td>
                  <td className="num negative">
                    {c.outflow > 0 ? formatCents(c.outflow, currency) : "—"}
                  </td>
                  <td className="num positive">
                    {c.inflow > 0 ? formatCents(c.inflow, currency) : "—"}
                  </td>
                  <td className={`num ${c.net < 0 ? "negative" : "positive"}`}>
                    {formatCents(c.net, currency)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </section>
  );
}

interface StatProps {
  label: string;
  value: string;
  tone?: "positive" | "negative";
}

function Stat({ label, value, tone }: StatProps) {
  return (
    <div className="stat">
      <span className="stat-label">{label}</span>
      <span className={`stat-value ${tone ?? ""}`}>{value}</span>
    </div>
  );
}
