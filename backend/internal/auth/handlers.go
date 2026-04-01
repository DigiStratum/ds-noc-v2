package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

// tokenResponse represents DSAccount's /api/sso/token response
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// CallbackHandler handles SSO callback [FR-AUTH-001]
// This exchanges the authorization code with DSAccount to validate the authentication.
//
// IMPORTANT: This handler does NOT set the ds_session cookie.
// DSAccount owns session management and has already set the ds_session cookie
// during the SSO flow (with Domain=.digistratum.com for cross-subdomain access).
// Consumer apps must NOT overwrite this cookie.
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	slog.Info("SSO callback received", "code_length", len(code))

	// Exchange code for token with DSAccount
	// This validates the authentication and "uses" the one-time code
	// The session cookie was already set by DSAccount during the auth flow
	ssoURL := os.Getenv("DSACCOUNT_SSO_URL")
	if ssoURL == "" {
		ssoURL = "https://account.digistratum.com"
	}

	appID := os.Getenv("DSACCOUNT_APP_ID")
	appSecret := os.Getenv("DSACCOUNT_APP_SECRET")

	tokenReq := map[string]string{
		"code":       code,
		"app_id":     appID,
		"app_secret": appSecret,
	}
	tokenBody, _ := json.Marshal(tokenReq)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(ssoURL+"/api/sso/token", "application/json", bytes.NewReader(tokenBody))
	if err != nil {
		slog.Error("failed to exchange code for token", "error", err)
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("token exchange failed", "status", resp.StatusCode, "body", string(body))
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		slog.Error("failed to decode token response", "error", err)
		http.Error(w, "Failed to process authentication", http.StatusInternalServerError)
		return
	}

	slog.Info("token exchange successful", "token_type", tokenResp.TokenType, "expires_in", tokenResp.ExpiresIn)

	// NOTE: We do NOT set any cookies here.
	// DSAccount has already set the ds_session cookie during the SSO flow.
	// Setting a cookie here would overwrite DSAccount's session with a JWT,
	// breaking session validation for all apps that share the session.

	// Get redirect URL from state param (how OAuth returns our original redirect)
	redirectURL := r.URL.Query().Get("state")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Validate redirect URL to prevent open redirect
	// Only allow relative paths or same-origin
	if len(redirectURL) > 0 && redirectURL[0] != '/' {
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// LogoutHandler handles logout [FR-AUTH-004]
// Redirects to DSAccount logout - DSAccount will clear the session cookie
//
// IMPORTANT: This handler does NOT clear the ds_session cookie directly.
// DSAccount owns session management and will clear the cookie during logout.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("logout initiated, redirecting to DSAccount")

	// NOTE: We do NOT clear the ds_session cookie here.
	// DSAccount will clear it as part of the logout flow.
	// Attempting to clear it here could cause race conditions or
	// leave stale cookies if the domain/path doesn't match exactly.

	// Redirect to DSAccount logout (with return URL)
	ssoURL := os.Getenv("DSACCOUNT_SSO_URL")
	if ssoURL == "" {
		ssoURL = "https://account.digistratum.com"
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://noc.digistratum.com"
	}

	// Include redirect_uri so DSAccount can redirect back after logout
	http.Redirect(w, r, fmt.Sprintf("%s/api/sso/logout?redirect_uri=%s", ssoURL, appURL), http.StatusFound)
}

// LoginHandler initiates the SSO login flow [FR-AUTH-001]
// Redirects to DSAccount's authorize endpoint with:
// - app_id: identifies this app (DSAccount uses registered redirect_uri)
// - state: preserves the user's intended destination through the auth flow
// SECURITY: redirect_uri is NOT passed in URL - DSAccount uses the registered value
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	ssoURL := os.Getenv("DSACCOUNT_SSO_URL")
	if ssoURL == "" {
		ssoURL = "https://account.digistratum.com"
	}

	// Preserve the user's intended destination through the auth flow (passed as state)
	// Check both "redirect" and "return_url" params for compatibility
	redirectPath := r.URL.Query().Get("redirect")
	if redirectPath == "" {
		redirectPath = r.URL.Query().Get("return_url")
	}
	if redirectPath == "" {
		redirectPath = "/"
	}

	// Build the OAuth authorize URL with properly encoded parameters
	// SECURITY: Only app_id is passed. redirect_uri comes from DSAccount app registration
	// to prevent open redirect vulnerabilities.
	params := url.Values{}
	params.Set("app_id", os.Getenv("DSACCOUNT_APP_ID"))
	params.Set("state", redirectPath)

	authURL := ssoURL + "/api/sso/authorize?" + params.Encode()

	slog.Info("initiating SSO login", "auth_url", authURL)
	http.Redirect(w, r, authURL, http.StatusFound)
}
