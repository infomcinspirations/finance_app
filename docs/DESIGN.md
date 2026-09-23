# finance_app: feature design against public applications

A gap analysis of finance_app against the personal finance applications people
actually use, and a design for closing it.

**Status of the thing being compared.** finance_app is a scaffold, and an
unverified one: as of this writing no part of it has been compiled or run. The
comparison below is against what the code *specifies*, not against observed
behaviour. Treat every "has" in the tables as "has, on paper".

---

## 1. What exists today

Ten endpoints, two entities, one report.

| Entity | Fields |
| --- | --- |
| `Account` | id, name, type, currency, openingBalance, createdAt |
| `Transaction` | id, accountId, date, payee, category, amount, note, createdAt |

```
GET    /api/v1/health
GET    /api/v1/accounts            POST /api/v1/accounts
GET    /api/v1/accounts/{id}       DELETE /api/v1/accounts/{id}
GET    /api/v1/transactions        POST /api/v1/transactions
GET    /api/v1/transactions/{id}   DELETE /api/v1/transactions/{id}
GET    /api/v1/summary
```

What it gets right, and should not be traded away:

- **Money is integer minor units throughout.** No float ever touches a balance.
- **Balances are derived, never stored.** `openingBalance + Σ transactions`, so a
  balance cannot silently disagree with the transactions under it.
- **Flows and balances are kept distinct** in `/summary`: flows respect the
  date window, balances include everything up to `to`.
- **Mixed currencies are refused rather than faked.** No exchange rates exist, so
  cross-currency totals are omitted and flagged instead of adding euros to
  dollars.

Three properties of the current model matter more than any missing feature, and
Section 4 is about them:

1. **There is no update operation.** Not for accounts, not for transactions. The
   only way to fix a typo is delete and re-create, which changes the ID.
2. **There are no transfers.** Moving £500 from checking to savings is two
   unrelated transactions. Nothing links them, nothing stops one being deleted
   alone, and both land in category totals as if money left the household.
3. **Categories are free text.** `"Groceries"`, `"groceries"` and `"Grocery"` are
   three categories. Nothing can be budgeted against a string typed twice.

---

## 2. The comparison set

Chosen because they bracket the design space, not because they are the most
popular:

