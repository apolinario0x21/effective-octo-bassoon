.PHONY: build run test vet fmt lint tidy clean swagger \
        docker-build compose-up compose-down compose-logs

# --- Desenvolvimento local ---------------------------------------------------

build:
	go build -o bin/api ./cmd/api

# Roda a API localmente com SQLite e sem Redis (sem dependências externas).
run:
	DB_DRIVER=sqlite DB_DSN=student.db go run ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

# Requer golangci-lint instalado (linha v2). Instalação:
#   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
lint:
	golangci-lint run ./...

# Regenera a documentação OpenAPI em docs/ a partir das annotations.
# Requer o swag instalado:
#   go install github.com/swaggo/swag/cmd/swag@latest
swagger:
	swag init -g internal/api/docs.go -o docs --parseInternal

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

clean:
	rm -rf bin *.db

# --- Docker / stack completa -------------------------------------------------

docker-build:
	docker build -t students-api:latest .

# Sobe API + PostgreSQL + Redis + Prometheus + Grafana.
compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f api
