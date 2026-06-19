package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type tenantContextKey string

const TenantContextKey tenantContextKey = "tenant"

var (
	ErrTenantRequired     = errors.New("organization context is required")
	ErrOnboardingRequired = errors.New("onboarding is required")
	ErrTenantForbidden    = errors.New("organization access forbidden")
)

type TenantContext struct {
	Mode             string
	UserID           uuid.UUID
	OrganizationID   *uuid.UUID
	IsPlatformAdmin  bool
	PlatformRole     string
	MembershipID     *uuid.UUID
	AuthorityTier    string
	EnabledModules   map[string]bool
	OnboardingStatus string
}

type TenantResolver interface {
	ResolveTenant(ctx context.Context, user AuthenticatedUser, requestedOrganizationID string) (*TenantContext, error)
}

func TenantFromContext(ctx context.Context) (*TenantContext, bool) {
	tenant, ok := ctx.Value(TenantContextKey).(*TenantContext)
	return tenant, ok
}

func TenantMiddleware(resolver TenantResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver == nil {
				writeTenantError(w, http.StatusInternalServerError, "tenant resolver is not configured")
				return
			}
			user, ok := UserFromContext(r.Context())
			if !ok {
				writeTenantError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			tenant, err := resolver.ResolveTenant(r.Context(), user, r.Header.Get("X-Organization-ID"))
			if err != nil {
				switch {
				case errors.Is(err, ErrOnboardingRequired):
					writeTenantError(w, http.StatusPreconditionRequired, "onboarding_required")
				case errors.Is(err, ErrTenantRequired):
					writeTenantError(w, http.StatusBadRequest, "organization_required")
				case errors.Is(err, ErrTenantForbidden):
					writeTenantError(w, http.StatusForbidden, "organization_forbidden")
				default:
					writeTenantError(w, http.StatusInternalServerError, "tenant_resolution_failed")
				}
				return
			}
			ctx := context.WithValue(r.Context(), TenantContextKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeTenantError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
