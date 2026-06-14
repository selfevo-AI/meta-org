package gateway

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/harness-org/backend/internal/domain/capability"
	"github.com/harness-org/backend/internal/domain/dashboard"
	"github.com/harness-org/backend/internal/domain/evolution"
	"github.com/harness-org/backend/internal/domain/governance"
	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/domain/layer"
	"github.com/harness-org/backend/internal/domain/observability"
	"github.com/harness-org/backend/internal/domain/organization"
	"github.com/harness-org/backend/internal/domain/project"
	"github.com/harness-org/backend/internal/domain/verification"
	"github.com/harness-org/backend/internal/domain/workflow"
	"github.com/harness-org/backend/internal/pkg/middleware"
)

type Dependencies struct {
	JWTSecret            string
	IdentityHandler      *identity.Handler
	OrganizationHandler  *organization.Handler
	LayerHandler         *layer.Handler
	CapabilityHandler    *capability.Handler
	DashboardHandler     *dashboard.Handler
	WorkflowHandler      *workflow.Handler
	ProjectHandler       *project.Handler
	ObservabilityHandler *observability.Handler
	VerificationHandler  *verification.Handler
	GovernanceHandler    *governance.Handler
	EvolutionHandler     *evolution.Handler
}

func RegisterRoutes(r *chi.Mux, deps *Dependencies) {
	if deps == nil {
		panic("gateway.RegisterRoutes: deps must not be nil")
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
		if deps.IdentityHandler != nil {
			deps.IdentityHandler.RegisterPublicRoutes(r)
		}
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(deps.JWTSecret))
			if deps.IdentityHandler != nil {
				deps.IdentityHandler.RegisterProtectedRoutes(r)
			}
			if deps.OrganizationHandler != nil {
				deps.OrganizationHandler.RegisterRoutes(r)
			}
			if deps.LayerHandler != nil {
				deps.LayerHandler.RegisterRoutes(r)
			}
			if deps.CapabilityHandler != nil {
				deps.CapabilityHandler.RegisterRoutes(r)
			}
			if deps.DashboardHandler != nil {
				deps.DashboardHandler.RegisterRoutes(r)
			}
			if deps.WorkflowHandler != nil {
				deps.WorkflowHandler.RegisterRoutes(r)
			}
			if deps.ProjectHandler != nil {
				deps.ProjectHandler.RegisterRoutes(r)
			}
			if deps.VerificationHandler != nil {
				deps.VerificationHandler.RegisterRoutes(r)
			}
			if deps.ObservabilityHandler != nil {
				deps.ObservabilityHandler.RegisterRoutes(r)
			}
			if deps.GovernanceHandler != nil {
				deps.GovernanceHandler.RegisterRoutes(r)
			}
			if deps.EvolutionHandler != nil {
				deps.EvolutionHandler.RegisterRoutes(r)
			}
		})
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Printf("health check write error: %v", err)
	}
}
