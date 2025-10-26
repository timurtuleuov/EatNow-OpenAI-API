# Этап сборки
FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Этап запуска
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/server .

# Можно задать переменные окружения (если хочешь)
ENV PORT=8080
EXPOSE 8080

CMD ["./server"]
