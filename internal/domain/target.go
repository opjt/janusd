package domain

import "time"

// Type represents what kind of secret Janusd manages
type Type string

const (
	TypeDatabase Type = "database"
	TypeSecret   Type = "secret"
	TypeManual   Type = "manual"
)

// DBType represents the database engine
type DBType string

const (
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "postgres"
	DBTypeMongoDB  DBType = "mongodb"
)

// Status represents whether a target is being actively managed
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// ManagedTarget is a Pod that Janusd is managing
type ManagedTarget struct {
	ID            int
	PodName       string
	Namespace     string
	SecretName    string
	Type          Type
	DBType        DBType
	DBHost        string
	DBPort        int
	RotationDays  int
	LastRotatedAt *time.Time
	Status        Status
	CreatedAt     time.Time
}
