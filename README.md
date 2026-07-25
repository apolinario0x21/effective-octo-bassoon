# Students API

API RESTful para gerenciar estudantes, escrita em Go com [Echo](https://echo.labstack.com/)
e [GORM](https://gorm.io/). Persistência em **PostgreSQL**, cache em **Redis** e
observabilidade com **Prometheus** e **Grafana** — tudo orquestrado por **Docker Compose**.

## Stack

| Camada          | Tecnologia                                   |
| --------------- | -------------------------------------------- |
| Linguagem       | Go 1.26                                       |
| Framework HTTP  | Echo v4                                        |
| ORM             | GORM                                           |
| Banco           | PostgreSQL 16 (SQLite pure-Go em dev/testes)   |
| Cache           | Redis 7                                        |
| Métricas        | Prometheus (`/metrics` via echo-prometheus)    |
| Dashboards      | Grafana (datasource e dashboard provisionados) |
| Logs            | zerolog (JSON estruturado, com request ID)     |
| Containers      | Docker + Docker Compose                        |

## Rotas

| Método | Rota                    | Descrição                            | Sucesso |
| ------ | ----------------------- | ------------------------------------ | ------- |
| GET    | `/healthz`              | Health check                         | 200     |
| GET    | `/metrics`              | Métricas Prometheus                  | 200     |
| GET    | `/students`             | Lista todos os estudantes            | 200     |
| GET    | `/students?active=true` | Lista estudantes ativos (ou `false`) | 200     |
| POST   | `/students`             | Cria um estudante                    | 201     |
| GET    | `/students/:id`         | Mostra um estudante                  | 200     |
| PUT    | `/students/:id`         | Atualiza um estudante (parcial)      | 200     |
| DELETE | `/students/:id`         | Remove um estudante                  | 204     |

Erros são retornados em JSON: `{"error": "mensagem"}` com o status HTTP adequado
(400 para requisição inválida, 404 para não encontrado, 409 para CPF duplicado).

## Modelo

```json
{
  "name": "Maria",
  "cpf": "52998224725",
  "email": "maria@example.com",
  "age": 22,
  "active": true
}
```

- `cpf` é uma string com 11 dígitos (sem pontuação), validado com os dígitos
  verificadores; CPFs duplicados são rejeitados com 409.
- `email` é validado quanto ao formato.
- No `PUT`, todos os campos são opcionais — apenas os enviados são atualizados.

## Arquitetura

```
cmd/api/             # entrypoint: config, wiring (DB + cache) e graceful shutdown
internal/api/        # camada HTTP: servidor, rotas, handlers, DTOs, validação, métricas
internal/cache/      # cliente Redis
internal/config/     # configuração via variáveis de ambiente
internal/db/         # conexão (postgres/sqlite), pool e migrações
internal/models/     # entidades de domínio
internal/repository/ # acesso a dados (GORM) + decorator de cache (Redis)
deploy/              # provisionamento de Prometheus e Grafana
```

O cache é aplicado como um **decorator** sobre o repositório: `CachedStudentRepository`
envolve o repositório GORM, cacheando leituras por ID e invalidando em update/delete.
Falhas de cache nunca quebram a requisição — a operação recorre ao banco. Se `REDIS_ADDR`
estiver vazio (ou o Redis indisponível), a API acessa o banco diretamente.

## Como rodar

### Opção 1 — stack completa com Docker Compose (recomendado)

Sobe API + PostgreSQL + Redis + Prometheus + Grafana:

```bash
cp .env.example .env      # opcional: ajuste credenciais
make compose-up           # ou: docker compose up --build -d
```

| Serviço    | URL                                             |
| ---------- | ----------------------------------------------- |
| API        | http://localhost:8080                           |
| Prometheus | http://localhost:9090                           |
| Grafana    | http://localhost:3000  (admin/admin por padrão) |

No Grafana, o dashboard **"Students API"** e o datasource Prometheus já vêm provisionados.

```bash
make compose-logs   # acompanha os logs da API
make compose-down   # derruba a stack
```

### Opção 2 — local, sem dependências externas

Usa SQLite (pure-Go, sem cgo) e sem Redis. Requer apenas Go 1.26+:

```bash
make run     # DB_DRIVER=sqlite, cache desabilitado
make test    # roda os testes
make build   # gera bin/api
```

## Configuração (variáveis de ambiente)

| Variável         | Padrão      | Descrição                                        |
| ---------------- | ----------- | ------------------------------------------------ |
| `PORT`           | `8080`      | Porta HTTP                                        |
| `LOG_LEVEL`      | `info`      | `debug` \| `info` \| `warn` \| `error`            |
| `DB_DRIVER`      | `postgres`  | `postgres` \| `sqlite`                            |
| `DB_HOST`        | `localhost` | Host do PostgreSQL                                |
| `DB_PORT`        | `5432`      | Porta do PostgreSQL                               |
| `DB_USER`        | `students`  | Usuário                                           |
| `DB_PASSWORD`    | `students`  | Senha                                             |
| `DB_NAME`        | `students`  | Nome do banco                                     |
| `DB_SSLMODE`     | `disable`   | Modo SSL do Postgres                              |
| `DB_DSN`         | —           | DSN completa (sobrepõe os campos acima)           |
| `REDIS_ADDR`     | —           | Ex.: `localhost:6379`; vazio desabilita o cache   |
| `REDIS_PASSWORD` | —           | Senha do Redis                                    |
| `REDIS_DB`       | `0`         | Índice do banco Redis                             |
| `REDIS_TTL`      | `5m`        | TTL das entradas de cache                         |

## Observabilidade

- **Métricas**: `/metrics` expõe contadores e histogramas por rota/método/status
  (subsystem `students_*`), coletados pelo Prometheus a cada 15s.
- **Dashboard**: painéis de request rate, taxa de erro (5xx), latência p50/p95/p99 e
  distribuição por status code.
- **Logs**: JSON estruturado via zerolog, com `request_id` correlacionável.

## Exemplos com curl

```bash
# Criar um estudante
curl -X POST localhost:8080/students \
  -H 'Content-Type: application/json' \
  -d '{"name":"Maria","cpf":"52998224725","email":"maria@example.com","age":22,"active":true}'

# Listar todos / apenas ativos
curl localhost:8080/students
curl "localhost:8080/students?active=true"

# Buscar, atualizar (parcial) e remover
curl localhost:8080/students/1
curl -X PUT localhost:8080/students/1 -H 'Content-Type: application/json' -d '{"age":23}'
curl -X DELETE localhost:8080/students/1
```
