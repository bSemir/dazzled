package incidents

import (
	"errors"
	"log/slog"
	"net/http"

	"dazzled/backend/api/rest/v1/handlers/response"
	"dazzled/backend/internal/domain"
	"dazzled/backend/internal/pkg/services/incidents"
)

type incidentRequest struct {
	VehicleType string `json:"vehicle_type"`
	Brand       string `json:"brand"`
	Location    string `json:"location"`
	Severity    int    `json:"severity"`
	LocalHour   *int   `json:"local_hour"`
}

func (h *APIHandler) HandleIncidents(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		h.GetCount(writer, request)
	case http.MethodPost:
		h.Create(writer, request)
	default:
		writer.Header().Set("Allow", "GET, POST")
		response.WriteError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (h *APIHandler) GetCount(writer http.ResponseWriter, request *http.Request) {
	count, err := h.service.Count(request.Context())
	if err != nil {
		slog.ErrorContext(request.Context(), "read incident count failed", "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "internal_error", "The incident count is unavailable.")
		return
	}
	response.WriteJSON(writer, http.StatusOK, map[string]int{"count": count})
}

func (h *APIHandler) Create(writer http.ResponseWriter, request *http.Request) {
	var input incidentRequest
	if err := response.DecodeJSON(writer, request, &input); err != nil {
		response.WriteDecodeError(writer, err)
		return
	}

	id, err := h.service.Create(request.Context(), domain.Incident{
		VehicleType: input.VehicleType,
		Brand:       input.Brand,
		Location:    input.Location,
		Severity:    input.Severity,
		LocalHour:   input.LocalHour,
	})
	if err != nil {
		if errors.Is(err, incidents.ErrInvalidIncident) {
			response.WriteError(writer, http.StatusBadRequest, "invalid_incident", "Choose a vehicle type and brand, use a location up to 100 characters, set severity from 1 to 5, and choose a local hour from 0 to 23.")
			return
		}
		slog.ErrorContext(request.Context(), "create incident failed", "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "internal_error", "The incident could not be saved.")
		return
	}
	response.WriteJSON(writer, http.StatusCreated, map[string]any{"id": id, "ok": true})
}

func (h *APIHandler) GetStatistics(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		response.WriteError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	statistics, err := h.service.Statistics(request.Context(), request.URL.Query().Get("period"))
	if err != nil {
		if errors.Is(err, incidents.ErrInvalidPeriod) {
			response.WriteError(writer, http.StatusBadRequest, "invalid_period", "Period must be today, week, or all.")
			return
		}
		slog.ErrorContext(request.Context(), "read statistics failed", "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "internal_error", "Statistics are unavailable.")
		return
	}
	response.WriteJSON(writer, http.StatusOK, statistics)
}
