package organization

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/organizations", h.createOrganization)
	r.Get("/organizations/{id}", h.getOrganization)
	r.Post("/muvrs", h.createMVRU)
	r.Get("/muvrs/{id}", h.getMVRU)
	r.Patch("/muvrs/{id}/status", h.updateMVRUStatus)
	r.Post("/muvrs/{id}/members", h.addMember)
	r.Delete("/muvrs/{id}/members", h.removeMember)
	r.Post("/relationships", h.createRelationship)
}

func (h *Handler) createOrganization(w http.ResponseWriter, r *http.Request) {
	var input CreateOrganizationInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	org, err := h.service.CreateOrganization(r.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrValidation) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (h *Handler) getOrganization(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	org, err := h.service.GetOrganization(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "organization not found"})
		return
	}
	chart, err := h.service.GetOrgChart(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch org chart"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"organization": org, "chart": chart})
}

func (h *Handler) createMVRU(w http.ResponseWriter, r *http.Request) {
	var input CreateMVRUInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	mvru, err := h.service.CreateMVRU(r.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrValidation) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, mvru)
}

func (h *Handler) getMVRU(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	mvru, err := h.service.GetMVRU(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "mvru not found"})
		return
	}
	writeJSON(w, http.StatusOK, mvru)
}

func (h *Handler) updateMVRUStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	switch req.Status {
	case "active":
		err = h.service.ActivateMVRU(r.Context(), id)
	case "evaluating":
		err = h.service.EvaluateMVRU(r.Context(), id)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	mvruID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mvru id"})
		return
	}
	var req struct {
		UserID  *string `json:"user_id"`
		AgentID *string `json:"agent_id"`
		RoleID  string  `json:"role_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	roleUUID, err := uuid.Parse(req.RoleID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}
	var userUUID, agentUUID *uuid.UUID
	if req.UserID != nil {
		u, err := uuid.Parse(*req.UserID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
			return
		}
		userUUID = &u
	}
	if req.AgentID != nil {
		a, err := uuid.Parse(*req.AgentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
			return
		}
		agentUUID = &a
	}
	if err := h.service.AddMember(r.Context(), mvruID, roleUUID, userUUID, agentUUID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrValidation) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "member added"})
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	mvruID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mvru id"})
		return
	}
	var req struct {
		UserID  *string `json:"user_id"`
		AgentID *string `json:"agent_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	var userUUID, agentUUID *uuid.UUID
	if req.UserID != nil {
		u, err := uuid.Parse(*req.UserID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
			return
		}
		userUUID = &u
	}
	if req.AgentID != nil {
		a, err := uuid.Parse(*req.AgentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
			return
		}
		agentUUID = &a
	}
	if err := h.service.RemoveMember(r.Context(), mvruID, userUUID, agentUUID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "member removed"})
}

func (h *Handler) createRelationship(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceMVRUID string         `json:"source_mvru_id"`
		TargetMVRUID string         `json:"target_mvru_id"`
		RelType      string         `json:"rel_type"`
		Config       map[string]any `json:"config,omitempty"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	src, err := uuid.Parse(req.SourceMVRUID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid source mvru id"})
		return
	}
	tgt, err := uuid.Parse(req.TargetMVRUID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target mvru id"})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	rel, err := h.service.CreateRelationship(r.Context(), src, tgt, req.RelType, req.Config)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
