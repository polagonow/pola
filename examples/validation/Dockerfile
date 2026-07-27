# syntax=docker/dockerfile:1
#
# Single static binary for the "validation" example, served from a scratch image.
#
# The build context MUST be the repository root: the example resolves the
# framework through  replace github.com/polagonow/pola => ../..
#
#   docker build -f examples/validation/Dockerfile -t validation .
#   docker run --rm -p 3000:3000 validation

FROM golang:1.26-alpine AS build

# Resolve any go.sum entries the example is missing from the full module graph.
ENV GOFLAGS=-mod=mod

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY . .

WORKDIR /src/examples/validation
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /app .

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
