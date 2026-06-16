package assistant

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/selfevo-AI/meta-org/backend/internal/pkg/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/assistant/sessions", h.createSession)
	r.Get("/assistant/sessions", h.listSessions)
	r.Get("/assistant/sessions/{id}", h.getSession)
	r.Get("/assistant/sessions/{id}/steps", h.listSteps)
	r.Post("/assistant/sessions/{id}/runs", h.runSession)
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	actorID, actorType, ok := authenticatedActor(w, r)
	if !ok {
		return
	}
	var input CreateSessionInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.CreateSession(r.Context(), actorID, actorType, input)
	writeResult(w, http.StatusCreated, result, err)
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	actorID, actorType, ok := authenticatedActor(w, r)
	if !ok {
		return
	}
	result, err := h.service.ListSessions(r.Context(), actorID, actorType, r.URL.Query().Get("module_key"), queryLimit(r))
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	actorID, actorType, ok := authenticatedActor(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	result, err := h.service.GetSession(r.Context(), id, actorID, actorType)
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) listSteps(w http.ResponseWriter, r *http.Request) {
	actorID, actorType, ok := authenticatedActor(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	result, err := h.service.ListSteps(r.Context(), id, actorID, actorType, queryLimit(r))
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) runSession(w http.ResponseWriter, r *http.Request) {
	actorID, actorType, ok := authenticatedActor(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var input RunInput
	if !decodeJSON(w, r, &input) {
		return
	}
	events, err := h.service.Run(r.Context(), id, actorID, actorType, input)
	if err != nil {
		writeJSON(w, statusFromError(err), map[string]string{"error": err.Error()})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	writeSSE(w, "lifecycle", map[string]any{"status": StatusRunning, "session_id": id.String()})
	flusher.Flush()
	for event := range events {
		name := event.Type
		if name == "" {
			name = "message"
		}
		writeSSE(w, name, event)
		flusher.Flush()
	}
}

func authenticatedActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, string, bool) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return uuid.Nil, "", false
	}
	id, err := uuid.Parse(user.ID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authenticated user"})
		return uuid.Nil, "", false
	}
	actorType := user.Type
	if actorType == "ai" {
		actorType = "ai_agent"
	}
	return id, actorType, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return false
	}
	return true
}

func parseID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return uuid.Nil, false
	}
	return id, true
}

func queryLimit(r *http.Request) int {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	return limit
}

func writeResult(w http.ResponseWriter, successStatus int, payload any, err error) {
	if err != nil {
		writeJSON(w, statusFromError(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, successStatus, payload)
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("assistant writeJSON error: %v", err)
	}
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"encode stream event failed"}`)
		event = "error"
	}
	if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
		log.Printf("assistant stream write error: %v", err)
		return
	}
	if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
		log.Printf("assistant stream write error: %v", err)
	}
}
