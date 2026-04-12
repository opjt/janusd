package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			namespace   TEXT NOT NULL,
			secret_name TEXT NOT NULL,
			action      TEXT NOT NULL,
			actor       TEXT NOT NULL DEFAULT 'karden',
			result      TEXT NOT NULL,
			reason      TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
