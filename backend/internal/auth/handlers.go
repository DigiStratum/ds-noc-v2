package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

// CallbackHandler handles SSO callback [FR-AUTH-001]
// DSAccount has already authenticated the user and set the ds_session cookie
// (shared across *.digistratum.com). We just redirect to the intended destination.
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// The code param confirms DSAccount authorized the request, but we don't need
	// to exchange it — the ds_session cookie was already set by DSAccount during login.
	// This callback is just the redirect back to complete the flow.

	slog.Info("SSO callback received, redirecting")

	// Get redirect URL from state param (preserved through the auth flow)
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
// Clears the session cookie and redirects to DSAccount logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the ds_session cookie
	cookie := &http.Cookie{
		Name:     "ds_session",
		Value:    "",
		Path:     "/",
		Domain:   ".digistratum.com",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	slog.Info("session cookie cleared on logout")

	// Redirect to DSAccount logout (with return URL)
	ssoURL := os.Getenv("DSACCOUNT_SSO_URL")
	if ssoURL == "" {
		ssoURL = "https://account.digistratum.com"
	}
	
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://developer.digistratum.com"
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
	redirectPath := r.URL.Query().Get("redirect")
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
