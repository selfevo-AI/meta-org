package gateway

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/harness-org/backend/internal/domain/capability"
	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/domain/layer"
	"github.com/harness-org/backend/internal/domain/organization"
)

type Dependencies struct {
	IdentityHandler     *identity.Handler
	OrganizationHandler *organization.Handler
	LayerHandler        *layer.Handler
	CapabilityHandler   *capability.Handler
}

func RegisterRoutes(r *chi.Mux, deps *Dependencies) {
	if deps == nil {
		panic("gateway.RegisterRoutes: deps must not be nil")
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
		deps.IdentityHandler.RegisterRoutes(r)
		if deps.OrganizationHandler != nil {
			deps.OrganizationHandler.RegisterRoutes(r)
		}
		if deps.LayerHandler != nil {
			deps.LayerHandler.RegisterRoutes(r)
		}
		if deps.CapabilityHandler != nil {
			deps.CapabilityHandler.RegisterRoutes(r)
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
