-- Cria a tabela de estudantes (dialeto PostgreSQL).
-- Reproduz o schema antes gerado pelo AutoMigrate do GORM a partir de
-- models.Student (embutindo os campos de gorm.Model: id/created_at/updated_at/deleted_at).
CREATE TABLE IF NOT EXISTS students (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    name       TEXT NOT NULL,
    cpf        VARCHAR(11) NOT NULL,
    email      TEXT NOT NULL,
    age        BIGINT NOT NULL,
    active     BOOLEAN NOT NULL
);

-- Índice do soft-delete do GORM (filtra deleted_at IS NULL nas queries).
CREATE INDEX IF NOT EXISTS idx_students_deleted_at ON students (deleted_at);

-- Índice de busca/checagem de unicidade por CPF (models.Student: `index`).
CREATE INDEX IF NOT EXISTS idx_students_cpf ON students (cpf);
