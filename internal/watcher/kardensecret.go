package watcher

import "karden/internal/domain/workload"

// kardenSecretSpec mirrors the CRD spec fields.
type kardenSecretSpec struct {
	Type         workload.Type   `json:"type"`
	DBType       workload.DBType `json:"dbType,omitempty"`
	DBService    string          `json:"dbService,omitempty"`
	RotationDays int             `json:"rotationDays,omitempty"`
}

// kardenSecret is the watcher-internal representation of the CRD resource.
type kardenSecret struct {
	Name      string
	Namespace string
	Spec      kardenSecretSpec
}

// toWorkload converts a kardenSecret into a ManagedWorkload.
func (ks *kardenSecret) toWorkload() *workload.ManagedWorkload {
	days := ks.Spec.RotationDays
	if days == 0 {
		days = 30
	}
	return &workload.ManagedWorkload{
		PodName:      ks.Name,
		Namespace:    ks.Namespace,
		SecretName:   ks.Name,
		Type:         ks.Spec.Type,
		DBType:       ks.Spec.DBType,
		DBService:    ks.Spec.DBService,
		RotationDays: days,
		Status:       workload.StatusActive,
	}
}
