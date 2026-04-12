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
		INSERT INTO audit_logs (namespace, secret_name, action, actor, result, reason)
		VALUES (?, ?, ?, ?, ?, ?)
	`, log.Namespace, log.SecretName, log.Action, log.Actor, log.Result, log.Reason)
	return err
}

func (r *AuditRepository) List(ctx context.Context, namespace, secretName string) ([]*audit.AuditLog, error) {
	query := `
		SELECT id, namespace, secret_name, action, actor, result, reason, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []any{}

	if namespace != "" {
		query += " AND namespace = ?"
		args = append(args, namespace)
	}
	if secretName != "" {
		query += " AND secret_name = ?"
		args = append(args, secretName)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*audit.AuditLog, 0)
	for rows.Next() {
		l := &audit.AuditLog{}
		var reason sql.NullString
		if err := rows.Scan(&l.ID, &l.Namespace, &l.SecretName, &l.Action, &l.Actor, &l.Result, &reason, &l.CreatedAt); err != nil {
			return nil, err
		}
		if reason.Valid {
			l.Reason = reason.String
		}
		results = append(results, l)
	}
	return results, nil
}
