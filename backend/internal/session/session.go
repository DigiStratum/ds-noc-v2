// Package session provides anonymous and authenticated session management.
//
// Sessions follow the guest-session-first pattern:
// - Anonymous sessions are created on first visit (no auth required)
// - Sessions are scoped to tenant and shared across subdomains
// - Authentication upgrades the session without replacing it
// - Session survives the auth flow
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type contextKey string

const (
	sessionContextKey contextKey = "session"
)

const cookieName = "ds_session"

// Session represents a user session (anonymous or authenticated)
type Session struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`     // Empty for anonymous sessions
	IsGuest    bool      `json:"is_guest"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// IsAuthenticated returns true if the session has an associated user
func (s *Session) IsAuthenticated() bool {
	return s.UserID != ""
}

// Store manages session persistence (in-memory for development, DynamoDB in production)
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// Global store (replace with DynamoDB in production)
var globalStore = &Store{
	sessions: make(map[string]*Session),
}

// GetStore returns the session store singleton
func GetStore() *Store {
	return globalStore
}

// Create creates a new anonymous session
func (s *Store) Create(tenantID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        generateSessionID(),
		TenantID:  tenantID,
		IsGuest:   true,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	s.sessions[session.ID] = session
	return session
}

// Get retrieves a session by ID
func (s *Store) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok || time.Now().After(session.ExpiresAt) {
		return nil
	}
	return session
}

// Upgrade upgrades an anonymous session to an authenticated one
// This preserves the session ID so that any pre-auth state is maintained
func (s *Store) Upgrade(id string, userID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil
	}

	session.UserID = userID
	session.IsGuest = false
	session.ExpiresAt = time.Now().Add(24 * time.Hour) // Refresh expiry on upgrade

	return session
}

// SetTenant updates the tenant context for a session
func (s *Store) SetTenant(id string, tenantID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil
	}

	session.TenantID = tenantID
	return session
}

// Delete removes a session (for logout)
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// GetOrCreate retrieves an existing session or creates a new one with the given ID.
// This is used for DSAccount SSO sessions where the session ID comes from DSAccount.
func (s *Store) GetOrCreate(id string, tenantID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if ok && time.Now().Before(session.ExpiresAt) {
		return session
	}

	// Create new session with the provided ID (DSAccount session ID)
	session = &Session{
		ID:        id,
		TenantID:  tenantID,
		IsGuest:   true, // Will be set to false when user is associated
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	s.sessions[id] = session
	return session
}

// Save persists a session to the store
func (s *Store) Save(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

// generateSessionID creates a cryptographically secure session ID
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// GetSession retrieves the session from context
func GetSession(ctx context.Context) *Session {
	session, _ := ctx.Value(sessionContextKey).(*Session)
	return session
}

// SetSession stores the session in context
func SetSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// getCookieDomain returns the domain for cross-subdomain cookies
// e.g., for "app.digistratum.com" returns ".digistratum.com"
func getCookieDomain(host string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Don't set domain for localhost
	if host == "localhost" || strings.HasPrefix(host, "127.") {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	// Check for dev environment (*.dev.digistratum.com)
	// Return ".dev.digistratum.com" for dev subdomain sharing
	if len(parts) >= 3 && parts[len(parts)-3] == "dev" {
		return "." + strings.Join(parts[len(parts)-3:], ".")
	}

	// Return ".domain.tld" for subdomain sharing
	return "." + strings.Join(parts[len(parts)-2:], ".")
}

// SetSessionCookie creates a session cookie that works across subdomains
func SetSessionCookie(w http.ResponseWriter, r *http.Request, session *Session) {
	domain := getCookieDomain(r.Host)
	secure := os.Getenv("ENV") != "local"

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    session.ID,
		Path:     "/",
		Domain:   domain, // Allows sharing across subdomains
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	domain := getCookieDomain(r.Host)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// GetSessionIDFromRequest extracts session ID from cookie or header
func GetSessionIDFromRequest(r *http.Request) string {
	// Check cookie first
	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Check Authorization header (for API clients)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}
