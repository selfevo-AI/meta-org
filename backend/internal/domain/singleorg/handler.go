package singleorg

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/pkg/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterAuthenticatedRoutes(r chi.Router) {
	r.Get("/auth/me", h.me)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if user.Type != "human" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "human account required"})
		return
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load profile"})
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("singleorg write json: %v", err)
	}
}
