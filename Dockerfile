FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /mnemo ./cmd/mnemo

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /mnemo /mnemo

EXPOSE 8765

ENTRYPOINT ["/mnemo"]
