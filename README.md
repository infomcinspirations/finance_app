# finance_app

A personal finance tracker: accounts, transactions, and period summaries.
Go API, TypeScript frontend, no third-party Go dependencies.

> **Status: scaffold.** None of this has been compiled or run — Go, Node and
> Docker were all absent from the machine it was written on, and there is no CI.
> Your first step should be `make docker-test` (needs only Docker) or
> `make setup && make test` (needs Go and Node). Expect to fix what it finds.

## Layout

```
backend/                 Go API (standard library only)
  cmd/server/            main: config, signals, graceful shutdown
  internal/model/        domain types, normalization, validation
  internal/store/        Store interface + in-memory implementation
  internal/api/          routes, handlers, middleware
  internal/web/          embeds and serves the built frontend
frontend/                Vite + React + TypeScript
  src/api.ts             typed API client
  src/money.ts           cents parsing and formatting
  src/components/        forms, lists, summary panel
Dockerfile               builds both halves into one static binary
```

## Quick start

With Docker, and nothing else installed:

```bash
make docker-run
```

Then open <http://localhost:8080>. See [Docker](#docker) for details.

### Or run it locally

Needs Go 1.22+ and Node 20+.

```bash
make setup
```

Then run the two halves in separate terminals:

```bash
make dev-backend
```

```bash
make dev-frontend
```

Open <http://localhost:5173>. The Vite dev server proxies `/api` to the Go
server on `:8080`, so the browser makes same-origin requests.

`make help` lists every target.

## Money is stored in integer cents

Every monetary value in the API and the database is an **integer count of minor
units** — `-1250` means $12.50 out, `250000` means $2,500.00 in. Floating point
is never used for money, because `0.1 + 0.2 !== 0.3` in binary floating point
and a ledger that cannot add up exactly is not a ledger.

Conversion happens only at the edges, in
[`frontend/src/money.ts`](frontend/src/money.ts): `parseAmountToCents` turns
`"12.50"` into `1250`, and `formatCents` turns it back for display.

Transaction amounts are **signed**: negative is an outflow, positive an inflow.
The UI takes a positive figure plus a direction and derives the sign, so the
store never has to guess.

## API

Base path `/api/v1`. All requests and responses are JSON. Unknown request
fields are rejected, so a typo fails loudly instead of being ignored.

| Method   | Path                 | Notes                                            |
| -------- | -------------------- | ------------------------------------------------ |
| `GET`    | `/health`            | Liveness check                                   |
| `GET`    | `/accounts`          | Accounts with computed `balance`                 |
| `POST`   | `/accounts`          | Create an account                                |
| `GET`    | `/accounts/{id}`     | One account with computed `balance`              |
| `DELETE` | `/accounts/{id}`     | Deletes the account **and its transactions**     |
| `GET`    | `/transactions`      | Filter by `accountId`, `category`, `from`, `to`  |
| `POST`   | `/transactions`      | Create a transaction                             |
| `GET`    | `/transactions/{id}` | One transaction                                  |
| `DELETE` | `/transactions/{id}` | Delete a transaction                             |
| `GET`    | `/summary`           | Period flows and point-in-time balances          |

`from` and `to` accept either `YYYY-MM-DD` or a full RFC 3339 timestamp. A bare
`to` date covers the whole of that day.

Errors use one shape throughout:

```json
{ "error": "validation failed: amount must be non-zero" }
```

Status codes: `400` validation or malformed body, `404` missing record, `409`
referential conflict (a transaction naming an account that does not exist),
`500` anything unexpected (logged server-side, never echoed to the client).

### Try it

```bash
curl -s localhost:8080/api/v1/accounts \
  -H 'Content-Type: application/json' \
  -d '{"name":"Everyday","type":"checking","currency":"USD","openingBalance":250000}'
```

```bash
curl -s "localhost:8080/api/v1/summary?from=2026-09-01&to=2026-09-30"
```

### Balances vs. flows

`/summary` keeps two ideas apart, which is the subtlety most easily got wrong:

- **Flows** (`inflow`, `outflow`, `net`, `byCategory`) cover only transactions
  inside the `from`–`to` window.
- **Balances** (`totalBalance`, `byAccount[].balance`) are point-in-time: they
  include every transaction up to `to` and ignore `from`, because a balance is
  not a flow.

### Multiple currencies

No exchange rates are applied anywhere. When accounts span more than one
currency, `/summary` sets `mixedCurrency: true` and the UI omits cross-currency
totals rather than showing a number that adds euros to dollars.

## Docker

The whole application ships as **one image running one process**. The frontend is
compiled to static files, embedded into the Go binary, and served by it — so
there is no nginx, no second container and no Compose file.

```bash
make docker-run
```

That builds the image and serves it on <http://localhost:8080>. Override the host
port with `make docker-run PORT=3000`.

Or by hand:

```bash
docker build -t finance_app .
```

```bash
docker run --rm -p 8080:8080 finance_app
```

### Running the tests without installing anything

```bash
make docker-test
```

That builds the `test` stage, which runs `go vet` and the full Go suite. The
frontend's `tsc --noEmit` runs earlier, as part of `npm run build`, so a type
error fails the build before the tests are reached. This is the quickest way to
find out whether the scaffold actually compiles.

### How the image is built

Three stages, and the toolchains never reach the runtime layer:

1. `node:20-alpine` runs `npm run build` — that is `tsc --noEmit && vite build`,
   so a type error fails the image rather than shipping a broken bundle.
