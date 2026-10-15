# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copia código fonte
COPY . .

# Compila o ponto de entrada da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

# Copia o binário compilado
COPY --from=builder /app/server .

# Expõe a porta 8080
EXPOSE 8080

# Comando para rodar o servidor
CMD ["./server"]
