package audit

import (
	"context"
	"time"
)

type Action string

const (
	ActionCreate    Action = "create"
	ActionRotate    Action = "rotate"
	ActionManualSet Action = "manual_set"
)

type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
)

// AuditLog is a record of every operation Karden performs on a secret.
type AuditLog struct {
	ID         int
	Namespace  string
	SecretName string
	Action     Action
	Actor      string
	Result     Result
	Reason     string
	CreatedAt  time.Time
}

// Repository is the port for persisting AuditLogs.
type Repository interface {
	Save(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, namespace, secretName string) ([]*AuditLog, error)
}
