# syntax=docker/dockerfile:1

# Multi-stage: the toolchain stays in the build stage, and the final image holds
# nothing but a static binary. No shell, no package manager, no libc.

ARG GO_VERSION=1.22

# --- build --------------------------------------------------------------------

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

# go.mod is copied on its own so dependency resolution is cached independently of
# the source: editing a .go file does not invalidate it.
COPY go.mod ./
RUN go mod download

COPY . .

ARG VERSION=dev

# CGO_ENABLED=0 makes the binary static so it runs on a base with no libc.
# -trimpath strips local filesystem paths; -s -w drop the symbol table and DWARF.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/server ./cmd/server

# --- test ---------------------------------------------------------------------

# A separate stage so `docker build --target test` can run the suite without
# every image build depending on it.
FROM build AS test
RUN go vet ./... && go test ./...

# --- runtime ------------------------------------------------------------------

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=build /out/server /server

ENV ADDR=:8080 \
    LOG_LEVEL=info

EXPOSE 8080
USER nonroot:nonroot

# The image has no shell and no curl, so the binary probes itself.
HEALTHCHECK --interval=30s --timeout=3s --start-period=3s --retries=3 \
    CMD ["/server", "-health"]

ENTRYPOINT ["/server"]
