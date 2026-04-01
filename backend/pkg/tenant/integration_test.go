package tenant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DigiStratum/ds-noc-v2/backend/pkg/tenant"
)

// TestCrossTenantIsolation_Integration tests the full middleware chain
// to verify cross-tenant data access is blocked.
func TestCrossTenantIsolation_Integration(t *testing.T) {
	// Simulate a protected handler that returns tenant-scoped data
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestTenant, err := tenant.RequireTenant(ctx)
		if err != nil {
			t.Errorf("Handler should have tenant: %v", err)
			http.Error(w, "no tenant", http.StatusInternalServerError)
			return
		}

		// Simulate returning data with tenant prefix in PK
		pk := tenant.BuildPK(requestTenant, "ISSUE", "123")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"pk":     pk,
			"tenant": requestTenant.String(),
		})
	})

	config := tenant.DefaultConfig()
	handler := tenant.Middleware(config)(protectedHandler)

	t.Run("same tenant access succeeds", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/issues/123", nil)
		req.Header.Set("X-Tenant-ID", "user:alice")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["tenant"] != "user:alice" {
			t.Errorf("Wrong tenant in response: %s", resp["tenant"])
		}
		if resp["pk"] != "TENANT#user:alice#ISSUE#123" {
			t.Errorf("Wrong pk in response: %s", resp["pk"])
		}
	})

	t.Run("missing tenant is rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/issues/123", nil)
		// No X-Tenant-ID header
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rec.Code)
		}
	})

	t.Run("different tenants get different PKs", func(t *testing.T) {
		// Request 1: Alice's tenant
		req1 := httptest.NewRequest("GET", "/api/issues/123", nil)
		req1.Header.Set("X-Tenant-ID", "user:alice")
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)

		var resp1 map[string]string
		if err := json.NewDecoder(rec1.Body).Decode(&resp1); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Request 2: Bob's tenant (same issue ID)
		req2 := httptest.NewRequest("GET", "/api/issues/123", nil)
		req2.Header.Set("X-Tenant-ID", "user:bob")
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)

		var resp2 map[string]string
		if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Same issue ID should result in different PKs
		if resp1["pk"] == resp2["pk"] {
			t.Errorf("Same issue ID should produce different PKs for different tenants: %s vs %s", resp1["pk"], resp2["pk"])
		}
	})
}

// TestPKValidation_Integration tests that PK validation catches cross-tenant attempts
func TestPKValidation_Integration(t *testing.T) {
	aliceTenant := tenant.Tenant{Type: tenant.TenantTypeUser, ID: "alice"}
	bobTenant := tenant.Tenant{Type: tenant.TenantTypeUser, ID: "bob"}

	// Alice's repository
	aliceRepo := tenant.NewBaseRepository(aliceTenant)

	// Alice creates an issue
	aliceIssuePK := aliceRepo.PK("ISSUE", "123")

	// Bob's repository
	bobRepo := tenant.NewBaseRepository(bobTenant)

	// Bob should NOT be able to validate Alice's PK
	if err := bobRepo.ValidatePK(aliceIssuePK); err == nil {
		t.Error("Bob should not be able to access Alice's data")
	}

	// Alice should be able to validate her own PK
	if err := aliceRepo.ValidatePK(aliceIssuePK); err != nil {
		t.Errorf("Alice should be able to access her own data: %v", err)
	}
}

// TestTenantContextPropagation tests that tenant flows through context correctly
func TestTenantContextPropagation(t *testing.T) {
	// Service function that requires tenant
	serviceFn := func(ctx context.Context) (string, error) {
		tnt, err := tenant.RequireTenant(ctx)
		if err != nil {
			return "", err
		}
		return tnt.String(), nil
	}

	// Handler that calls service
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, err := serviceFn(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(result))
	})

	config := tenant.DefaultConfig()
	wrappedHandler := tenant.Middleware(config)(handler)

	t.Run("tenant propagates to service layer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Tenant-ID", "org:my-org")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rec.Code)
		}

		if rec.Body.String() != "org:my-org" {
			t.Errorf("Expected tenant 'org:my-org', got %s", rec.Body.String())
		}
	})

	t.Run("missing tenant causes service failure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		// Call handler without middleware (simulating internal bug)
		handler.ServeHTTP(rec, req)

		// Should fail because no tenant in context
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 without tenant, got %d", rec.Code)
		}
	})
}

// TestPublicRoutesBypass tests that public routes don't require tenant
func TestPublicRoutesBypass(t *testing.T) {
	publicPaths := []string{"/api/health", "/api/discovery"}

	config := tenant.DefaultConfig()
	middleware := tenant.PublicRoutes(publicPaths, tenant.Middleware(config))

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if tenant is present (shouldn't be for public routes)
		tnt := tenant.GetTenant(r.Context())
		if tnt.IsZero() {
			_, _ = w.Write([]byte("public"))
		} else {
			_, _ = w.Write([]byte("tenant:" + tnt.String()))
		}
	}))

	t.Run("health check works without tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Health check should succeed without tenant: %d", rec.Code)
		}
		if rec.Body.String() != "public" {
			t.Errorf("Expected 'public', got %s", rec.Body.String())
		}
	})

	t.Run("protected route requires tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/issues", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Protected route should require tenant: %d", rec.Code)
		}
	})
}
