package api

import (
	"net/http"
)

func NewServer(addr string, h *Handler) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/secrets", h.listSecrets)
	mux.HandleFunc("GET /api/v1/secrets/{namespace}/{name}", h.getSecret)
	mux.HandleFunc("POST /api/v1/secrets/{namespace}/{name}/rotate", h.rotateSecret)

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