2. `golang:1.22-alpine` copies that bundle into `internal/web/dist` and compiles
   a static binary (`CGO_ENABLED=0`, `-trimpath`, symbols stripped).
3. `gcr.io/distroless/static-debian12:nonroot` receives the binary and nothing
   else. No shell, no package manager, no libc, and it runs as a non-root user.

The final image is a single executable, which is why `make docker-run` can add
`--read-only --cap-drop ALL --security-opt no-new-privileges` without anything
breaking: the process writes nothing and needs no capabilities.

Because there is no shell or `curl` to health-check with, the binary probes
itself — `HEALTHCHECK` runs `/server -health`, which requests its own health
endpoint and exits 0 or 1.

### How one binary serves both

`internal/web` embeds `dist/` with `//go:embed` and serves it on every path the
API does not claim:

- `/api/v1/...` — the API.
- Anything else under `/api/` — a JSON 404. An unknown endpoint is a client bug,
  and handing it the HTML shell would make a broken call look like a successful
  page load.
- Everything else — the frontend, with unknown paths falling through to
  `index.html` so a hard refresh on a client-side route returns the app rather
  than a 404.

Fingerprinted files under `assets/` are served `immutable` with a one-year
lifetime; `index.html` is always `no-cache`, because a stale shell points the
browser at asset URLs that no longer exist after a deploy.

`internal/web/dist/` holds a **committed placeholder page**, which exists so that
`go build` works in a bare checkout — `//go:embed` is resolved at compile time
and fails outright if the directory is missing. The Docker build replaces it with
the real bundle. A binary built locally without that step serves the placeholder,
which says so and points you at the dev server.

### CORS

In the container there is a single origin, so CORS never applies. It matters only
for local development, where Vite serves on `:5173` and the API on `:8080` —
and even then the dev server proxies `/api`, keeping requests same-origin.

### A note on the data

The store is in memory and the container has no volumes, so **everything is lost
when it stops**. That is not an oversight to fix with a volume; it needs a real
database behind `store.Store`. See [Known gaps](#known-gaps).

## Configuration

Backend, via environment variables:

| Variable               | Default                 | Meaning                            |
| ---------------------- | ----------------------- | ---------------------------------- |
| `ADDR`                 | `:8080`                 | Listen address                     |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated exact origins      |
| `LOG_LEVEL`            | `info`                  | `debug`, `info`, `warn`, `error`   |

CORS is an explicit allowlist rather than `*`, since these endpoints carry
personal financial data.

Frontend: copy `frontend/.env.example` to `frontend/.env` to point the dev proxy
somewhere other than `localhost:8080`.

Docker: pass `VERSION` as a build argument to stamp it into the binary, where
`/server -version` reports it. The Makefile exposes `IMAGE`, `VERSION` and
`PORT`.

## Testing

```bash
make test
```

Go tests run with `-race`. They cover the balance calculation, the cascade from
deleting an account, filter behaviour, validation rejections, the CORS
allowlist, and the summary's balance/flow split.

## Known gaps

The store is **in memory** — everything is lost when the server restarts. That
is deliberate for a scaffold: `store.Store` is a small interface, so a SQL
implementation drops in beside `store.Memory` without touching a handler.

Also not built yet:

- Persistence (Postgres or SQLite behind `store.Store`)
- Authentication — the API is currently wide open to anyone who can reach it
- Updating records; only create, read and delete exist
- Transfers between accounts as a first-class object, rather than two
  transactions
- Budgets, recurring transactions, and CSV or OFX import
- Pagination on the transaction list

## Pushing to GitHub

`backend/go.mod` declares `github.com/infomcinspirations/finance_app/backend`, guessed
from the local username. Since that path appears in every import, the helper
script rewrites it, sets the remote and pushes in one go:

```bash
./scripts/push-with-token.sh        # token auth; resolves your username for you
./scripts/push-to-github.sh NAME   # SSH or an existing credential helper
```

Both need you to be able to authenticate to GitHub first — see below.

### First-time authentication

This machine has no SSH key, no stored GitHub credential, and no `gh` CLI.
Pick one.

### Option 1: a personal access token (no extra tooling)

Fill in the token file, which lives outside the repo so it cannot be committed:

```bash
open -e ~/.config/finance_app/github.env
```

Then:

```bash
./scripts/push-with-token.sh
```

Add `--create` if the repository does not exist yet; it is created **private**
unless you also pass `--public`.

The script verifies the token, resolves your username from it, rewrites the Go
module path to match, checks that the rewrite still builds, and pushes. The
token is kept out of `argv` (curl reads the auth header from stdin), out of
`.git/config` (a one-shot credential helper supplies it), and out of all script
output.

A fine-grained token needs **Contents: Read and write** on `finance_app` and
nothing else. Creating a repository with `--create` requires a *classic* token
with the `repo` scope, because fine-grained tokens cannot create repositories.

When you are done, delete the file with `rm ~/.config/finance_app/github.env`
and revoke the token at <https://github.com/settings/tokens>.

### Option 2: an SSH key

```bash
ssh-keygen -t ed25519 -C "your-email@example.com" && cat ~/.ssh/id_ed25519.pub
```

Add that public key at <https://github.com/settings/keys>, confirm with
`ssh -T git@github.com`, then run the push script.

### Option 3: the GitHub CLI

Also gives you `gh repo create` and `gh pr`:

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" && brew install gh && gh auth login
```

## License

MIT — see [LICENSE](LICENSE). Replace the copyright holder with your own name.
