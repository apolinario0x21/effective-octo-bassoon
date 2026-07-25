.PHONY: build run test vet fmt tidy clean \
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
