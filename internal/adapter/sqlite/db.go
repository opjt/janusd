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
		CREATE TABLE IF NOT EXISTS managed_workloads (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			pod_name        TEXT NOT NULL,
			namespace       TEXT NOT NULL,
			secret_name     TEXT NOT NULL,
			type            TEXT NOT NULL,
			db_type         TEXT,
			db_host         TEXT,
			db_port         INTEGER,
			rotation_days   INTEGER NOT NULL DEFAULT 30,
			last_rotated_at DATETIME,
			status          TEXT NOT NULL DEFAULT 'active',
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(pod_name, namespace)
		);

		CREATE TABLE IF NOT EXISTS audit_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id   INTEGER NOT NULL,
			action      TEXT NOT NULL,
			actor       TEXT NOT NULL DEFAULT 'karden',
			result      TEXT NOT NULL,
			reason      TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(target_id) REFERENCES managed_workloads(id)
		);
	`)
	return err
}