| App | Model | Why it is here |
| --- | --- | --- |
| [**Actual Budget**](https://actualbudget.org/docs/tour/) | Open source, local-first, envelope | The closest architectural sibling; rules, schedules, reconciliation |
| [**Firefly III**](https://docs.firefly-iii.org/explanation/firefly-iii/about/introduction/) | Open source, self-hosted, double-entry | The most rigorous data model in the category |
| [**YNAB**](https://www.ynab.com/blog/ynab-vs-monarch) | Commercial, zero-based envelope | The reference implementation of envelope budgeting |
| [**Monarch**](https://www.monarch.com/compare/ynab-alternative) | Commercial, aggregation-first | Net worth, investments, household sharing |
| [**Copilot**](https://wallethub.com/edu/b/ynab-vs-monarch-vs-copilot-vs-wallethub/150687) | Commercial, Apple-native | Automatic categorisation, daily spending awareness |

Two structural facts about this set are worth stating before the matrix:

**Firefly III is double-entry, and says so as a feature.** Every transaction
moves money from one account to another, which is why a transfer is one entry
rather than two guesses, and why its reports reconcile by construction.

**Bank aggregation is the commercial moat, not a feature.** Monarch's 13,000+
institutions exist because it pays an aggregator. The open-source options mostly
do not have it, and compete on import quality and rules instead. This is the one
gap finance_app should not try to close — see Section 7.

---

## 3. Gap matrix

`✓` shipped · `~` partial · `✗` absent

| Capability | finance_app | Actual | Firefly III | YNAB | Monarch |
| --- | :-: | :-: | :-: | :-: | :-: |
| **Records** | | | | | |
| Accounts, balances | ✓ | ✓ | ✓ | ✓ | ✓ |
| Create transactions | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Edit** a transaction | ✗ | ✓ | ✓ | ✓ | ✓ |
| Transfers as one record | ✗ | ✓ | ✓ | ✓ | ✓ |
| Split transactions | ✗ | ✓ | ✓ | ✓ | ✓ |
| Structured categories | ✗ | ✓ | ✓ | ✓ | ✓ |
| Payees as entities | ✗ | ✓ | ✓ | ✓ | ✓ |
| Tags / labels | ✗ | ~ | ✓ | ~ | ✓ |
| Attachments (receipts) | ✗ | ✗ | ✓ | ✗ | ✓ |
| **Budgeting** | | | | | |
| Envelope budgeting | ✗ | ✓ | ✓ | ✓ | ~ |
| Category spending limits | ✗ | ✓ | ✓ | ✓ | ✓ |
| Rollover of unspent money | ✗ | ✓ | ✓ | ✓ | ~ |
| Goals / savings targets | ✗ | ✓ | ✓ | ✓ | ✓ |
| **Automation** | | | | | |
| Categorisation rules | ✗ | ✓ | ✓ | ~ | ✓ |
| Recurring transactions | ✗ | ✓ | ✓ | ✓ | ✓ |
| Bill due-date tracking | ✗ | ✓ | ✓ | ✓ | ✓ |
| Bank sync | ✗ | ~ | ~ | ✓ | ✓ |
| CSV / OFX / QFX import | ✗ | ✓ | ✓ | ✓ | ✓ |
| Import de-duplication | ✗ | ✓ | ✓ | ✓ | ✓ |
| **Reporting** | | | | | |
| Period summary | ✓ | ✓ | ✓ | ✓ | ✓ |
| Spending by category | ✓ | ✓ | ✓ | ✓ | ✓ |
| Net worth over time | ✗ | ~ | ✓ | ~ | ✓ |
| Cash-flow / trend charts | ✗ | ✓ | ✓ | ✓ | ✓ |
| Reconciliation against statement | ✗ | ✓ | ✓ | ✓ | ~ |
| Investment holdings | ✗ | ✗ | ~ | ~ | ✓ |
| **Platform** | | | | | |
| Persistence | ✗ | ✓ | ✓ | ✓ | ✓ |
| Authentication | ✗ | ✓ | ✓ | ✓ | ✓ |
| Multi-user / household | ✗ | ~ | ~ | ✓ | ✓ |
| Multi-currency with rates | ✗ | ~ | ✓ | ✗ | ~ |
| Pagination on listings | ✗ | ✓ | ✓ | ✓ | ✓ |
| Full-text search | ✗ | ✓ | ✓ | ✓ | ✓ |
| Export your own data | ✗ | ✓ | ✓ | ✓ | ✓ |

Counting is the wrong way to read this table. Four of these absences are
load-bearing, in that every other row depends on them: persistence, editing,
transfers, and structured categories. The rest is ordinary work.

---

## 4. Four decisions to settle before writing any of it

These are the ones that are expensive to reverse, because they change data
already on disk.

### 4.1 Transfers: make transactions balanced posting sets

**The problem.** A transaction belongs to exactly one account and carries a
signed amount. A transfer is therefore two records with nothing joining them.
Delete one side and the household gains or loses money. Both sides land in
category spending, so moving savings around looks like expenditure.

**Recommendation.** Give a transaction a *set of postings* and a type:

```go
// Transaction is one financial event. Its postings say which accounts moved.
type Transaction struct {
    ID          string
    Type        TransactionType // withdrawal | deposit | transfer
    Date        time.Time
    Description string
    PayeeID     *string
    Postings    []Posting // exactly 2 for a transfer; 1+ otherwise
    TagIDs      []string
    Notes       string
    ImportID    *string    // dedupe key; unique per account when present
    ReconciledAt *time.Time
    CreatedAt, UpdatedAt time.Time
}

// Posting is one account's side of a transaction, in that account's currency.
type Posting struct {
    ID         string
    AccountID  string
    Amount     int64   // signed minor units
    CategoryID *string // nil on transfers: moving money is not spending
    // Set when the posting's currency differs from the transaction's, so the
    // rate used at the time is preserved rather than recomputed later.
    ForeignAmount   *int64
    ForeignCurrency *string
}
```

Invariants the store enforces, not the handlers:

- `transfer` has exactly two postings, against different accounts, summing to
  zero in a common currency, and **no category on either**.
- `withdrawal` has one or more postings, all negative, each with a category.
  More than one posting is a split.
- `deposit` is the same with positive amounts.
- A posting cannot exist without its transaction; deleting the transaction
  deletes every posting atomically.

**Why not full double-entry with categories as accounts** (the Firefly III and
GnuCash model, where groceries is an expense account and every transaction sums
to zero without exception): it is more rigorous and its reports reconcile by
construction. It is not recommended here because envelope budgeting needs
categories to carry monthly allocations and rollover, which is a different
lifecycle from an account, and because it forces the concept on the UI. The
posting-set model above fixes the actual defect — unlinked transfers — at a
fraction of the conceptual cost. Revisit if strict bookkeeping becomes a goal.

**Migration cost if deferred:** every transaction row, plus a guess at which
pairs were meant to be transfers. Do it before there is data worth keeping.

### 4.2 Scope every row to an owner from the first migration

Even for a single-user deployment, give every table a `household_id` on day one
and put it in every query and index. Retrofitting a tenancy column means
rewriting every query and every index, under the one condition where mistakes
leak one person's finances to another. The cost today is a column. The cost later
is an audit.

### 4.3 Categories become entities, and the budget hangs off them

A category needs an ID, a parent, an archived flag, and a stable identity that
survives renaming. Free-text categories cannot be budgeted, cannot be renamed
without rewriting history, and cannot roll up.

```go
type Category struct {
    ID       string
    Name     string
    ParentID *string // one level of nesting is enough; two is a taxonomy project
    Kind     CategoryKind // expense | income
    Archived bool // hidden from pickers, still valid in history
}
```

Archived rather than deleted, because deleting a category orphans historical
transactions and silently changes last year's reports.

### 4.4 Choose the budgeting model deliberately

The two models in the comparison set are genuinely different products:

| | Envelope (YNAB, Actual) | Limits (Monarch, Copilot) |
| --- | --- | --- |
| Question answered | "What may I spend this money on?" | "Did I overspend?" |
| Needs | Money available now, per-category allocation, rollover, overspend handling | A number per category per month |
| Effort for the user | High — allocate every month | Low — set once |
| Effort to build | High | Low |

**Recommendation: build limits first, design the schema for envelopes.** A
`BudgetPeriod` row per `(category, month)` holding an `allocated` amount serves
both: limits read it as a cap, envelopes additionally compute
`available = allocated + rolloverIn − spent`. Shipping limits first gets a useful
feature out without foreclosing the harder one.

---

## 5. Designed feature set

Grouped by what has to exist before the next thing can.

### Layer 0 — Make it real

Nothing in this document matters while the store is a map that empties on
restart.

| Item | Design |
| --- | --- |
| **Persistence** | SQLite via `database/sql` behind the existing `store.Store`. Single file, no server, honest for one household, and PostgreSQL later needs no interface change. Schema in numbered, forward-only migration files applied at startup. |
| **Editing** | `PATCH /accounts/{id}`, `PATCH /transactions/{id}`. PATCH not PUT: a partial update must not silently blank a field the client did not send. Needs a nullable-field-aware decoder — distinguishing "absent" from "null" is the whole difficulty. |
| **Optimistic concurrency** | `ETag` on single-resource GETs, `If-Match` required on PATCH and DELETE. Two people editing the same transaction on two phones is the ordinary case, not the exotic one. |
| **Pagination** | Cursor-based on `(date, id)`, returning `{ items, nextCursor }`. Not offset: rows are inserted mid-list constantly, and offset pagination skips and repeats rows when that happens. |
| **Authentication** | Sessions in an HTTP-only, `Secure`, `SameSite=Lax` cookie; Argon2id password hashes. Not JWTs: logout has to actually revoke, and a session row does that while a signed token does not. |
| **Audit trail** | Append-only `audit_log` of who changed what and when. Financial records that change without a trace are not records. |

### Layer 1 — Records worth keeping

| Item | Design |
| --- | --- |
| **Payees** | `Payee{id, name, aliases[]}`. Aliases are what makes rules and import matching possible: `AMZN MKTP US*2K4L` and `Amazon` are one payee. Merging two payees rewrites references in one transaction. |
| **Splits** | Falls out of Section 4.1 for free: a withdrawal with several postings. Needs UI, not schema. |
| **Tags** | `Tag{id, name}` many-to-many with transactions. Orthogonal to categories: a category answers *what kind of spending*, a tag answers *which trip, which project, which reimbursement*. |
| **Attachments** | Content-addressed blobs (`sha256`) on disk with a metadata row, so the same receipt attached twice is stored once. Cap size, validate the declared type, and serve with `Content-Disposition: attachment` — never inline, since an HTML attachment served inline is stored XSS against the person's own ledger. |
| **Reconciliation** | `ReconciledAt` per transaction plus a session: enter the statement's closing balance and date, the server returns the difference against its own computed balance and the unreconciled transactions in the window. Clearing a statement is the moment errors are actually found, which is why every app in the set has this and it is not optional. |
| **Search** | SQLite FTS5 over description, payee, notes and tags. |

### Layer 2 — Budgeting

| Item | Design |
| --- | --- |
| **Periods** | `BudgetPeriod{householdID, categoryID, month, allocatedMinor}`. Month as `YYYY-MM`, not a timestamp: a budget month is a label, and storing it as an instant invites a timezone to move it. |
| **Limits** | `GET /budgets/{month}` returns per category: allocated, spent, remaining, and `%` used. This alone matches Monarch and Copilot. |
| **Envelopes** | Adds `available = allocated + rolloverIn − spent`, and `toBudget` = funds not yet assigned. Overspend must be an explicit policy — carry the negative into next month, or absorb it from unassigned — because leaving it implicit is how envelope implementations go wrong. |
| **Goals** | `Goal{name, targetMinor, targetDate, linkedAccountID?, categoryID?}`, with progress derived from balance or contributions. Equivalent to Firefly III's piggy banks. |

### Layer 3 — Automation

| Item | Design |
| --- | --- |
| **Rules** | `Rule{priority, conditions[], actions[], stopProcessing}`. Conditions over payee, description, amount range, account. Actions set category, add tags, rename payee. Two requirements the set treats as essential: rules run on import **and** can be re-run retroactively over history; and a dry-run endpoint shows what *would* change before it does. |
| **Recurring** | `Schedule{template, rrule, nextDue, autoPost}`. RFC 5545 recurrence rules rather than a homegrown cron, because "last weekday of the month" is already solved and solved badly by everyone who reinvents it. `autoPost=false` produces a suggestion to confirm; `true` writes it. |
| **Bills** | A schedule with an expected amount and a due date, plus a matcher that marks it paid when a transaction matches. The output is "what is due, and did it clear" — which is the actual daily question. |
| **Import** | `POST /imports` (CSV/OFX/QFX) → a **preview** with proposed mappings, detected duplicates and rule effects → `POST /imports/{id}/commit`. Never write on upload. De-duplicate on a stable `importID` per account (OFX `FITID`, or a hash of date+amount+description), which is why `Transaction.ImportID` is in the model above. |
| **Export** | `GET /export?format=csv|json` over everything. Non-negotiable: a finance app that holds data it will not give back is a trap, and every app in the comparison set exports. |

### Layer 4 — Reporting

| Item | Design |
| --- | --- |
| **Net worth series** | `GET /reports/net-worth?from&to&interval=month`. Assets minus liabilities per point. Needs `Account.Class` (asset/liability) — a `credit` account is a liability and must subtract, which the current `AccountType` does not encode. |
| **Cash flow** | Income vs expense per interval, transfers excluded. Excluding transfers is the whole point and is only possible once Section 4.1 exists. |
| **Category trend** | One category over time, to answer "is this getting worse". |
| **Multi-currency** | `ExchangeRate{base, quote, asOf, rate}`, with a household display currency. Convert **at transaction date, not at read time**, so last year's report does not change when today's rate moves. Until rates exist, keep refusing to total mixed currencies — the current behaviour is correct and better than a plausible wrong number. |
| **Investments** | `Holding{accountID, symbol, quantity, costBasis}` and optional `Lot`s. Deliberately last: it is the largest subsystem here and only Monarch does it properly. Manual quantities and prices first; price feeds are a separate project. |

---

## 6. Sequencing

Each step leaves the app working, and no step's schema is invalidated by a later
one.

| # | Step | Why here |
| --- | --- | --- |
| 1 | **Verify the scaffold compiles** (`make docker-test`) | Nine commits of unverified code. Everything below is speculation until this passes. |
| 2 | SQLite + migrations + household scope | Persistence, plus the tenancy column that is cheap now and expensive later (4.2) |
| 3 | Postings, transaction types, transfers (4.1) | Most expensive to retrofit; do it while the table is empty |
| 4 | Category and payee entities (4.3) | Prerequisite for budgets, rules and every report |
| 5 | PATCH, ETag/If-Match, pagination, search | Makes it usable on a real ledger rather than a demo one |
| 6 | Auth, sessions, audit log | Before it holds anything true about anyone |
| 7 | CSV/OFX import with preview and dedupe | The point at which entering a year of history stops being manual |
| 8 | Rules, with dry-run and retroactive re-run | Multiplies the value of step 7 |
| 9 | Budget periods and limits | First genuinely differentiating feature |
| 10 | Reconciliation | Where errors from steps 7–8 get caught |
| 11 | Net worth and cash-flow reports | Needs account class and transfers, both settled by now |
| 12 | Envelope budgeting, goals, recurring | The YNAB/Actual tier |
| 13 | Attachments, investments | Largest scope, least leverage |

Steps 2–6 are unglamorous and are the whole job. An app that cannot edit a
transaction, cannot page a listing, and forgets everything on restart is not
behind on features; it is not yet an application.

---

## 7. Deliberately not designed

| Not doing | Why |
| --- | --- |
| **Bank aggregation** | Requires a paid aggregator (Plaid, TrueLayer, GoCardless), per-institution credential handling, and the regulatory surface that comes with holding banking credentials. This is Monarch's moat and it is bought, not built. Excellent import plus rules gets most of the value at a fraction of the risk. |
| **Automatic categorisation by model** | Copilot's version is good because it is trained on a large labelled corpus. Deterministic rules the user can read and correct beat an opaque classifier on one household's data, and they fail comprehensibly. |
| **Tax preparation, forecasting, retirement projection** | Each is a product. Monarch's multi-year plan is a separate surface with its own assumptions, and getting it subtly wrong is worse than not having it. |
| **Mobile apps** | The frontend is responsive. Native apps are two more platforms to keep in step with an API that is still changing shape. |
| **Sharing, permissions, invitations** | Section 4.2 makes it possible later. Designing roles before a second user exists is speculation. |

---

## Sources

- [A Tour of Actual](https://actualbudget.org/docs/tour/) and [Envelope Budgeting](https://actualbudget.org/docs/getting-started/envelope-budgeting/)
- [Firefly III: Introduction and features](https://docs.firefly-iii.org/explanation/firefly-iii/about/introduction/)
- [Firefly III vs. Actual Budget](https://beancount.io/blog/2026/07/26/firefly-iii-vs-actual-budget-self-hosted-open-source-budgeting-guide)
- [YNAB vs. Monarch](https://www.ynab.com/blog/ynab-vs-monarch)
- [Monarch: YNAB alternative comparison](https://www.monarch.com/compare/ynab-alternative)
- [YNAB vs. Monarch vs. Copilot vs. WalletHub](https://wallethub.com/edu/b/ynab-vs-monarch-vs-copilot-vs-wallethub/150687)
