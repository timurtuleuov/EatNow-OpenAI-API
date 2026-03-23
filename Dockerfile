# Этап сборки
FROM golang:1.23 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Этап запуска

FROM alpine:latest


WORKDIR /app

# 1. Копируем бинарник из билдера
COPY --from=builder /app/server .
COPY images ./images

# 2. Создаем папку configs
RUN mkdir configs

# 3. ПРАВИЛЬНЫЙ ПУТЬ: копируем из папки configs билдера в папку configs финала
# Если файл на хосте лежит в configs/config.default.yaml, 
# то в билдере он тоже в /app/configs/
COPY --from=builder /app/configs/config.default.yaml ./configs/
# Можно задать переменные окружения (если хочешь)
ENV PORT=8080
EXPOSE 8080

CMD ["./server"]
