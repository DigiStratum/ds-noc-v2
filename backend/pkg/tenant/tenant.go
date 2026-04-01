// Package tenant provides multi-tenant isolation primitives for DS Ecosystem apps.
//
// Tenant Model:
//   - User tenant: A personal workspace owned by an individual user (type: "user")
//   - Org tenant: A shared workspace owned by an organization (type: "org")
//
// Canonical Format:
//
//	Tenants are serialized as "type:id" (e.g., "user:123" or "org:abc-def")
//	This format is used in DynamoDB partition keys: TENANT#user:123#...
//
// Context Propagation:
//
//	Use SetTenant/GetTenant to pass tenant through request context.
//	All tenant-scoped operations MUST verify tenant is set.
//
// Security:
//   - All data access MUST be scoped to tenant
//   - Cross-tenant access is denied by default
//   - Missing tenant context = request rejected (not guest fallback)
package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// TenantType identifies whether a tenant is a user or organization
type TenantType string

const (
	// TenantTypeUser represents a personal user workspace
	TenantTypeUser TenantType = "user"

	// TenantTypeOrg represents a shared organization workspace
	TenantTypeOrg TenantType = "org"
)

// Valid returns true if the tenant type is a known value
func (t TenantType) Valid() bool {
	return t == TenantTypeUser || t == TenantTypeOrg
}

// String returns the string representation of the tenant type
func (t TenantType) String() string {
	return string(t)
}

// Tenant represents a tenant (user or organization) in the system.
// All data in DS Ecosystem apps is scoped to a tenant.
type Tenant struct {
	// Type is either "user" or "org"
	Type TenantType `json:"type"`

	// ID is the unique identifier for this tenant (user ID or org ID)
	ID string `json:"id"`
}

// String returns the canonical string representation: "type:id"
// This format is used in DynamoDB partition keys.
func (t Tenant) String() string {
	return string(t.Type) + ":" + t.ID
}

// IsZero returns true if the tenant is uninitialized
func (t Tenant) IsZero() bool {
	return t.Type == "" || t.ID == ""
}

// IsUser returns true if this is a user (personal) tenant
func (t Tenant) IsUser() bool {
	return t.Type == TenantTypeUser
}

// IsOrg returns true if this is an organization tenant
func (t Tenant) IsOrg() bool {
	return t.Type == TenantTypeOrg
}

// Errors returned by Parse
var (
	ErrInvalidFormat = errors.New("tenant: invalid format, expected 'type:id'")
	ErrInvalidType   = errors.New("tenant: invalid type, expected 'user' or 'org'")
	ErrEmptyID       = errors.New("tenant: id cannot be empty")
)

// Parse reconstructs a Tenant from its canonical string form.
// Returns an error if the format is invalid.
//
// Examples:
//
//	tenant.Parse("user:123")  // → Tenant{Type: "user", ID: "123"}
//	tenant.Parse("org:abc")   // → Tenant{Type: "org", ID: "abc"}
//	tenant.Parse("invalid")   // → error: invalid format
func Parse(s string) (Tenant, error) {
	if s == "" {
		return Tenant{}, ErrInvalidFormat
	}

	idx := strings.Index(s, ":")
	if idx == -1 {
		return Tenant{}, ErrInvalidFormat
	}

	tenantType := TenantType(s[:idx])
	id := s[idx+1:]

	if !tenantType.Valid() {
		return Tenant{}, ErrInvalidType
	}

	if id == "" {
		return Tenant{}, ErrEmptyID
	}

	return Tenant{Type: tenantType, ID: id}, nil
}

// MustParse is like Parse but panics on error.
// Use only for known-valid tenant strings (e.g., tests, constants).
func MustParse(s string) Tenant {
	t, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("tenant.MustParse(%q): %v", s, err))
	}
	return t
}

// New creates a new Tenant with the given type and ID.
// Validates that type is valid and ID is non-empty.
func New(tenantType TenantType, id string) (Tenant, error) {
	if !tenantType.Valid() {
		return Tenant{}, ErrInvalidType
	}
	if id == "" {
		return Tenant{}, ErrEmptyID
	}
	return Tenant{Type: tenantType, ID: id}, nil
}

// NewUser creates a user tenant with the given ID.
func NewUser(userID string) (Tenant, error) {
	return New(TenantTypeUser, userID)
}

// NewOrg creates an organization tenant with the given ID.
func NewOrg(orgID string) (Tenant, error) {
	return New(TenantTypeOrg, orgID)
}

// Context key for tenant storage
type contextKey struct{}

// SetTenant stores the tenant in the context.
// Use this in middleware after validating tenant access.
func SetTenant(ctx context.Context, tenant Tenant) context.Context {
	return context.WithValue(ctx, contextKey{}, tenant)
}

// GetTenant retrieves the tenant from context.
// Returns the zero Tenant if not set.
//
// Most code should use RequireTenant instead to enforce tenant presence.
func GetTenant(ctx context.Context) Tenant {
	if t, ok := ctx.Value(contextKey{}).(Tenant); ok {
		return t
	}
	return Tenant{}
}

// RequireTenant retrieves the tenant from context, returning an error if not set.
// Use this in handlers/services that require tenant isolation.
func RequireTenant(ctx context.Context) (Tenant, error) {
	t := GetTenant(ctx)
	if t.IsZero() {
		return Tenant{}, ErrNoTenantInContext
	}
	return t, nil
}

// ErrNoTenantInContext is returned when RequireTenant is called but no tenant is set
var ErrNoTenantInContext = errors.New("tenant: no tenant in context")
