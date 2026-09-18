import type { Account } from "../types";

export interface FilterState {
  accountId: string;
  category: string;
  from: string;
  to: string;
}

interface Props {
  value: FilterState;
  accounts: Account[];
  categories: string[];
  onChange: (next: FilterState) => void;
}

export function Filters({ value, accounts, categories, onChange }: Props) {
  function set(patch: Partial<FilterState>) {
    onChange({ ...value, ...patch });
  }

  const active =
    value.accountId !== "" ||
    value.category !== "" ||
    value.from !== "" ||
    value.to !== "";

  return (
    <section className="card form">
      <h2>Filters</h2>

      <div className="row">
        <label>
          Account
          <select
            value={value.accountId}
            onChange={(e) => set({ accountId: e.target.value })}
          >
            <option value="">All accounts</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </label>

        <label>
          Category
          <select
            value={value.category}
            onChange={(e) => set({ category: e.target.value })}
          >
            <option value="">All categories</option>
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>

        <label>
          From
          <input
            type="date"
            value={value.from}
            onChange={(e) => set({ from: e.target.value })}
          />
        </label>

        <label>
          To
          <input
            type="date"
            value={value.to}
            onChange={(e) => set({ to: e.target.value })}
          />
        </label>
      </div>

      {active && (
        <button
          type="button"
          className="link"
          onClick={() =>
            onChange({ accountId: "", category: "", from: "", to: "" })
          }
        >
          Clear filters
        </button>
      )}
    </section>
  );
}
