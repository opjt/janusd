package watcher

import "karden/internal/domain"

// Annotation keys that Janusd reads from Pods
const (
	AnnotationInject       = "karden.io/inject"
	AnnotationType         = "karden.io/type"
	AnnotationSecretName   = "karden.io/secret-name"
	AnnotationDBType       = "karden.io/db-type"
	AnnotationDBHost       = "karden.io/db-host"
	AnnotationDBPort       = "karden.io/db-port"
	AnnotationRotationDays = "karden.io/rotation-days"
)

// defaultDBPort returns the well-known(잘 알려진) port for each DB type.
func defaultDBPort(dbType domain.DBType) int {
	switch dbType {
	case domain.DBTypePostgres:
		return 5432
	case domain.DBTypeMySQL:
		return 3306
	case domain.DBTypeMongoDB:
		return 27017
	default:
		return 0
	}
}
