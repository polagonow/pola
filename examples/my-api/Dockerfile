# syntax=docker/dockerfile:1
#
# Single static binary for the "my-api" example, served from a scratch image.
#
# The build context MUST be the repository root: the example resolves the
# framework through  replace github.com/polagonow/pola => ../..
#
#   docker build -f examples/my-api/Dockerfile -t my-api .
#   docker run --rm -p 3000:3000 my-api

FROM golang:1.26-alpine AS build

# Resolve any go.sum entries the example is missing from the full module graph.
ENV GOFLAGS=-mod=mod

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY . .

RUN go build -o /usr/local/bin/pola ./cmd/pola

# API-only: wire plugins and compile a pure-Go static binary.
WORKDIR /src/examples/my-api
RUN pola build --cgo 0 -o /app

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
