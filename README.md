# API Students

API RESTful para gerenciar estudantes, escrita em Go com [Echo](https://echo.labstack.com/), [GORM](https://gorm.io/) e SQLite.

## Rotas

| Método | Rota                    | Descrição                            | Sucesso |
| ------ | ----------------------- | ------------------------------------ | ------- |
| GET    | `/healthz`              | Health check                         | 200     |
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

- `cpf` é uma string com 11 dígitos (sem pontuação) e é validado com os
  dígitos verificadores; CPFs duplicados são rejeitados com 409.
- `email` é validado quanto ao formato.
- No `PUT`, todos os campos são opcionais — apenas os enviados são atualizados.

## Estrutura do projeto

```
cmd/api/             # entrypoint (main): configuração, wiring e graceful shutdown
internal/api/        # camada HTTP: servidor, rotas, handlers, DTOs e validação
internal/config/     # configuração via variáveis de ambiente
internal/db/         # conexão com o banco e migrações
internal/models/     # entidades de domínio
internal/repository/ # acesso a dados (GORM), por trás de uma interface
```

## Como rodar

Requisitos: Go 1.23+ e um compilador C (o driver SQLite usa cgo).

```bash
make run    # ou: go run ./cmd/api
make test   # roda os testes
make build  # gera bin/api
```

Configuração por variáveis de ambiente:

| Variável  | Padrão       | Descrição                  |
| --------- | ------------ | -------------------------- |
| `PORT`    | `8080`       | Porta HTTP do servidor     |
| `DB_PATH` | `student.db` | Caminho do arquivo SQLite  |

O servidor sobe em `http://localhost:8080` e cria o banco automaticamente.
`Ctrl+C` (SIGINT/SIGTERM) encerra o servidor de forma graciosa.

## Exemplos com curl

```bash
# Criar um estudante
curl -X POST localhost:8080/students \
  -H 'Content-Type: application/json' \
  -d '{"name":"Maria","cpf":"52998224725","email":"maria@example.com","age":22,"active":true}'

# Listar todos
curl localhost:8080/students

# Listar apenas ativos
curl "localhost:8080/students?active=true"

# Buscar por ID
curl localhost:8080/students/1

# Atualizar (parcial: só os campos enviados)
curl -X PUT localhost:8080/students/1 \
  -H 'Content-Type: application/json' \
  -d '{"age":23}'

# Remover
curl -X DELETE localhost:8080/students/1
```
