package api

import (
	"karden/internal/domain/audit"
	"karden/internal/domain/workload"
	"net/http"
)

type Handler struct {
	secretSvc workload.Service
	auditRepo audit.Repository
}

func NewHandler(secretSvc workload.Service, auditRepo audit.Repository) *Handler {
	return &Handler{secretSvc: secretSvc, auditRepo: auditRepo}
}

// secretResponse is the API DTO for a managed secret.
type secretResponse struct {
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	Type          string            `json:"type"`
	DBType        string            `json:"db_type,omitempty"`
	RotationDays  int               `json:"rotation_days"`
	LastRotatedAt *string           `json:"last_rotated_at"`
	Status        string            `json:"status"`
	Pods          []string          `json:"pods"`
	Data          map[string]string `json:"data,omitempty"`
}

func toResponse(v *workload.SecretView) *secretResponse {
	var lastRotatedAt *string
	if v.LastRotatedAt != nil {
		s := v.LastRotatedAt.UTC().Format("2006-01-02T15:04:05Z")
		lastRotatedAt = &s
	}
	return &secretResponse{
		Name:          v.Name,
		Namespace:     v.Namespace,
		Type:          string(v.Type),
		DBType:        string(v.DBType),
		RotationDays:  v.RotationDays,
		LastRotatedAt: lastRotatedAt,
		Status:        string(v.Status),
		Pods:          v.Pods,
		Data:          v.Data,
	}
}

// GET /api/v1/secrets
func (h *Handler) listSecrets(w http.ResponseWriter, r *http.Request) {
	secrets, err := h.secretSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list secrets")
		return
	}

	result := make([]*secretResponse, len(secrets))
	for i, s := range secrets {
		result[i] = toResponse(s)
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/secrets/{namespace}/{name}
func (h *Handler) getSecret(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	secret, err := h.secretSvc.Get(r.Context(), namespace, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get secret")
		return
	}
	if secret == nil {
		writeError(w, http.StatusNotFound, "secret not found")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(secret))
}

// POST /api/v1/secrets/{namespace}/{name}/rotate
func (h *Handler) rotateSecret(w http.ResponseWriter, r *http.Request) {
	// TODO: implement rotation logic
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "rotation triggered"})
}

// auditResponse is the API DTO for an audit log entry.
type auditResponse struct {
	ID        int    `json:"id"`
	TargetID  int    `json:"target_id"`
	Action    string `json:"action"`
	Actor     string `json:"actor"`
	Result    string `json:"result"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

// GET /api/v1/audit?namespace=&secret=
func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	secretName := r.URL.Query().Get("secret")

	logs, err := h.auditRepo.List(r.Context(), namespace, secretName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	result := make([]*auditResponse, len(logs))
	for i, l := range logs {
		result[i] = &auditResponse{
			ID:        l.ID,
			TargetID:  l.TargetID,
			Action:    string(l.Action),
			Actor:     l.Actor,
			Result:    string(l.Result),
			Reason:    l.Reason,
			CreatedAt: l.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, result)
}
