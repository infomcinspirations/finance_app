import type {
  Account,
  NewAccount,
  NewTransaction,
  Summary,
  Transaction,
} from "./types";

const BASE = "/api/v1";

/** An error carrying the HTTP status so callers can distinguish 404 from 500. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/** The error shape every failing endpoint returns. */
interface ErrorBody {
  error?: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // Headers is used rather than an object spread because init.headers may be a
  // Headers instance or an entry array, which do not spread into an object.
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let res: Response;
  try {
    res = await fetch(`${BASE}${path}`, { ...init, headers });
  } catch {
    // fetch only rejects on network-level failures, which almost always means
    // the Go server is not running.
    throw new ApiError(
      0,
      "Could not reach the API. Is the backend running on port 8080?",
    );
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();

  if (!res.ok) {
    let message = `Request failed with status ${res.status}`;
    try {
      const body = JSON.parse(text) as ErrorBody;
      if (body.error) message = body.error;
    } catch {
      if (text) message = text;
    }
    throw new ApiError(res.status, message);
  }

  return (text ? JSON.parse(text) : undefined) as T;
}

/** Drops empty values so we never send `?category=`. */
function query(params: Record<string, string | undefined>): string {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value) q.set(key, value);
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

// A type alias, not an interface: only aliases get the implicit index signature
// that lets this be passed to query(). The explicit `| undefined` is required
// under exactOptionalPropertyTypes, since callers do assign undefined.
export type TransactionQuery = {
  accountId?: string | undefined;
  category?: string | undefined;
  from?: string | undefined;
  to?: string | undefined;
};

export const api = {
  health: () => request<{ status: string }>("/health"),

  listAccounts: () => request<Account[]>("/accounts"),

  createAccount: (a: NewAccount) =>
    request<Account>("/accounts", { method: "POST", body: JSON.stringify(a) }),

  deleteAccount: (id: string) =>
    request<void>(`/accounts/${encodeURIComponent(id)}`, { method: "DELETE" }),

  listTransactions: (q: TransactionQuery = {}) =>
    request<Transaction[]>(`/transactions${query(q)}`),

  createTransaction: (t: NewTransaction) =>
    request<Transaction>("/transactions", {
      method: "POST",
      body: JSON.stringify(t),
    }),

  deleteTransaction: (id: string) =>
    request<void>(`/transactions/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),

  summary: (q: TransactionQuery = {}) => request<Summary>(`/summary${query(q)}`),
};
