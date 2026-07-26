// Package migrations aplica migrações de schema versionadas em SQL, substituindo
// o AutoMigrate do GORM. As migrações ficam em arquivos .sql numerados, separados
// por dialeto (postgres/ e sqlite/), e são aplicadas em ordem, uma única vez cada,
// controladas pela tabela schema_migrations.
package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed postgres/*.sql sqlite/*.sql
var files embed.FS

// Run aplica as migrações pendentes ao banco, escolhendo os arquivos conforme o
// driver ("postgres" ou "sqlite"). É idempotente: migrações já aplicadas são
// puladas. Cada migração roda dentro de uma transação.
func Run(db *gorm.DB, driver string) error {
	dir, err := dirForDriver(driver)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("accessing sql.DB: %w", err)
	}

	if err := ensureMigrationsTable(sqlDB); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	applied, err := appliedVersions(sqlDB)
	if err != nil {
		return fmt.Errorf("reading applied migrations: %w", err)
	}

	names, err := migrationFiles(dir)
	if err != nil {
		return err
	}

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}

		script, err := files.ReadFile(dir + "/" + name)
		if err != nil {
			return fmt.Errorf("reading migration %q: %w", name, err)
		}

		if err := applyMigration(sqlDB, driver, version, string(script)); err != nil {
			return fmt.Errorf("applying migration %q: %w", name, err)
		}
	}

	return nil
}

func dirForDriver(driver string) (string, error) {
	switch driver {
	case "postgres", "sqlite":
		return driver, nil
	default:
		return "", fmt.Errorf("unsupported db driver %q (want postgres or sqlite)", driver)
	}
}

// migrationFiles devolve os arquivos .sql do diretório do dialeto, ordenados
// pelo nome (o prefixo numérico define a ordem de aplicação).
func migrationFiles(dir string) ([]string, error) {
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return nil, fmt.Errorf("reading migrations dir %q: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	return names, nil
}

func ensureMigrationsTable(db *sql.DB) error {
	// Statement compatível com Postgres e SQLite.
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`)
	return err
}

func appliedVersions(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func applyMigration(db *sql.DB, driver, version, script string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op após Commit

	for _, stmt := range statements(script) {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(insertVersionStmt(driver), version, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}

	return tx.Commit()
}

// insertVersionStmt devolve o INSERT com o placeholder do dialeto ($1 no Postgres,
// ? no SQLite).
func insertVersionStmt(driver string) string {
	if driver == "postgres" {
		return `INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)`
	}
	return `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`
}

// statements quebra um script em comandos individuais separados por ';',
// descartando comentários de linha (--) e trechos vazios. Os drivers via
// database/sql executam um statement por chamada, então dividimos manualmente.
func statements(script string) []string {
	var out []string
	for _, chunk := range strings.Split(script, ";") {
		stmt := strings.TrimSpace(stripLineComments(chunk))
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}

func stripLineComments(chunk string) string {
	var lines []string
	for _, line := range strings.Split(chunk, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
