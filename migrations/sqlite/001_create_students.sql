-- Cria a tabela de estudantes (dialeto SQLite, usado em dev/testes por `make run`).
-- Mesmo schema lógico do Postgres, com os tipos apropriados ao SQLite:
-- INTEGER PRIMARY KEY AUTOINCREMENT para o id e DATETIME para os timestamps.
CREATE TABLE IF NOT EXISTS students (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    name       TEXT NOT NULL,
    cpf        TEXT NOT NULL,
    email      TEXT NOT NULL,
    age        INTEGER NOT NULL,
    active     BOOLEAN NOT NULL
);

-- Índice do soft-delete do GORM (filtra deleted_at IS NULL nas queries).
CREATE INDEX IF NOT EXISTS idx_students_deleted_at ON students (deleted_at);

-- Índice de busca/checagem de unicidade por CPF (models.Student: `index`).
CREATE INDEX IF NOT EXISTS idx_students_cpf ON students (cpf);
