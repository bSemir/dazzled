package incidents

import (
	"net/http"

	inbound "dazzled/backend/api/rest/v1/ports/inbound/incidents"
)

type APIHandler struct {
	service inbound.Service
}

func New(service inbound.Service) *APIHandler {
	return &APIHandler{service: service}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/incidents", h.HandleIncidents)
	mux.HandleFunc("/api/stats", h.GetStatistics)
}
