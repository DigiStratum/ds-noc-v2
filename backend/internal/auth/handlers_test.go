package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCallbackHandler_MissingCode_Returns400 tests missing code validation
func TestCallbackHandler_MissingCode_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/callback", nil)
	rr := httptest.NewRecorder()

	CallbackHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestCallbackHandler_TokenExchangeSuccess_RedirectsWithoutSettingCookie tests FR-AUTH-001 happy path
// IMPORTANT: This test verifies that the callback handler does NOT set cookies.
// DSAccount owns session management and sets the ds_session cookie during the SSO flow.
func TestCallbackHandler_TokenExchangeSuccess_RedirectsWithoutSettingCookie(t *testing.T) {
	// Arrange: Mock DSAccount token endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sso/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return a valid token response
		response := tokenResponse{
			AccessToken: "mock-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	t.Setenv("DSACCOUNT_SSO_URL", mockServer.URL)
	t.Setenv("DSACCOUNT_APP_ID", "test-app")
	t.Setenv("DSACCOUNT_APP_SECRET", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=valid-code&state=/dashboard", nil)
	rr := httptest.NewRecorder()

	// Act
	CallbackHandler(rr, req)

	// Assert: Should redirect
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	// CRITICAL: Verify NO ds_session cookie is set
	// DSAccount owns session cookies - consumer apps must not set them
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "ds_session" {
			t.Fatal("ds_session cookie should NOT be set by consumer apps - DSAccount owns session management")
		}
	}

	// Check redirect location
	location := rr.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %s", location)
	}
}

// TestCallbackHandler_NoState_RedirectsToRoot tests default redirect behavior
func TestCallbackHandler_NoState_RedirectsToRoot(t *testing.T) {
	// Arrange: Mock DSAccount token endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := tokenResponse{
			AccessToken: "mock-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	t.Setenv("DSACCOUNT_SSO_URL", mockServer.URL)
	t.Setenv("DSACCOUNT_APP_ID", "test-app")
	t.Setenv("DSACCOUNT_APP_SECRET", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=valid-code", nil)
	rr := httptest.NewRecorder()

	// Act
	CallbackHandler(rr, req)

	// Assert: Should redirect to root
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to /, got %s", location)
	}
}

// TestCallbackHandler_TokenExchangeFails_Returns401 tests auth failure
func TestCallbackHandler_TokenExchangeFails_Returns401(t *testing.T) {
	// Arrange: Mock DSAccount returning 401
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_code"}`))
	}))
	defer mockServer.Close()

	t.Setenv("DSACCOUNT_SSO_URL", mockServer.URL)
	t.Setenv("DSACCOUNT_APP_ID", "test-app")
	t.Setenv("DSACCOUNT_APP_SECRET", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=invalid-code", nil)
	rr := httptest.NewRecorder()

	// Act
	CallbackHandler(rr, req)

	// Assert
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestCallbackHandler_PreventOpenRedirect tests security against open redirect
func TestCallbackHandler_PreventOpenRedirect(t *testing.T) {
	// Arrange: Mock DSAccount token endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := tokenResponse{
			AccessToken: "mock-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	t.Setenv("DSACCOUNT_SSO_URL", mockServer.URL)
	t.Setenv("DSACCOUNT_APP_ID", "test-app")
	t.Setenv("DSACCOUNT_APP_SECRET", "test-secret")

	// Try to redirect to external URL
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=valid-code&state=https://evil.com", nil)
	rr := httptest.NewRecorder()

	// Act
	CallbackHandler(rr, req)

	// Assert: Should redirect to / not evil.com
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to / for external URL, got %s", location)
	}
}

// TestLogoutHandler_RedirectsToDSAccount tests FR-AUTH-004
// IMPORTANT: This test verifies that logout does NOT clear cookies locally.
// DSAccount owns session management and will clear the cookie during logout.
func TestLogoutHandler_RedirectsToDSAccount(t *testing.T) {
	t.Setenv("DSACCOUNT_SSO_URL", "https://account.digistratum.com")
	t.Setenv("APP_URL", "https://noc.digistratum.com")

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	// Act
	LogoutHandler(rr, req)

	// Assert: Should redirect to DSAccount logout
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "https://account.digistratum.com/api/sso/logout") {
		t.Errorf("expected redirect to DSAccount logout, got %s", location)
	}

	// CRITICAL: Verify NO cookie clearing is attempted
	// DSAccount owns session cookies - consumer apps must not modify them
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "ds_session" {
			t.Fatal("logout should NOT attempt to clear ds_session cookie - DSAccount owns session management")
		}
	}
}

// TestLoginHandler_RedirectsToSSO tests FR-AUTH-001
func TestLoginHandler_RedirectsToSSO(t *testing.T) {
	t.Setenv("DSACCOUNT_SSO_URL", "https://account.digistratum.com")
	t.Setenv("DSACCOUNT_APP_ID", "noc")

	req := httptest.NewRequest(http.MethodGet, "/auth/login?redirect=/dashboard", nil)
	rr := httptest.NewRecorder()

	// Act
	LoginHandler(rr, req)

	// Assert: Should redirect to DSAccount authorize
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "https://account.digistratum.com/api/sso/authorize") {
		t.Errorf("expected redirect to DSAccount authorize, got %s", location)
	}

	// Verify state parameter preserves redirect
	if !strings.Contains(location, "state=%2Fdashboard") {
		t.Errorf("expected state to contain encoded redirect path, got %s", location)
	}
}

// TestLoginHandler_ReturnUrlParam tests backward compatibility with return_url param
func TestLoginHandler_ReturnUrlParam(t *testing.T) {
	t.Setenv("DSACCOUNT_SSO_URL", "https://account.digistratum.com")
	t.Setenv("DSACCOUNT_APP_ID", "noc")

	req := httptest.NewRequest(http.MethodGet, "/auth/login?return_url=/settings", nil)
	rr := httptest.NewRecorder()

	// Act
	LoginHandler(rr, req)

	// Assert: Should use return_url as state
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "state=%2Fsettings") {
		t.Errorf("expected state to contain encoded return_url path, got %s", location)
	}
}

// TestLoginHandler_DefaultRedirect tests default behavior when no redirect specified
func TestLoginHandler_DefaultRedirect(t *testing.T) {
	t.Setenv("DSACCOUNT_SSO_URL", "https://account.digistratum.com")
	t.Setenv("DSACCOUNT_APP_ID", "noc")

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()

	// Act
	LoginHandler(rr, req)

	// Assert: Should use / as default state
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "state=%2F") {
		t.Errorf("expected state to contain encoded / path, got %s", location)
	}
}
