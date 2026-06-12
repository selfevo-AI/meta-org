package capability

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo   *Repository
	router *Router
}

func NewHandler(repo *Repository, router *Router) *Handler {
	return &Handler{repo: repo, router: router}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/capabilities", h.createCapability)
	r.Get("/capabilities", h.listCapabilities)
	r.Get("/capabilities/{id}", h.getCapability)
	r.Post("/capabilities/match", h.matchCapability)
	r.Post("/bindings", h.bindCapability)
	r.Delete("/bindings/{id}", h.unbindCapability)
	r.Get("/bindings", h.listBindings)
}

func (h *Handler) createCapability(w http.ResponseWriter, r *http.Request) {
	var input CreateCapabilityInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	cap, err := h.repo.CreateCapability(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, cap)
}

func (h *Handler) listCapabilities(w http.ResponseWriter, r *http.Request) {
	caps, err := h.repo.ListCapabilities(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, caps)
}

func (h *Handler) getCapability(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	cap, err := h.repo.GetCapabilityByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "capability not found"})
		return
	}
	writeJSON(w, http.StatusOK, cap)
}

func (h *Handler) matchCapability(w http.ResponseWriter, r *http.Request) {
	var req MatchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	results, err := h.router.MatchTask(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) bindCapability(w http.ResponseWriter, r *http.Request) {
	var input BindCapabilityInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	binding, err := h.repo.BindCapability(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, binding)
}

func (h *Handler) unbindCapability(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.UnbindCapability(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unbound"})
}

func (h *Handler) listBindings(w http.ResponseWriter, r *http.Request) {
	mvruStr := r.URL.Query().Get("mvru_id")
	if mvruStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mvru_id query param required"})
		return
	}
	mvruID, err := uuid.Parse(mvruStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mvru_id"})
		return
	}
	bindings, err := h.repo.ListBoundCapabilities(r.Context(), mvruID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, bindings)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
