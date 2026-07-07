# API Students

API RESTful para gerenciar estudantes, escrita em Go com [Echo](https://echo.labstack.com/), [GORM](https://gorm.io/) e SQLite.

## Rotas

| Método | Rota                     | Descrição                                  |
| ------ | ------------------------ | ------------------------------------------ |
| GET    | `/students`              | Lista todos os estudantes                  |
| GET    | `/students?active=true`  | Lista estudantes ativos (ou `false`)       |
| POST   | `/students`              | Cria um estudante                          |
| GET    | `/students/:id`          | Mostra um estudante                        |
| PUT    | `/students/:id`          | Atualiza um estudante                      |
| DELETE | `/students/:id`          | Remove um estudante                        |

## Struct Student

```go
type Student struct {
    Name   string
    CPF    int
    Email  string
    Age    int
    Active bool
}
```

## Como rodar

Requisitos: Go 1.23+ e um compilador C (o driver SQLite usa cgo).

```bash
go run main.go
```

O servidor sobe em `http://localhost:8080` e cria o arquivo `student.db` automaticamente.

## Exemplos com curl

```bash
# Criar um estudante
curl -X POST localhost:8080/students \
  -H 'Content-Type: application/json' \
  -d '{"name":"Maria","cpf":12345678901,"email":"maria@example.com","age":22,"active":true}'

# Listar todos
curl localhost:8080/students

# Listar apenas ativos
curl "localhost:8080/students?active=true"

# Buscar por ID
curl localhost:8080/students/1

# Atualizar
curl -X PUT localhost:8080/students/1 \
  -H 'Content-Type: application/json' \
  -d '{"age":23}'

# Remover
curl -X DELETE localhost:8080/students/1
```
