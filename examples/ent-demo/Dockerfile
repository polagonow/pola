# syntax=docker/dockerfile:1
#
# Single static binary for the "ent-demo" example, served from a scratch image.
# Runs against PostgreSQL (see docker-compose.yml). A scratch image cannot carry
# the cgo SQLite driver, so the container uses the pure-Go Postgres driver.
#
# Build context MUST be the repo root (replace github.com/polagonow/pola => ../..):
#   docker compose -f examples/ent-demo/docker-compose.yml up --build

FROM golang:1.26-alpine AS build

ENV GOFLAGS=-mod=mod

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY . .
RUN go build -o /usr/local/bin/pola ./cmd/pola

# API-only: no front-end bundling, and building never requires a database
# connection. Force the postgresql adapter so the pure-Go driver is compiled in.
WORKDIR /src/examples/ent-demo
RUN POLA_DATABASE_ADAPTER=postgresql pola build --cgo 0 -o /app

FROM scratch
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /app /app
ENV PORT=3000
EXPOSE 3000
CMD ["/app"]
