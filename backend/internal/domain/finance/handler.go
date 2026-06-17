package finance

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
	"github.com/selfevo-AI/meta-org/backend/internal/pkg/dberrors"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/finance/webhooks/{adapterID}", h.receiveWebhook)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/finance/adapters", h.createAdapter)
	r.Get("/finance/adapters", h.listAdapters)
	r.Patch("/finance/adapters/{id}", h.updateAdapter)
	r.Post("/finance/adapters/{id}/test", h.testAdapter)
	r.Post("/finance/export-batches", h.createExportBatch)
	r.Get("/finance/export-batches", h.listExportBatches)
	r.Get("/finance/export-batches/{id}", h.getExportBatch)
	r.Post("/finance/export-batches/{id}/submit", h.submitExportBatch)
	r.Get("/finance/reconciliation", h.listReconciliation)
}

func (h *Handler) createAdapter(w http.ResponseWriter, r *http.Request) {
	var input CreateAdapterInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.CreateAdapter(r.Context(), input)
	writeResult(w, http.StatusCreated, result, err)
}

func (h *Handler) listAdapters(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListAdapters(r.Context(), queryLimit(r))
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) updateAdapter(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var input UpdateAdapterInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.UpdateAdapter(r.Context(), id, input)
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) testAdapter(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	err := h.service.TestAdapter(r.Context(), id)
	writeResult(w, http.StatusOK, map[string]string{"status": "ok"}, err)
}

func (h *Handler) createExportBatch(w http.ResponseWriter, r *http.Request) {
	var input CreateExportBatchInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.CreateExportBatch(r.Context(), input)
	writeResult(w, http.StatusCreated, result, err)
}

func (h *Handler) listExportBatches(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListExportBatches(r.Context(), queryLimit(r))
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) getExportBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	result, err := h.service.GetExportBatch(r.Context(), id)
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) submitExportBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	result, err := h.service.SubmitExportBatch(r.Context(), id)
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) receiveWebhook(w http.ResponseWriter, r *http.Request) {
	adapterID, ok := parseID(w, r, "adapterID")
	if !ok {
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	result, err := h.service.ReceiveWebhook(
		r.Context(),
		adapterID,
		body,
		firstHeader(r, "X-Meta-Org-Signature", "X-Hub-Signature-256"),
		r.Header.Get("Authorization"),
	)
	writeResult(w, http.StatusOK, result, err)
}

func (h *Handler) listReconciliation(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListReconciliation(r.Context(), queryLimit(r))
	writeResult(w, http.StatusOK, result, err)
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
	case dberrors.IsUniqueViolation(err):
		return http.StatusConflict
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
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
		log.Printf("finance writeJSON error: %v", err)
	}
}

func firstHeader(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := r.Header.Get(name); value != "" {
			return value
		}
	}
	return ""
}
