package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_ExtractFromHeader(t *testing.T) {
	config := DefaultConfig()

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := GetTenant(r.Context())
		if tenant.IsZero() {
			t.Error("Expected tenant in context")
		}
		if tenant.Type != TenantTypeUser || tenant.ID != "123" {
			t.Errorf("Got tenant %+v, want user:123", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Tenant-ID", "user:123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_MissingTenant(t *testing.T) {
	config := DefaultConfig()

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when tenant is missing")
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	errObj, _ := resp["error"].(map[string]any)
	if errObj["code"] != "TENANT_REQUIRED" {
		t.Errorf("Expected TENANT_REQUIRED error code, got %v", errObj["code"])
	}
}

func TestMiddleware_AllowAnonymous(t *testing.T) {
	config := DefaultConfig()
	config.AllowAnonymous = true

	called := false
	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenant := GetTenant(r.Context())
		if !tenant.IsZero() {
			t.Error("Expected no tenant in context for anonymous request")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler should be called when AllowAnonymous is true")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_InvalidTenantHeader(t *testing.T) {
	config := DefaultConfig()

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid tenant")
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Tenant-ID", "invalid-format")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_CustomExtractor(t *testing.T) {
	config := DefaultConfig()
	config.TenantExtractor = func(r *http.Request) (Tenant, bool) {
		// Simulate extracting from session
		return Tenant{Type: TenantTypeOrg, ID: "custom-org"}, true
	}

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := GetTenant(r.Context())
		if tenant.Type != TenantTypeOrg || tenant.ID != "custom-org" {
			t.Errorf("Got tenant %+v, want org:custom-org", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	// No header - should use extractor
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_ExtractorTakesPrecedence(t *testing.T) {
	config := DefaultConfig()
	config.TenantExtractor = func(r *http.Request) (Tenant, bool) {
		return Tenant{Type: TenantTypeOrg, ID: "from-extractor"}, true
	}

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := GetTenant(r.Context())
		// Extractor should take precedence over header
		if tenant.ID != "from-extractor" {
			t.Errorf("Got tenant %+v, extractor should take precedence", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Tenant-ID", "user:from-header")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_CustomErrorHandler(t *testing.T) {
	customErrorCalled := false
	config := DefaultConfig()
	config.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		customErrorCalled = true
		w.WriteHeader(http.StatusForbidden)
	}

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !customErrorCalled {
		t.Error("Custom error handler should be called")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 from custom handler, got %d", rec.Code)
	}
}

func TestRequireTenantMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(r *http.Request) *http.Request
		wantStatus int
	}{
		{
			name: "with tenant",
			setupCtx: func(r *http.Request) *http.Request {
				ctx := SetTenant(r.Context(), Tenant{Type: TenantTypeUser, ID: "123"})
				return r.WithContext(ctx)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "without tenant",
			setupCtx: func(r *http.Request) *http.Request {
				return r
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireTenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := tt.setupCtx(httptest.NewRequest("GET", "/api/test", nil))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestPublicRoutes(t *testing.T) {
	publicPaths := []string{"/api/health", "/api/discovery", "/api/public/*"}

	strictConfig := DefaultConfig()
	middleware := PublicRoutes(publicPaths, Middleware(strictConfig))

	tests := []struct {
		path       string
		hasTenant  bool
		wantStatus int
	}{
		{"/api/health", false, http.StatusOK},
		{"/api/discovery", false, http.StatusOK},
		{"/api/public/anything", false, http.StatusOK},
		{"/api/public/nested/path", false, http.StatusOK},
		{"/api/issues", false, http.StatusUnauthorized}, // requires tenant
		{"/api/issues", true, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.hasTenant {
				req.Header.Set("X-Tenant-ID", "user:123")
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Path %s: expected %d, got %d", tt.path, tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestWithSessionTenantExtractor(t *testing.T) {
	// Simulate session storage
	type sessionKey struct{}
	
	sessionGetter := func(ctx context.Context) (tenantStr, userID string, ok bool) {
		if s, ok := ctx.Value(sessionKey{}).(map[string]string); ok {
			return s["tenant"], s["user"], true
		}
		return "", "", false
	}

	config := WithSessionTenantExtractor(sessionGetter)

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := GetTenant(r.Context())
		if tenant.Type != TenantTypeOrg || tenant.ID != "my-org" {
			t.Errorf("Got tenant %+v, want org:my-org", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	// Add session to context
	ctx := context.WithValue(req.Context(), sessionKey{}, map[string]string{
		"tenant": "org:my-org",
		"user":   "user-123",
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	// This test verifies that context isolation works correctly
	config := DefaultConfig()

	var capturedTenants []Tenant

	handler := Middleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := GetTenant(r.Context())
		capturedTenants = append(capturedTenants, tenant)
		w.WriteHeader(http.StatusOK)
	}))

	// First request - user tenant
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	req1.Header.Set("X-Tenant-ID", "user:tenant-1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Second request - org tenant
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	req2.Header.Set("X-Tenant-ID", "org:tenant-2")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Verify both requests got correct tenants
	if len(capturedTenants) != 2 {
		t.Fatalf("Expected 2 captured tenants, got %d", len(capturedTenants))
	}

	if capturedTenants[0].ID != "tenant-1" || capturedTenants[0].Type != TenantTypeUser {
		t.Errorf("First request got wrong tenant: %+v", capturedTenants[0])
	}

	if capturedTenants[1].ID != "tenant-2" || capturedTenants[1].Type != TenantTypeOrg {
		t.Errorf("Second request got wrong tenant: %+v", capturedTenants[1])
	}
}
