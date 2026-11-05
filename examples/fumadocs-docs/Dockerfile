# syntax=docker/dockerfile:1
#
# Single static binary for the "fumadocs-docs" example, served from a scratch image.

# Build context MUST be the repo root (replace github.com/polagonow/pola => ../..):
#   docker build -f examples/fumadocs-docs/Dockerfile -t fumadocs-docs .
#   docker run --rm -p 3000:3000 fumadocs-docs

FROM golang:1.26-alpine AS build

# Resolve any go.sum entries the example is missing from the full module graph.
ENV GOFLAGS=-mod=mod

# git/nodejs/npm – build the CLI and bundle the .tsx front-end (esbuild + pnpm)
RUN apk add --no-cache git nodejs npm ca-certificates tzdata \
 && npm install -g pnpm@10

# Pola release the docs are built for (e.g. v1.2.3); CI passes the release tag.
ARG POLA_VERSION=dev

WORKDIR /src
COPY . .
RUN go build -ldflags "-X github.com/polagonow/pola/internal/cli.version=${POLA_VERSION}" \
      -o /usr/local/bin/pola ./cmd/pola

# Bundle assets + compile a pure-Go static binary. Building the binary does not
# require a database connection (the DB connects lazily on first query).
WORKDIR /src/examples/fumadocs-docs
RUN pola build --cgo 0 -o /app

FROM scratch
ARG POLA_VERSION=dev
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /app /app

# pola listens on PORT (default 3000); POLA_ADDRESS overrides the bind host.
ENV PORT=3000
# Read by Server.getPolaVersion() to show the release in the docs nav.
ENV POLA_VERSION=${POLA_VERSION}
EXPOSE 3000
CMD ["/app"]
