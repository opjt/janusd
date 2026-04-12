package workload

import "time"

type Type string

const (
	TypeDatabase Type = "database"
	TypeSecret   Type = "secret"
	TypeManual   Type = "manual"
)

type DBType string

const (
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "postgres"
	DBTypeMongoDB  DBType = "mongodb"
	DBTypeRedis    DBType = "redis"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// ManagedWorkload is a Pod whose secret lifecycle is managed by Karden.
type ManagedWorkload struct {
	ID            int
	PodName       string
	Namespace     string
	SecretName    string
	Type          Type
	DBType        DBType
	DBService     string // K8s Service name; resolved to {name}.{namespace} at runtime
	RotationDays  int
	LastRotatedAt *time.Time
	Status        Status
	CreatedAt     time.Time
}

