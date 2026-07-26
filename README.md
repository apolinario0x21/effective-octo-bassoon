# Students API

[![CI](https://github.com/apolinario0x21/students-api/actions/workflows/ci.yml/badge.svg)](https://github.com/apolinario0x21/students-api/actions/workflows/ci.yml)
[![Security](https://github.com/apolinario0x21/students-api/actions/workflows/security.yml/badge.svg)](https://github.com/apolinario0x21/students-api/actions/workflows/security.yml)

API RESTful para gerenciar estudantes, escrita em Go com [Echo](https://echo.labstack.com/)
e [GORM](https://gorm.io/). Persistência em **PostgreSQL**, cache em **Redis** e
observabilidade com **Prometheus** e **Grafana** — tudo orquestrado por **Docker Compose**.

---

## 🧒 Explicando como se você nunca tivesse programado

Imagine uma **secretaria de escola** que guarda uma ficha de cada aluno (nome, CPF,
e-mail, idade e se está ativo). Este projeto é essa secretaria, só que digital.

- Você **conversa** com ela mandando "recadinhos" pela internet (chamados de
  _requisições_). Cada recadinho pede uma coisa: "me mostre todos os alunos",
  "cadastre esse aluno novo", "apague o aluno número 3".
- A secretaria **guarda as fichas** num arquivo bem organizado (o banco de dados
  PostgreSQL). É como um armário de gavetas que nunca esquece nada.
- Para ser mais rápida, ela mantém as fichas mais pedidas numa **mesa ao lado**
  (o cache Redis) — assim não precisa abrir a gaveta toda hora.
- E tem um **painel de controle** (Grafana) que mostra em tempo real quantos
  recadinhos chegaram, quais deram erro e quão rápido ela respondeu.

Você não precisa entender nada disso para usar. Só precisa mandar os recadinhos.
As próximas seções mostram como — passo a passo, sem pressa.

### O que é uma "rota"?

Uma **rota** é um endereço + uma ação. Assim como uma casa tem um endereço, cada
função da API tem o seu. Por exemplo, o endereço `/students` com a ação "GET"
(pegar) significa "me dê a lista de alunos". Os verbos são sempre estes:

| Verbo    | O que significa, em palavras simples          |
| -------- | --------------------------------------------- |
| `GET`    | "Me **mostre** algo" (só lê, não muda nada)   |
| `POST`   | "**Crie** uma coisa nova"                     |
| `PUT`    | "**Altere** uma coisa que já existe"          |
| `DELETE` | "**Apague** uma coisa"                        |

---

## 🚀 Começando: rodando a aplicação pela primeira vez

Você tem **dois caminhos**. Se está começando agora, use a **Opção 2** — é a mais
simples e não instala nada além do Go.

### Opção 2 (mais fácil) — rodar localmente, sem instalar banco nem cache

O único pré-requisito é ter o **Go 1.25+** instalado ([baixe aqui](https://go.dev/dl/)).
Abra um terminal na pasta do projeto e digite:

```bash
make run
```

Pronto! 🎉 A API sobe em `http://localhost:8080` usando um banco de dados de
arquivo simples (SQLite) e sem cache. Isso é perfeito para testar e aprender.

> **O que aconteceu?** O comando `make run` é um atalho. Por baixo ele roda a API
> pedindo para usar o banco simples (`DB_DRIVER=sqlite`) e criando um arquivo
> `student.db` na pasta. Nenhum programa externo é necessário.

Para verificar se está no ar, abra **outro** terminal e peça um "sinal de vida":

```bash
curl http://localhost:8080/healthz
```

Se responder algo como `{"status":"ok"}`, está tudo funcionando.
Para desligar, volte ao primeiro terminal e aperte `Ctrl + C`.

### Opção 1 (completa) — a stack inteira com Docker

Esta opção sobe **tudo junto** (API + PostgreSQL + Redis + Prometheus + Grafana).
Você precisa ter o [Docker](https://docs.docker.com/get-docker/) instalado.

```bash
cp .env.example .env      # opcional: cria um arquivo de configuração; pode ajustar senhas
make compose-up           # sobe tudo (equivale a: docker compose up --build -d)
```

Depois de alguns segundos, tudo estará no ar nestes endereços:

| Serviço    | Onde acessar (abra no navegador)                | Para quê serve                        |
| ---------- | ----------------------------------------------- | ------------------------------------- |
| API        | http://localhost:8080                           | Onde você manda os recadinhos         |
| Prometheus | http://localhost:9090                           | Coletor de estatísticas               |
| Grafana    | http://localhost:3000  (login `admin`/`admin`)  | Painel visual bonito das estatísticas |

Comandos úteis:

```bash
make compose-logs   # ver o que a API está fazendo, ao vivo
make compose-down   # desligar tudo
```

---

## 📬 Usando a API na prática (passo a passo)

Vamos usar o `curl`, um programinha de terminal que manda recadinhos pela internet.
(Se preferir clicar em botões, ferramentas como [Postman](https://www.postman.com/)
ou [Insomnia](https://insomnia.rest/) fazem o mesmo de forma visual.)

Cada aluno é uma ficha assim:

```json
{
  "name": "Maria",
  "cpf": "52998224725",
  "email": "maria@example.com",
  "age": 22,
  "active": true
}
```

> **Regrinhas das fichas:**
> - `cpf`: 11 dígitos, **só números** (sem pontos ou traços). A API confere se o
>   CPF é matematicamente válido e **não deixa cadastrar dois CPFs iguais**.
> - `email`: precisa ter cara de e-mail de verdade.
> - Ao **alterar** (PUT), você manda só os campos que quer mudar; o resto fica igual.

### 1. Cadastrar um aluno novo (POST)

```bash
curl -X POST localhost:8080/students \
  -H 'Content-Type: application/json' \
  -d '{"name":"Maria","cpf":"52998224725","email":"maria@example.com","age":22,"active":true}'
```

A API responde com a ficha criada, agora com um número de identificação (`id`).
Guarde esse `id` — é como o número da carteirinha do aluno.

### 2. Ver os alunos (GET)

```bash
curl localhost:8080/students                    # primeira página (20 por padrão)
curl "localhost:8080/students?active=true"      # só os que estão ativos
curl "localhost:8080/students?limit=10&offset=20"  # página: 10 alunos, pulando os 20 primeiros
curl localhost:8080/students/1                  # só o aluno de id 1
```

A lista vem **paginada**. Você controla com dois parâmetros opcionais:

- `limit` — quantos alunos por página (padrão `20`, máximo `100`).
- `offset` — quantos pular antes de começar (padrão `0`).

Valores inválidos (`limit` fora de `1..100`, `offset` negativo, ou não numéricos)
retornam `400`. A resposta traz a lista e mais três campos: `total` (quantos
alunos existem no filtro), `limit` e `offset` aplicados:

```json
{
  "students": [ { "id": 21, "name": "Maria", "...": "..." } ],
  "total": 137,
  "limit": 10,
  "offset": 20
}
```

### 3. Alterar um aluno (PUT)

Digamos que a Maria fez aniversário. Mande **só** o campo que mudou:

```bash
curl -X PUT localhost:8080/students/1 \
  -H 'Content-Type: application/json' \
  -d '{"age":23}'
```

### 4. Apagar um aluno (DELETE)

```bash
curl -X DELETE localhost:8080/students/1
```

### E quando dá errado?

A API sempre explica o erro em português-de-máquina (JSON), com um número que diz
o "tipo" do problema:

| Número | Significado                | Exemplo                                  |
| ------ | -------------------------- | ---------------------------------------- |
| `400`  | Você mandou algo inválido  | CPF com letras, e-mail sem `@`           |
| `404`  | Não achei                  | Pediu o aluno 999 que não existe         |
| `409`  | Conflito                   | Tentou cadastrar um CPF que já existe    |

Exemplo de resposta de erro: `{"error": "cpf já cadastrado"}`.

---

## 📋 Referência das rotas

| Método | Rota                    | Descrição                            | Acesso        | Sucesso |
| ------ | ----------------------- | ------------------------------------ | ------------- | ------- |
| GET    | `/healthz`              | Sinal de vida (health check)         | público       | 200     |
| GET    | `/metrics`              | Métricas para o Prometheus           | público       | 200     |
| POST   | `/auth/register`        | Cria uma conta (papel `user`)        | público       | 201     |
| POST   | `/auth/login`           | Autentica e emite tokens             | público       | 200     |
| POST   | `/auth/refresh`         | Renova o access token                | público       | 200     |
| POST   | `/auth/logout`          | Revoga o refresh token (encerra sessão) | público    | 204     |
| GET    | `/students`             | Lista estudantes (paginada)          | autenticado   | 200     |
| GET    | `/students/:id`         | Mostra um estudante                  | autenticado   | 200     |
| POST   | `/students`             | Cria um estudante                    | **admin**     | 201     |
| PUT    | `/students/:id`         | Atualiza um estudante (parcial)      | **admin**     | 200     |
| DELETE | `/students/:id`         | Remove um estudante                  | **admin**     | 204     |

Rotas **autenticadas** exigem o cabeçalho `Authorization: Bearer <access_token>`
(401 sem token válido). As de **admin** exigem, além disso, o papel admin (403 caso
contrário).

---

## 📖 Documentação interativa (Swagger)

Com a aplicação rodando, abra no navegador:

```
http://localhost:8080/swagger/index.html
```

É uma página onde você **vê todas as rotas, os campos de cada uma e pode testá-las
clicando em "Try it out"** — sem precisar decorar comandos `curl`. A especificação
OpenAPI crua fica em `/swagger/doc.json`.

A documentação é **gerada a partir de comentários** nos handlers (`internal/api/`) e
versionada em [`docs/`](docs/). Se você mudar uma rota, regenere com:

```bash
make swagger   # requer: go install github.com/swaggo/swag/cmd/swag@latest
```

---

## 🔐 Autenticação (JWT + papéis)

A API usa **JWT** com dois papéis: `user` (só lê) e `admin` (lê e escreve). O fluxo:

**1. Crie uma conta** (nasce como `user`):

```bash
curl -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"maria","password":"password123"}'
```

**2. Faça login** para receber os tokens:

```bash
curl -X POST localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"maria","password":"password123"}'
# → {"access_token":"...","refresh_token":"...","token_type":"Bearer"}
```

**3. Use o access token** nas rotas protegidas:

```bash
curl localhost:8080/students -H 'Authorization: Bearer SEU_ACCESS_TOKEN'
```

**4. Quando o access token expirar** (15 min), troque o refresh token por um novo par
(o refresh usado é revogado):

```bash
curl -X POST localhost:8080/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"SEU_REFRESH_TOKEN"}'
```

**5. Para encerrar a sessão**, revogue o refresh token:

```bash
curl -X POST localhost:8080/auth/logout \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"SEU_REFRESH_TOKEN"}'
```

Refresh tokens expirados ou revogados são apagados do banco periodicamente por um
worker em background (intervalo em `REFRESH_CLEANUP_INTERVAL`, padrão `1h`).

> **Papéis:** `register` sempre cria `user`. Para ter um **admin**, defina
> `ADMIN_USERNAME` e `ADMIN_PASSWORD` no ambiente — no boot, se o usuário não existir,
> ele é criado como admin. As senhas são guardadas apenas como **hash bcrypt**, nunca em
> texto puro, e não aparecem em respostas nem em logs.
>
> **Segredo do JWT:** `JWT_SECRET` nunca é hardcoded. Em produção
> (`APP_ENV=production`) é obrigatório e a API não sobe sem ele; em desenvolvimento,
> se vazio, um segredo efêmero é gerado no boot.

---

## 🧰 Stack (as ferramentas usadas)

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
| Docs da API     | OpenAPI/Swagger (swaggo + echo-swagger)        |
| Containers      | Docker + Docker Compose                        |
| CI              | GitHub Actions (lint, testes, build, docker)   |
| Segurança       | golangci-lint, govulncheck, gosec, Trivy       |

---

## 🤖 Integração Contínua (CI)

Toda vez que alguém envia código (push) ou abre um Pull Request para o branch
`main`, o **GitHub Actions** roda automaticamente uma esteira de verificação —
como um inspetor de qualidade que confere o trabalho antes de aceitá-lo. Se algo
quebrar, o selo do topo do README fica vermelho. As etapas estão em
[`.github/workflows/ci.yml`](.github/workflows/ci.yml):

| Etapa         | O que verifica                                                        |
| ------------- | -------------------------------------------------------------------- |
| **Lint & Vet**| Formatação (`gofmt`), erros suspeitos (`go vet`), `golangci-lint` e `go.mod` em dia |
| **Test**      | Roda todos os testes com detector de _race conditions_ e mede cobertura |
| **Build**     | Garante que o binário compila                                         |
| **Docker**    | Garante que a imagem Docker constrói                                  |

Além disso, um workflow separado de **segurança**
([`.github/workflows/security.yml`](.github/workflows/security.yml)) roda a cada
push/PR na `main` e também semanalmente:

| Etapa           | O que verifica                                                     |
| --------------- | ----------------------------------------------------------------- |
| **govulncheck** | Vulnerabilidades conhecidas (CVEs) nas dependências Go            |
| **gosec**       | Padrões inseguros no código-fonte Go                             |
| **Trivy**       | CVEs corrigíveis na imagem Docker (severidade CRITICAL/HIGH)      |

As atualizações de dependências (Go, GitHub Actions e Docker) são propostas
automaticamente pelo **Dependabot** ([`.github/dependabot.yml`](.github/dependabot.yml)),
semanalmente e agrupadas por ecossistema.

Você pode rodar as mesmas checagens no seu computador antes de enviar:

```bash
make fmt    # formata o código
make vet    # procura erros comuns
make lint   # golangci-lint (mesmo conjunto do CI)
make test   # roda os testes
make build  # compila o binário
```

---

## 🏛️ Arquitetura

```
cmd/api/             # entrypoint: config, wiring (DB + cache + auth) e graceful shutdown
internal/api/        # camada HTTP: servidor, rotas, handlers, DTOs, validação, auth middleware
internal/auth/       # emissão/validação de JWT e refresh tokens
internal/crypto/     # hashing de senha (bcrypt)
internal/cache/      # cliente Redis
internal/config/     # configuração via variáveis de ambiente
internal/db/         # conexão (postgres/sqlite) e pool
internal/models/     # entidades de domínio
internal/repository/ # acesso a dados (GORM) + decorator de cache (Redis)
migrations/          # migrações SQL versionadas + runner (postgres/ e sqlite/)
deploy/              # provisionamento de Prometheus e Grafana
docs/                # especificação OpenAPI gerada pelo swag (make swagger)
.github/workflows/   # pipeline de CI (GitHub Actions)
```

### Migrações de schema

O schema do banco é criado por **migrações SQL versionadas** (não mais pelo
`AutoMigrate` do GORM). Os arquivos ficam em [`migrations/`](migrations/),
numerados (`001_create_students.sql`, ...) e separados por dialeto —
[`postgres/`](migrations/postgres/) e [`sqlite/`](migrations/sqlite/) — porque os
tipos divergem (ex.: `BIGSERIAL`/`TIMESTAMPTZ` no Postgres vs.
`INTEGER AUTOINCREMENT`/`DATETIME` no SQLite).

Um runner aplica as pendentes **no boot** (em `make run` e no container),
controlando o que já rodou pela tabela `schema_migrations`; rodar de novo é
seguro (idempotente). Para adicionar uma mudança de schema, crie o próximo par de
arquivos numerados nas duas pastas.

O cache é aplicado como um **decorator** sobre o repositório: `CachedStudentRepository`
envolve o repositório GORM, cacheando leituras por ID e invalidando em update/delete.
Falhas de cache nunca quebram a requisição — a operação recorre ao banco. Se `REDIS_ADDR`
estiver vazio (ou o Redis indisponível), a API acessa o banco diretamente.

---

## ⚙️ Configuração (variáveis de ambiente)

Não precisa mexer em nada para começar (a Opção 2 já vem configurada). Estas
variáveis servem para ajustar o comportamento em produção:

| Variável         | Padrão      | Descrição                                        |
| ---------------- | ----------- | ------------------------------------------------ |
| `PORT`           | `8080`      | Porta HTTP                                        |
| `LOG_LEVEL`      | `info`      | `debug` \| `info` \| `warn` \| `error`            |
| `APP_ENV`        | `development` | `development` \| `production`                   |
| `JWT_SECRET`     | —           | Segredo de assinatura do JWT (obrigatório em prod) |
| `JWT_ACCESS_TTL` | `15m`       | Validade do access token                          |
| `JWT_REFRESH_TTL`| `168h`      | Validade do refresh token (7 dias)                |
| `REFRESH_CLEANUP_INTERVAL` | `1h` | Limpeza de refresh tokens expirados (`0` desabilita) |
| `ADMIN_USERNAME` | —           | Usuário admin a provisionar no boot (opcional)    |
| `ADMIN_PASSWORD` | —           | Senha do admin a provisionar (opcional)           |
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

---

## 📊 Observabilidade

- **Métricas**: `/metrics` expõe contadores e histogramas por rota/método/status
  (subsystem `students_*`), coletados pelo Prometheus a cada 15s.
- **Dashboard**: painéis de request rate, taxa de erro (5xx), latência p50/p95/p99 e
  distribuição por status code.
- **Logs**: JSON estruturado via zerolog, com `request_id` correlacionável.
