# Students API

[![CI](https://github.com/apolinario0x21/students-api/actions/workflows/ci.yml/badge.svg)](https://github.com/apolinario0x21/students-api/actions/workflows/ci.yml)
[![Security](https://github.com/apolinario0x21/students-api/actions/workflows/security.yml/badge.svg)](https://github.com/apolinario0x21/students-api/actions/workflows/security.yml)

API RESTful para gestão de estudantes, escrita em **Go** com [Echo](https://echo.labstack.com/)
e [GORM](https://gorm.io/). Inclui autenticação JWT com RBAC, listagem paginada,
cache, migrações versionadas, documentação OpenAPI e observabilidade — com persistência
em **PostgreSQL** e stack completa orquestrada por **Docker Compose**.

## Funcionalidades

- **CRUD de estudantes** com validação de CPF (dígitos verificadores + unicidade),
  e-mail e demais campos.
- **Autenticação JWT + RBAC**: papéis `user` (leitura) e `admin` (escrita), com
  access token curto, refresh token com rotação, logout e limpeza automática.
- **Listagem paginada** (`limit`/`offset` com teto) e filtro por status.
- **Cache Redis** aplicado como decorator, com invalidação correta por página.
- **Migrações SQL versionadas** por dialeto, executadas no boot.
- **Documentação OpenAPI/Swagger** interativa, gerada a partir do código.
- **Observabilidade**: métricas Prometheus, logs estruturados e trilha de auditoria.
- **Qualidade e segurança automatizadas** no CI (lint, testes, `govulncheck`, `gosec`, Trivy).

## Stack

| Camada          | Tecnologia                                     |
| --------------- | ---------------------------------------------- |
| Linguagem       | Go 1.25                                         |
| Framework HTTP  | Echo v4                                         |
| ORM             | GORM                                           |
| Banco           | PostgreSQL 16 (SQLite pure-Go em dev/testes)   |
| Autenticação    | JWT (golang-jwt) + RBAC + bcrypt               |
| Cache           | Redis 7                                        |
| Métricas        | Prometheus (`/metrics` via echo-prometheus)    |
| Dashboards      | Grafana (datasource e dashboard provisionados) |
| Logs            | zerolog (JSON estruturado, com request ID)     |
| Documentação    | OpenAPI/Swagger (swaggo + echo-swagger)        |
| Containers      | Docker + Docker Compose                        |
| CI/Segurança    | GitHub Actions · golangci-lint · govulncheck · gosec · Trivy |

## Arquitetura

```
cmd/api/             # entrypoint: configuração, wiring e graceful shutdown
internal/api/        # camada HTTP: servidor, rotas, handlers, DTOs, middleware, auditoria
internal/auth/       # emissão e validação de JWT e refresh tokens
internal/crypto/     # hashing de senha (bcrypt)
internal/cache/      # cliente Redis
internal/config/     # configuração via variáveis de ambiente
internal/db/         # conexão (postgres/sqlite) e pool
internal/models/     # entidades de domínio
internal/repository/ # acesso a dados (GORM) + decorator de cache
migrations/          # migrações SQL versionadas + runner (postgres/ e sqlite/)
docs/                # especificação OpenAPI gerada por swag
deploy/              # provisionamento de Prometheus e Grafana
.github/workflows/   # pipelines de CI e segurança
```

O cache é um **decorator** sobre o repositório GORM (`CachedStudentRepository`):
cacheia leituras e listagens e invalida em cada escrita. Falhas de cache nunca quebram
a requisição — a operação recorre ao banco. Sem `REDIS_ADDR`, a API acessa o banco
diretamente.

## Início rápido

### Opção 1 — stack completa via Docker Compose (recomendado)

Sobe API, PostgreSQL, Redis, Prometheus e Grafana:

```bash
cp .env.example .env      # ajuste credenciais e segredos conforme necessário
make compose-up           # docker compose up --build -d
```

| Serviço    | URL                                             |
| ---------- | ----------------------------------------------- |
| API        | http://localhost:8080                           |
| Swagger    | http://localhost:8080/swagger/index.html        |
| Prometheus | http://localhost:9090                           |
| Grafana    | http://localhost:3000  (admin/admin por padrão) |

```bash
make compose-logs   # acompanha os logs da API
make compose-down   # derruba a stack
```

### Opção 2 — local, sem dependências externas

Usa SQLite (pure-Go, sem cgo) e sem Redis. Requer apenas Go 1.25+:

```bash
make run     # DB_DRIVER=sqlite, cache desabilitado
```

A API sobe em `http://localhost:8080`. Verifique com `curl http://localhost:8080/healthz`.

## Autenticação

A API usa **JWT** com dois papéis: `user` (somente leitura) e `admin` (leitura e
escrita). As senhas são armazenadas apenas como **hash bcrypt**, nunca em texto puro,
e não aparecem em respostas nem em logs.

```bash
# 1. Registrar uma conta (criada como 'user')
curl -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"maria","password":"password123"}'

# 2. Autenticar e receber os tokens
curl -X POST localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"maria","password":"password123"}'
# → {"access_token":"...","refresh_token":"...","token_type":"Bearer"}

# 3. Chamar rotas protegidas com o access token
curl localhost:8080/students -H 'Authorization: Bearer <ACCESS_TOKEN>'

# 4. Renovar o access token quando expirar (rotaciona o refresh)
curl -X POST localhost:8080/auth/refresh \
  -H 'Content-Type: application/json' -d '{"refresh_token":"<REFRESH_TOKEN>"}'

# 5. Encerrar a sessão (revoga o refresh token)
curl -X POST localhost:8080/auth/logout \
  -H 'Content-Type: application/json' -d '{"refresh_token":"<REFRESH_TOKEN>"}'
```

- **Papéis:** o registro sempre cria um `user`. Para provisionar um **admin**, defina
  `ADMIN_USERNAME` e `ADMIN_PASSWORD` — no boot, se ainda não existir, o usuário é criado
  como admin.
- **Segredo do JWT:** `JWT_SECRET` nunca é hardcoded. Em produção (`APP_ENV=production`)
  é obrigatório e a API não sobe sem ele; em desenvolvimento, se vazio, um segredo
  efêmero é gerado no boot.
- **Ciclo de vida dos tokens:** o refresh é rotacionado a cada uso (o anterior é
  revogado) e tokens expirados/revogados são apagados periodicamente por um worker
  (`REFRESH_CLEANUP_INTERVAL`, padrão `1h`).

## Referência da API

| Método | Rota                | Descrição                          | Acesso      | Sucesso |
| ------ | ------------------- | ---------------------------------- | ----------- | ------- |
| GET    | `/healthz`          | Health check                       | público     | 200     |
| GET    | `/metrics`          | Métricas Prometheus                | público     | 200     |
| POST   | `/auth/register`    | Cria uma conta (papel `user`)      | público     | 201     |
| POST   | `/auth/login`       | Autentica e emite tokens           | público     | 200     |
| POST   | `/auth/refresh`     | Renova o access token              | público     | 200     |
| POST   | `/auth/logout`      | Revoga o refresh token             | público     | 204     |
| GET    | `/students`         | Lista estudantes (paginada)        | autenticado | 200     |
| GET    | `/students/:id`     | Detalha um estudante               | autenticado | 200     |
| POST   | `/students`         | Cria um estudante                  | **admin**   | 201     |
| PUT    | `/students/:id`     | Atualiza um estudante (parcial)    | **admin**   | 200     |
| DELETE | `/students/:id`     | Remove um estudante                | **admin**   | 204     |

Rotas **autenticadas** exigem `Authorization: Bearer <access_token>` (401 sem token
válido); as de **admin** exigem, adicionalmente, o papel admin (403 caso contrário).
Erros são retornados como `{"error": "mensagem"}` com o status adequado (400 inválido,
404 não encontrado, 409 conflito de CPF).

### Modelo de estudante

```json
{
  "name": "Maria",
  "cpf": "52998224725",
  "email": "maria@example.com",
  "age": 22,
  "active": true
}
```

- `cpf`: string de 11 dígitos (sem pontuação), validada pelos dígitos verificadores;
  duplicatas retornam 409.
- `email`: validado quanto ao formato.
- No `PUT`, todos os campos são opcionais — apenas os enviados são atualizados.

### Paginação

`GET /students` é paginada por dois parâmetros opcionais:

- `limit` — itens por página (padrão `20`, máximo `100`).
- `offset` — itens a pular (padrão `0`).

Aceita também `active=true|false` como filtro. Valores inválidos retornam 400. A
resposta inclui a lista e os metadados `total`, `limit` e `offset`:

```json
{ "students": [ ... ], "total": 137, "limit": 10, "offset": 20 }
```

### Documentação interativa (Swagger)

Com a aplicação no ar, a UI do Swagger fica em `http://localhost:8080/swagger/index.html`
(especificação crua em `/swagger/doc.json`). É gerada a partir de anotações nos handlers
e versionada em [`docs/`](docs/); regenere com `make swagger`.

## Configuração

Todas as variáveis têm padrões adequados a desenvolvimento (a Opção 2 funciona sem
configuração). Ajuste-as para produção:

| Variável                   | Padrão        | Descrição                                          |
| -------------------------- | ------------- | -------------------------------------------------- |
| `PORT`                     | `8080`        | Porta HTTP                                          |
| `LOG_LEVEL`                | `info`        | `debug` \| `info` \| `warn` \| `error`              |
| `APP_ENV`                  | `development` | `development` \| `production`                       |
| `JWT_SECRET`               | —             | Segredo de assinatura do JWT (obrigatório em prod)  |
| `JWT_ACCESS_TTL`           | `15m`         | Validade do access token                            |
| `JWT_REFRESH_TTL`          | `168h`        | Validade do refresh token (7 dias)                  |
| `REFRESH_CLEANUP_INTERVAL` | `1h`          | Limpeza de refresh tokens expirados (`0` desabilita)|
| `ADMIN_USERNAME`           | —             | Admin a provisionar no boot (opcional)              |
| `ADMIN_PASSWORD`           | —             | Senha do admin a provisionar (opcional)             |
| `DB_DRIVER`                | `postgres`    | `postgres` \| `sqlite`                              |
| `DB_HOST`                  | `localhost`   | Host do PostgreSQL                                  |
| `DB_PORT`                  | `5432`        | Porta do PostgreSQL                                 |
| `DB_USER`                  | `students`    | Usuário                                             |
| `DB_PASSWORD`              | `students`    | Senha                                               |
| `DB_NAME`                  | `students`    | Nome do banco                                       |
| `DB_SSLMODE`               | `disable`     | Modo SSL do Postgres                                |
| `DB_DSN`                   | —             | DSN completa (sobrepõe os campos acima)             |
| `REDIS_ADDR`               | —             | Ex.: `localhost:6379`; vazio desabilita o cache     |
| `REDIS_PASSWORD`           | —             | Senha do Redis                                      |
| `REDIS_DB`                 | `0`           | Índice do banco Redis                               |
| `REDIS_TTL`                | `5m`          | TTL das entradas de cache                           |

## Migrações

O schema é gerido por **migrações SQL versionadas** (não por `AutoMigrate`). Os arquivos
ficam em [`migrations/`](migrations/), numerados e separados por dialeto —
[`postgres/`](migrations/postgres/) e [`sqlite/`](migrations/sqlite/) — já que os tipos
divergem (`BIGSERIAL`/`TIMESTAMPTZ` vs. `INTEGER AUTOINCREMENT`/`DATETIME`). Um runner
aplica as pendentes **no boot**, controlando o histórico pela tabela `schema_migrations`
(idempotente). Para uma nova mudança de schema, crie o próximo par de arquivos numerados
nas duas pastas.

## Observabilidade

- **Métricas**: `/metrics` expõe contadores e histogramas por rota/método/status
  (subsystem `students_*`), coletados pelo Prometheus. O Grafana provisiona um dashboard
  com request rate, taxa de erro e latência p50/p95/p99.
- **Logs**: JSON estruturado via zerolog, com `request_id` correlacionável.
- **Auditoria**: cada escrita em `/students` gera uma linha com `audit=student`, a ação,
  o `student_id` e quem executou (`user_id` + `role`), correlacionada pelo `request_id`.

## Desenvolvimento

```bash
make run       # sobe a API local (SQLite, sem Redis)
make test      # executa os testes
make lint      # golangci-lint (mesmo conjunto do CI)
make fmt       # formata o código
make swagger   # regenera a documentação OpenAPI
make build     # compila o binário em bin/api
```

`make lint` requer [golangci-lint](https://golangci-lint.run/) e `make swagger` requer
[swag](https://github.com/swaggo/swag) instalados.

### Integração contínua

Cada push ou pull request para `main` dispara dois workflows do GitHub Actions:

- **CI** ([`ci.yml`](.github/workflows/ci.yml)): formatação, `go vet`, golangci-lint,
  `go mod tidy`, testes com detector de _race_ e cobertura, build do binário e da imagem
  Docker.
- **Segurança** ([`security.yml`](.github/workflows/security.yml)): `govulncheck` (CVEs
  em dependências), `gosec` (análise estática) e `Trivy` (CVEs na imagem) — também em
  execução semanal.

Atualizações de dependências (Go, GitHub Actions e Docker) são propostas semanalmente
pelo **Dependabot**, agrupadas por ecossistema.
