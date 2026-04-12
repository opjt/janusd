package sqlite

import (
	"context"
	"database/sql"
	"karden/internal/domain/audit"
)

type AuditRepository struct {
	db *sql.DB
}

// compile-time check
var _ audit.Repository = (*AuditRepository)(nil)

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Save(ctx context.Context, log *audit.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (target_id, action, actor, result, reason)
		VALUES (?, ?, ?, ?, ?)
	`, log.TargetID, log.Action, log.Actor, log.Result, log.Reason)
	return err
}

func (r *AuditRepository) ListByTarget(ctx context.Context, targetID int) ([]*audit.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, target_id, action, actor, result, reason, created_at
		FROM audit_logs
		WHERE target_id = ?
		ORDER BY created_at DESC
	`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditRows(rows)
}

// List returns audit logs optionally filtered by namespace and/or secret name.
func (r *AuditRepository) List(ctx context.Context, namespace, secretName string) ([]*audit.AuditLog, error) {
	query := `
		SELECT a.id, a.target_id, a.action, a.actor, a.result, a.reason, a.created_at
		FROM audit_logs a
		JOIN managed_workloads w ON a.target_id = w.id
		WHERE 1=1
	`
	args := []any{}

	if namespace != "" {
		query += " AND w.namespace = ?"
		args = append(args, namespace)
	}
	if secretName != "" {
		query += " AND w.secret_name = ?"
		args = append(args, secretName)
	}

	query += " ORDER BY a.created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditRows(rows)
}

func scanAuditRows(rows *sql.Rows) ([]*audit.AuditLog, error) {
	results := make([]*audit.AuditLog, 0)
	for rows.Next() {
		l := &audit.AuditLog{}
		var reason sql.NullString
		if err := rows.Scan(&l.ID, &l.TargetID, &l.Action, &l.Actor, &l.Result, &reason, &l.CreatedAt); err != nil {
			return nil, err
		}
		if reason.Valid {
			l.Reason = reason.String
		}
		results = append(results, l)
	}
	return results, nil
}
