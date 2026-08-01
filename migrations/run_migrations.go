package migrations

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedFS embed.FS

func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(db, ".")
}
