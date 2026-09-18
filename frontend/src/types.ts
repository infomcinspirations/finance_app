// Mirrors the Go types in backend/internal/model and the API response shapes.
// Every monetary field is an integer count of minor units (cents for USD).

export type AccountType =
  | "checking"
  | "savings"
  | "credit"
  | "investment"
  | "cash";

export const ACCOUNT_TYPES: readonly AccountType[] = [
  "checking",
  "savings",
  "credit",
  "investment",
  "cash",
];

export interface Account {
  id: string;
  name: string;
  type: AccountType;
  currency: string;
  openingBalance: number;
  createdAt: string;
  /** Opening balance plus every transaction against this account. */
  balance: number;
  transactionCount: number;
}

export interface Transaction {
  id: string;
  accountId: string;
  date: string;
  payee: string;
  category: string;
  /** Signed minor units: negative is an outflow, positive an inflow. */
  amount: number;
  note?: string;
  createdAt: string;
}

export interface CategoryTotal {
  category: string;
  inflow: number;
  /** Positive magnitude of money out. */
  outflow: number;
  net: number;
  count: number;
}

export interface AccountTotal {
  accountId: string;
  name: string;
  currency: string;
  balance: number;
  net: number;
}

export interface Summary {
  from: string | null;
  to: string | null;
  currencies: string[];
  /** True when accounts span more than one currency, making totals unsafe. */
  mixedCurrency: boolean;
  totalBalance: number;
  inflow: number;
  outflow: number;
  net: number;
  transactionCount: number;
  byCategory: CategoryTotal[];
  byAccount: AccountTotal[];
}

export interface NewAccount {
  name: string;
  type: AccountType;
  currency: string;
  openingBalance: number;
}

export interface NewTransaction {
  accountId: string;
  date?: string;
  payee: string;
  category: string;
  amount: number;
  note?: string;
}
