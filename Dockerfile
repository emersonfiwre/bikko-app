# Build stage
FROM golang:1.26.0-alpine AS builder

WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copia código fonte
COPY . .

# Compila o ponto de entrada da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/bikkofy-api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o reset-local-accounts ./cmd/reset-local-accounts
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate

# Production stage
FROM alpine:3.23

WORKDIR /app

# Copia o binário compilado
COPY --from=builder /app/server .
COPY --from=builder /app/reset-local-accounts /app/migrate ./

RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app && mkdir /app/.local-media && chown -R app:app /app
USER app

# Expõe a porta 8080
EXPOSE 8080

# Comando para rodar o servidor
CMD ["./server"]
