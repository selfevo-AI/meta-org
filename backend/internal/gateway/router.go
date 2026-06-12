package gateway

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/harness-org/backend/internal/domain/identity"
)

type Dependencies struct {
	IdentityHandler *identity.Handler
}

func RegisterRoutes(r *chi.Mux, deps *Dependencies) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
		if deps != nil && deps.IdentityHandler != nil {
			deps.IdentityHandler.RegisterRoutes(r)
		}
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Printf("health check write error: %v", err)
	}
}
