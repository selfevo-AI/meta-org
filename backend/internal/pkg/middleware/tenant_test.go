package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type staticTenantResolver struct {
	tenant *TenantContext
	err    error
}

func (r staticTenantResolver) ResolveTenant(ctx context.Context, user AuthenticatedUser, requestedOrganizationID string) (*TenantContext, error) {
	return r.tenant, r.err
}

func TestTenantMiddlewareAttachesSingleOrgTenant(t *testing.T) {
	orgID := uuid.New()
	baseTenant := &TenantContext{
		Mode:           "single_org",
		UserID:         uuid.New(),
		OrganizationID: &orgID,
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := TenantFromContext(r.Context())
		if !ok {
			t.Fatalf("tenant context missing")
		}
		if tenant.Mode != "single_org" || tenant.OrganizationID == nil || *tenant.OrganizationID != orgID {
			t.Fatalf("tenant = %#v, want single_org organization %s", tenant, orgID)
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := TenantMiddleware(staticTenantResolver{tenant: baseTenant})(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, AuthenticatedUser{
		ID:   uuid.New().String(),
		Type: "human",
		Name: "Tester",
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestTenantMiddlewareOnboardingRequired(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := TenantMiddleware(staticTenantResolver{err: ErrOnboardingRequired})(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, AuthenticatedUser{
		ID:   uuid.New().String(),
		Type: "human",
		Name: "Tester",
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionRequired {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPreconditionRequired)
	}
}
