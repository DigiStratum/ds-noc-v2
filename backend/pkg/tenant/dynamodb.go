// Package tenant provides multi-tenant isolation primitives.
//
// This file contains DynamoDB key schema patterns for tenant-scoped data.
package tenant

import (
	"fmt"
	"strings"
)

// Key Prefixes for DynamoDB partition keys
const (
	// TenantPrefix is prepended to all tenant-scoped partition keys
	TenantPrefix = "TENANT#"
)

// PKPrefix returns the DynamoDB partition key prefix for a tenant.
// Format: TENANT#{type}:{id}#
//
// Example:
//
//	tenant.PKPrefix(Tenant{Type: "user", ID: "123"})
//	// Returns: "TENANT#user:123#"
//
// Use this as the start of partition keys for tenant-scoped entities:
//
//	pk := tenant.PKPrefix(t) + "ISSUE#" + issueID
//	// Result: "TENANT#user:123#ISSUE#456"
func PKPrefix(t Tenant) string {
	return TenantPrefix + t.String() + "#"
}

// BuildPK creates a complete partition key for a tenant-scoped entity.
// The segments are joined with # separators after the tenant prefix.
//
// Example:
//
//	tenant.BuildPK(t, "ISSUE", "123")
//	// Returns: "TENANT#user:456#ISSUE#123"
//
//	tenant.BuildPK(t, "PROJECT", "abc", "MEMBER", "user-1")
//	// Returns: "TENANT#user:456#PROJECT#abc#MEMBER#user-1"
func BuildPK(t Tenant, segments ...string) string {
	return PKPrefix(t) + strings.Join(segments, "#")
}

// ParsePK extracts the tenant and remaining key segments from a partition key.
// Returns an error if the key doesn't match the expected format.
//
// Example:
//
//	tenant, segments, err := ParsePK("TENANT#user:123#ISSUE#456")
//	// tenant = {Type: "user", ID: "123"}
//	// segments = ["ISSUE", "456"]
func ParsePK(pk string) (Tenant, []string, error) {
	if !strings.HasPrefix(pk, TenantPrefix) {
		return Tenant{}, nil, fmt.Errorf("partition key missing TENANT# prefix: %s", pk)
	}

	// Remove prefix
	remainder := strings.TrimPrefix(pk, TenantPrefix)

	// Find the tenant portion (ends at next #)
	parts := strings.SplitN(remainder, "#", 2)
	if len(parts) == 0 {
		return Tenant{}, nil, fmt.Errorf("partition key has no tenant: %s", pk)
	}

	t, err := Parse(parts[0])
	if err != nil {
		return Tenant{}, nil, fmt.Errorf("invalid tenant in partition key: %w", err)
	}

	// Parse remaining segments
	var segments []string
	if len(parts) > 1 && parts[1] != "" {
		segments = strings.Split(parts[1], "#")
	}

	return t, segments, nil
}

// ValidatePKBelongsToTenant verifies that a partition key belongs to the expected tenant.
// Returns an error if the key doesn't match or is malformed.
//
// Use this for paranoid validation before returning data to ensure
// cross-tenant data leakage cannot occur.
func ValidatePKBelongsToTenant(pk string, expected Tenant) error {
	t, _, err := ParsePK(pk)
	if err != nil {
		return fmt.Errorf("validate pk: %w", err)
	}

	if t != expected {
		return fmt.Errorf("tenant mismatch: pk has %s, expected %s", t.String(), expected.String())
	}

	return nil
}

// GSI (Global Secondary Index) patterns for common access patterns

// BuildGSI1PK creates a GSI partition key for entity-type queries within a tenant.
// Use for queries like "all issues for tenant" or "all projects for tenant".
//
// Format: TENANT#{type}:{id}#TYPE#{entityType}
//
// Example:
//
//	tenant.BuildGSI1PK(t, "ISSUE")
//	// Returns: "TENANT#user:123#TYPE#ISSUE"
func BuildGSI1PK(t Tenant, entityType string) string {
	return PKPrefix(t) + "TYPE#" + entityType
}

// BuildGSI1SK creates a GSI sort key, typically including timestamp for range queries.
// Format depends on use case - this is a suggested pattern.
//
// Example for time-ordered entities:
//
//	tenant.BuildGSI1SK(createdAt.Format(time.RFC3339), entityID)
//	// Returns: "2024-01-15T10:30:00Z#issue-123"
func BuildGSI1SK(timestamp, entityID string) string {
	return timestamp + "#" + entityID
}
