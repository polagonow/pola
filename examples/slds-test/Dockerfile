# syntax=docker/dockerfile:1
#
# Single static binary for the "slds-test" example, served from a scratch image.
# Runs against PostgreSQL (see docker-compose.yml). A scratch image cannot carry
# the cgo SQLite driver, so the container uses the pure-Go Postgres driver.
#
# Build context MUST be the repo root (replace github.com/polagonow/pola => ../..):
#   docker build -f examples/slds-test/Dockerfile -t slds-test .
#   docker compose -f examples/slds-test/docker-compose.yml up --build

FROM golang:1.26-alpine AS build

# Resolve any go.sum entries the example is missing from the full module graph.
ENV GOFLAGS=-mod=mod

# git/nodejs/npm – build the CLI and bundle the .tsx front-end (esbuild + pnpm)
RUN apk add --no-cache git nodejs npm ca-certificates tzdata \
 && npm install -g pnpm@10

WORKDIR /src
COPY . .
RUN go build -o /usr/local/bin/pola ./cmd/pola

# Bundle assets + compile a pure-Go static binary. Building the binary does not
# require a database connection (the DB connects lazily on first query).
WORKDIR /src/examples/slds-test
RUN POLA_DATABASE_ADAPTER=postgresql pola build --cgo 0 -o /app

FROM scratch
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /app /app

# pola listens on PORT (default 3000); POLA_ADDRESS overrides the bind host.
ENV PORT=3000
EXPOSE 3000
CMD ["/app"]
