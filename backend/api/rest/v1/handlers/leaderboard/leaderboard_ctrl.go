// Package leaderboard adapts game-session and score requests to the leaderboard service.
package leaderboard

import (
	"errors"
	"log/slog"
	"net/http"

	"dazzled/backend/api/rest/v1/handlers/response"
	"dazzled/backend/internal/domain"
	"dazzled/backend/internal/pkg/services/leaderboard"
)

type scoreRequest struct {
	Username  string  `json:"username"`
	Distance  float64 `json:"distance"`
	GameToken string  `json:"game_token"`
}

func (h *APIHandler) StartGame(writer http.ResponseWriter, request *http.Request) {
	token, err := h.service.StartGame(request.Context())
	if err != nil {
		slog.ErrorContext(request.Context(), "start game session failed", "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "internal_error", "A verified game could not be started.")
		return
	}
	response.WriteJSON(writer, http.StatusCreated, map[string]string{"game_token": token})
}

func (h *APIHandler) Handle(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		h.GetTop(writer, request)
	case http.MethodPost:
		h.Submit(writer, request)
	default:
		writer.Header().Set("Allow", "GET, POST")
		response.WriteError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (h *APIHandler) GetTop(writer http.ResponseWriter, request *http.Request) {
	scores, err := h.service.Top(request.Context())
	if err != nil {
		slog.ErrorContext(request.Context(), "read leaderboard failed", "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "internal_error", "The scoreboard is unavailable.")
		return
	}
	response.WriteJSON(writer, http.StatusOK, map[string]any{"scores": scores})
}

func (h *APIHandler) Submit(writer http.ResponseWriter, request *http.Request) {
	var input scoreRequest
	if err := response.DecodeJSON(writer, request, &input); err != nil {
		response.WriteDecodeError(writer, err)
		return
	}
	if err := h.service.Submit(request.Context(), domain.Score{Username: input.Username, Distance: input.Distance}, input.GameToken); err != nil {
		if errors.Is(err, leaderboard.ErrInvalidScore) {
			response.WriteError(writer, http.StatusBadRequest, "invalid_score", "Name must be 2–20 letters, numbers, spaces, _ or -, and score must be between 0 and 10000 km.")
			return
		}
		if errors.Is(err, leaderboard.ErrInvalidGameToken) {
			response.WriteError(writer, http.StatusBadRequest, "invalid_game_token", "This game session is expired or has already been used. Start a new run.")
			return
		}
		if errors.Is(err, leaderboard.ErrScoreExceedsElapsedTime) {
			response.WriteError(writer, http.StatusUnprocessableEntity, "score_exceeds_elapsed_time", "That distance is not possible in the time this run was active.")
			return
		}
		slog.ErrorContext(request.Context(), "create leaderboard score failed", "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "internal_error", "The score could not be saved.")
		return
	}
	response.WriteJSON(writer, http.StatusCreated, map[string]bool{"ok": true})
}
