package api

import (
	"karden/internal/adapter/k8s"
	"karden/internal/domain/workload"
	"net/http"
)

type Handler struct {
	repo  workload.Repository
	store *k8s.SecretStore
}

func NewHandler(repo workload.Repository, store *k8s.SecretStore) *Handler {
	return &Handler{repo: repo, store: store}
}

// secretResponse is the API shape for a managed secret.
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

// GET /api/v1/secrets
func (h *Handler) listSecrets(w http.ResponseWriter, r *http.Request) {
	workloads, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workloads")
		return
	}

	type key struct{ name, namespace string }
	index := map[key]*secretResponse{}

	for _, wl := range workloads {
		k := key{wl.SecretName, wl.Namespace}
		if _, ok := index[k]; !ok {
			var lastRotatedAt *string
			if wl.LastRotatedAt != nil {
				s := wl.LastRotatedAt.UTC().Format("2006-01-02T15:04:05Z")
				lastRotatedAt = &s
			}
			index[k] = &secretResponse{
				Name:          wl.SecretName,
				Namespace:     wl.Namespace,
				Type:          string(wl.Type),
				DBType:        string(wl.DBType),
				RotationDays:  wl.RotationDays,
				LastRotatedAt: lastRotatedAt,
				Status:        string(wl.Status),
				Pods:          []string{},
			}
		}
		index[k].Pods = append(index[k].Pods, wl.PodName)
	}

	result := make([]*secretResponse, 0, len(index))
	for _, v := range index {
		result = append(result, v)
	}

	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/secrets/{namespace}/{name}
func (h *Handler) getSecret(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	workloads, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workloads")
		return
	}

	var resp *secretResponse
	for _, wl := range workloads {
		if wl.SecretName != name || wl.Namespace != namespace {
			continue
		}
		if resp == nil {
			var lastRotatedAt *string
			if wl.LastRotatedAt != nil {
				s := wl.LastRotatedAt.UTC().Format("2006-01-02T15:04:05Z")
				lastRotatedAt = &s
			}
			resp = &secretResponse{
				Name:          wl.SecretName,
				Namespace:     wl.Namespace,
				Type:          string(wl.Type),
				DBType:        string(wl.DBType),
				RotationDays:  wl.RotationDays,
				LastRotatedAt: lastRotatedAt,
				Status:        string(wl.Status),
				Pods:          []string{},
			}
		}
		resp.Pods = append(resp.Pods, wl.PodName)
	}

	if resp == nil {
		writeError(w, http.StatusNotFound, "secret not found")
		return
	}

	// K8s에서 실제 data 조회
	data, err := h.store.GetSecretData(r.Context(), namespace, name)
	if err == nil {
		resp.Data = data
	}

	writeJSON(w, http.StatusOK, resp)
}

// POST /api/v1/secrets/{namespace}/{name}/rotate
func (h *Handler) rotateSecret(w http.ResponseWriter, r *http.Request) {
	// TODO: rotation login implement
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "rotation triggered"})
}
