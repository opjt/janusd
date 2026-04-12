package sqlite

import (
	"context"
	"database/sql"
	"karden/internal/domain/workload"
	"time"

	_ "modernc.org/sqlite"
)

type WorkloadRepository struct {
	db *sql.DB
}

// compile-time check
var _ workload.Repository = (*WorkloadRepository)(nil)

func NewWorkloadRepository(db *sql.DB) *WorkloadRepository {
	return &WorkloadRepository{db: db}
}

func (r *WorkloadRepository) Upsert(ctx context.Context, w *workload.ManagedWorkload) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO managed_workloads
			(pod_name, namespace, secret_name, type, db_type, db_host, db_port, rotation_days, status)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(pod_name, namespace) DO UPDATE SET
			secret_name   = excluded.secret_name,
			type          = excluded.type,
			db_type       = excluded.db_type,
			db_host       = excluded.db_host,
			db_port       = excluded.db_port,
			rotation_days = excluded.rotation_days,
			status        = excluded.status
	`,
		w.PodName, w.Namespace, w.SecretName,
		w.Type, w.DBType, w.DBHost, w.DBPort,
		w.RotationDays, w.Status,
	)
	return err
}

func (r *WorkloadRepository) SetInactive(ctx context.Context, podName, namespace string) error {
	err := r.updateStatus(ctx, podName, namespace, workload.StatusInactive)
	return err
}
func (r *WorkloadRepository) updateStatus(ctx context.Context, podName, namespace string, status workload.Status) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE managed_workloads SET status = ?
		WHERE pod_name = ? AND namespace = ?
	`, string(status), podName, namespace)
	return err
}

func (r *WorkloadRepository) UpdateLastRotated(ctx context.Context, podName, namespace string, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE managed_workloads SET last_rotated_at = ?
		WHERE pod_name = ? AND namespace = ?
	`, t, podName, namespace)
	return err
}

func (r *WorkloadRepository) List(ctx context.Context) ([]*workload.ManagedWorkload, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, pod_name, namespace, secret_name, type, db_type, db_host, db_port,
		       rotation_days, last_rotated_at, status, created_at
		FROM managed_workloads
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*workload.ManagedWorkload
	for rows.Next() {
		w := &workload.ManagedWorkload{}
		var lastRotatedAt sql.NullTime
		var dbType, dbHost sql.NullString
		var dbPort sql.NullInt64

		err := rows.Scan(
			&w.ID, &w.PodName, &w.Namespace, &w.SecretName,
			&w.Type, &dbType, &dbHost, &dbPort,
			&w.RotationDays, &lastRotatedAt, &w.Status, &w.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if lastRotatedAt.Valid {
			w.LastRotatedAt = &lastRotatedAt.Time
		}
		if dbType.Valid {
			w.DBType = workload.DBType(dbType.String)
		}
		if dbHost.Valid {
			w.DBHost = dbHost.String
		}
		if dbPort.Valid {
			w.DBPort = int(dbPort.Int64)
		}

		results = append(results, w)
	}
	return results, nil
}
