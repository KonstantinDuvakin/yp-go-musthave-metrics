// Package migrations применяет к базе данных SQL-миграции, встроенные в
// бинарник через embed, средствами goose.
package migrations

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedFS embed.FS

// RunMigrations накатывает все непримененные миграции на БД db (диалект
// PostgreSQL) до последней версии.
func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(db, ".")
}
