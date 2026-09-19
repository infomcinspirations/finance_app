# syntax=docker/dockerfile:1

# One image, one process. The frontend is compiled to static files, embedded into
# the Go binary, and served by it — so there is no nginx, no second container and
# no internal network. The runtime image is a single static executable on a base
# with no shell, no package manager and no libc.

ARG GO_VERSION=1.22
ARG NODE_VERSION=20

# --- frontend -----------------------------------------------------------------

FROM node:${NODE_VERSION}-alpine AS frontend

WORKDIR /app

# Manifests first so dependency installation caches independently of the source.
COPY frontend/package.json frontend/package-lock.json* ./

# npm ci is reproducible but needs a lockfile; fall back so a checkout without
# one still builds.
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

COPY frontend/ ./

# `npm run build` is `tsc --noEmit && vite build`, so a type error fails the
# image build instead of shipping a broken bundle.
RUN npm run build

# --- backend ------------------------------------------------------------------

FROM golang:${GO_VERSION}-alpine AS backend

WORKDIR /src

COPY backend/go.mod ./
RUN go mod download

COPY backend/ ./

# internal/web/dist holds a committed placeholder page so `go build` works in a
# bare checkout. Swap in the real bundle before compiling, since //go:embed
# captures whatever is on disk at that moment.
RUN rm -rf internal/web/dist
COPY --from=frontend /app/dist/ ./internal/web/dist/

ARG VERSION=dev

# CGO_ENABLED=0 makes the binary static so it runs on a base with no libc.
# -trimpath strips local paths; -s -w drop the symbol table and DWARF.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/server ./cmd/server

# --- test ---------------------------------------------------------------------

# Its own stage, so `docker build --target test .` runs the suite without making
# every image build depend on it. It sits after the frontend copy, so the tests
# run against the same embedded assets the shipped binary has.
FROM backend AS test
RUN go vet ./... && go test ./...

# --- runtime ------------------------------------------------------------------

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=backend /out/server /server

ENV ADDR=:8080 \
    LOG_LEVEL=info

EXPOSE 8080
USER nonroot:nonroot

# No shell and no curl in this image, so the binary probes itself.
HEALTHCHECK --interval=30s --timeout=3s --start-period=3s --retries=3 \
    CMD ["/server", "-health"]

ENTRYPOINT ["/server"]
