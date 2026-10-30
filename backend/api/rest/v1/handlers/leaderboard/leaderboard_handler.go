package leaderboard

import (
	"net/http"

	inbound "dazzled/backend/api/rest/v1/ports/inbound/leaderboard"
)

type APIHandler struct {
	service inbound.Service
}

func New(service inbound.Service) *APIHandler {
	return &APIHandler{service: service}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/leaderboard", h.Handle)
	mux.HandleFunc("POST /api/leaderboard/start", h.StartGame)
}
