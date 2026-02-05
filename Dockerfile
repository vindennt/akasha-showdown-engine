FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o engine ./cmd

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/engine .

EXPOSE 8282

CMD ["./engine"]