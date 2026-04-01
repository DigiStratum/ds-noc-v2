package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DigiStratum/ds-noc-v2/backend/pkg/tenant"
)

func TestListTenants_NoTenant(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/tenant", nil)
	w := httptest.NewRecorder()

	ListTenants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response TenantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.TenantID != "" {
		t.Errorf("expected empty tenant_id, got %q", response.TenantID)
	}
	if !response.IsPersonal {
		t.Error("expected is_personal=true when no tenant")
	}
}

func TestListTenants_WithTenant(t *testing.T) {
	tenantVal, _ := tenant.Parse("org:acme")

	req := httptest.NewRequest("GET", "/api/tenant", nil)
	ctx := tenant.SetTenant(req.Context(), tenantVal)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ListTenants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response TenantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.TenantID != "org:acme" {
		t.Errorf("expected tenant_id='org:acme', got %q", response.TenantID)
	}
	if response.IsPersonal {
		t.Error("expected is_personal=false when tenant is set")
	}
}

// withTenant adds a tenant to the context for testing.
func withTenant(ctx context.Context, tenantID string) context.Context {
	tenantVal, err := tenant.Parse(tenantID)
	if err != nil {
		return ctx
	}
	return tenant.SetTenant(ctx, tenantVal)
}
