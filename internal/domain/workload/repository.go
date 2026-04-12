package workload

import (
	"context"
	"time"
)

// Repository is the port for persisting ManagedWorkloads.
type Repository interface {
	Upsert(ctx context.Context, w *ManagedWorkload) (int64, error)
	List(ctx context.Context) ([]*ManagedWorkload, error)
	SetInactive(ctx context.Context, podName, namespace string) error
	UpdateLastRotated(ctx context.Context, podName, namespace string, t time.Time) error
}
