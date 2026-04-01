// Package featureflags provides feature flag infrastructure for deploy/release separation.
// This enables safe deployments by separating code deployment from feature availability.
//
// Evaluation priority:
//  1. User-specific override (explicit enable/disable for specific users)
//  2. Tenant-specific override (explicit enable/disable for specific tenants)
//  3. Percentage rollout (gradual rollout based on user/session hash)
//  4. Global default (base enabled/disabled state)
package featureflags

import (
	"time"
)

// FeatureFlag represents a feature flag configuration
type FeatureFlag struct {
	// Key is the unique identifier for the flag (e.g., "new-dashboard", "beta-feature")
	Key string `json:"key" dynamodbav:"Key"`

	// Enabled is the global default state
	Enabled bool `json:"enabled" dynamodbav:"Enabled"`

	// Description explains what this flag controls
	Description string `json:"description" dynamodbav:"Description"`

	// Tenants is a list of tenant IDs where this flag is explicitly enabled
	// Empty means no tenant-specific override
	Tenants []string `json:"tenants,omitempty" dynamodbav:"Tenants,omitempty"`

	// Users is a list of user IDs where this flag is explicitly enabled
	// Empty means no user-specific override
	Users []string `json:"users,omitempty" dynamodbav:"Users,omitempty"`

	// DisabledTenants is a list of tenant IDs where this flag is explicitly disabled
	// Takes precedence over Tenants list
	DisabledTenants []string `json:"disabled_tenants,omitempty" dynamodbav:"DisabledTenants,omitempty"`

	// DisabledUsers is a list of user IDs where this flag is explicitly disabled
	// Takes precedence over Users list
	DisabledUsers []string `json:"disabled_users,omitempty" dynamodbav:"DisabledUsers,omitempty"`

	// Percentage is the rollout percentage (0-100)
	// 0 means no percentage rollout, 100 means enabled for everyone in rollout
	Percentage int `json:"percentage" dynamodbav:"Percentage"`

	// CreatedAt is when the flag was created
	CreatedAt time.Time `json:"created_at" dynamodbav:"CreatedAt"`

	// UpdatedAt is when the flag was last modified
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"UpdatedAt"`
}

// EvaluationContext provides the context needed to evaluate flags
type EvaluationContext struct {
	// UserID is the authenticated user's ID (may be empty for guests)
	UserID string

	// SessionID is the current session ID (used for percentage rollout of guests)
	SessionID string

	// TenantID is the current tenant context
	TenantID string
}

// EvaluatedFlag represents a flag evaluation result
type EvaluatedFlag struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"` // Why it's enabled/disabled (for debugging)
}

// FlagUpdate represents an update request for a feature flag
type FlagUpdate struct {
	Enabled         bool     `json:"enabled"`
	Description     string   `json:"description,omitempty"`
	Tenants         []string `json:"tenants,omitempty"`
	Users           []string `json:"users,omitempty"`
	DisabledTenants []string `json:"disabled_tenants,omitempty"`
	DisabledUsers   []string `json:"disabled_users,omitempty"`
	Percentage      int      `json:"percentage"`
}

// NewFeatureFlag creates a new feature flag with sensible defaults
func NewFeatureFlag(key, description string, enabled bool) *FeatureFlag {
	now := time.Now().UTC()
	return &FeatureFlag{
		Key:         key,
		Enabled:     enabled,
		Description: description,
		Tenants:     []string{},
		Users:       []string{},
		Percentage:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// containsString checks if a slice contains a string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
