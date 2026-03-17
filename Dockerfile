FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -tags sqlite_fts5 -ldflags="-s -w -extldflags=-static" -o /mnemo ./cmd/mnemo

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /mnemo /mnemo

EXPOSE 8765

ENTRYPOINT ["/mnemo"]
