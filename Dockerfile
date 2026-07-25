# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Baixa as dependências primeiro para aproveitar o cache de camadas.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Binário estático (sem cgo) — o driver SQLite usado é pure-Go e o Postgres
# não requer cgo, então o binário roda numa imagem mínima.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# --- Runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /app/api

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
