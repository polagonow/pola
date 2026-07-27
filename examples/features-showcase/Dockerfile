# syntax=docker/dockerfile:1
#
# Single static binary for the "features-showcase" example, served from a scratch image.
# Runs against PostgreSQL (see docker-compose.yml). A scratch image cannot carry
# the cgo SQLite driver, so the container uses the pure-Go Postgres driver.
#
# Build context MUST be the repo root (replace github.com/polagonow/pola => ../..):
#   docker compose -f examples/features-showcase/docker-compose.yml up --build

FROM golang:1.26-alpine AS build

ENV GOFLAGS=-mod=mod

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY . .
RUN go build -o /usr/local/bin/pola ./cmd/pola

WORKDIR /src/examples/features-showcase
# API-only: no front-end bundling. Force the postgresql adapter so the pure-Go
# driver is compiled in and the binary runs on scratch.
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
