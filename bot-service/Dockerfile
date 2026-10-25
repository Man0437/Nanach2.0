FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .

RUN go mod tidy
RUN go build -o bot .

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/bot .

CMD ["./bot"]