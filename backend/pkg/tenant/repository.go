// Package tenant provides multi-tenant isolation primitives.
//
// This file contains a base repository pattern for tenant-scoped DynamoDB access.
package tenant

import (
	"fmt"
)

// Repository is the interface for tenant-scoped data access.
// Embed this in your app's repository structs.
type Repository interface {
	// GetTenant returns the tenant this repository is scoped to.
	GetTenant() Tenant

	// WithTenant returns a new repository scoped to a different tenant.
	// Use for system operations that need to access multiple tenants.
	WithTenant(t Tenant) Repository
}

// BaseRepository provides common tenant-scoped repository functionality.
// Embed this in your concrete repository implementations.
//
// Example:
//
//	type IssueRepository struct {
//	    tenant.BaseRepository
//	    db *dynamodb.Client
//	}
//
//	func NewIssueRepository(t tenant.Tenant, db *dynamodb.Client) *IssueRepository {
//	    return &IssueRepository{
//	        BaseRepository: tenant.NewBaseRepository(t),
//	        db:             db,
//	    }
//	}
//
//	func (r *IssueRepository) GetByID(ctx context.Context, id string) (*Issue, error) {
//	    pk := r.PK("ISSUE", id)
//	    // ... DynamoDB GetItem with pk
//	}
type BaseRepository struct {
	tenant Tenant
}

// NewBaseRepository creates a new base repository scoped to a tenant.
func NewBaseRepository(t Tenant) BaseRepository {
	return BaseRepository{tenant: t}
}

// GetTenant returns the tenant this repository is scoped to.
func (r BaseRepository) GetTenant() Tenant {
	return r.tenant
}

// PK builds a partition key for this tenant with the given segments.
//
// Example:
//
//	r.PK("ISSUE", "123")          // TENANT#user:456#ISSUE#123
//	r.PK("PROJECT", "abc", "TASK", "1")  // TENANT#user:456#PROJECT#abc#TASK#1
func (r BaseRepository) PK(segments ...string) string {
	return BuildPK(r.tenant, segments...)
}

// GSI1PK builds a GSI partition key for entity-type queries.
//
// Example:
//
//	r.GSI1PK("ISSUE")  // TENANT#user:123#TYPE#ISSUE
func (r BaseRepository) GSI1PK(entityType string) string {
	return BuildGSI1PK(r.tenant, entityType)
}

// ValidatePK ensures a partition key belongs to this repository's tenant.
// Use this before returning data to prevent cross-tenant leaks.
func (r BaseRepository) ValidatePK(pk string) error {
	return ValidatePKBelongsToTenant(pk, r.tenant)
}

// ScopedQuery represents a tenant-scoped DynamoDB query.
// Use this to build safe queries that always include tenant isolation.
type ScopedQuery struct {
	// TableName is the DynamoDB table name
	TableName string

	// PKPrefix is the partition key prefix for the query
	PKPrefix string

	// Tenant is the tenant context
	Tenant Tenant

	// IndexName is the GSI name (empty for primary index)
	IndexName string
}

// NewScopedQuery creates a query builder scoped to a tenant.
//
// Example:
//
//	q := tenant.NewScopedQuery(t, "my-table", "ISSUE")
//	// q.PKPrefix = "TENANT#user:123#ISSUE"
func NewScopedQuery(t Tenant, tableName, entityType string) *ScopedQuery {
	return &ScopedQuery{
		TableName: tableName,
		PKPrefix:  BuildPK(t, entityType),
		Tenant:    t,
	}
}

// UseGSI sets the query to use a Global Secondary Index.
func (q *ScopedQuery) UseGSI(indexName, entityType string) *ScopedQuery {
	q.IndexName = indexName
	q.PKPrefix = BuildGSI1PK(q.Tenant, entityType)
	return q
}

// ContextRepository returns a repository from context with the current tenant.
// Use this to create repositories scoped to the request's tenant.
//
// Example:
//
//	func (h *Handler) ListIssues(w http.ResponseWriter, r *http.Request) {
//	    t, err := tenant.RequireTenant(r.Context())
//	    if err != nil {
//	        // handle error
//	    }
//	    repo := NewIssueRepository(t, h.db)
//	    issues, err := repo.List(r.Context())
//	    // ...
//	}
//
// For convenience, you can create a factory:
//
//	type RepoFactory struct {
//	    db *dynamodb.Client
//	}
//
//	func (f *RepoFactory) Issues(ctx context.Context) (*IssueRepository, error) {
//	    t, err := tenant.RequireTenant(ctx)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return NewIssueRepository(t, f.db), nil
//	}

// EnsureTenantMatch validates that an entity belongs to the expected tenant.
// Use this when loading data that might have been tampered with.
//
// Example:
//
//	func (r *IssueRepository) GetByID(ctx context.Context, id string) (*Issue, error) {
//	    // ... load from DynamoDB
//	    if err := tenant.EnsureTenantMatch(r.tenant, item.PK); err != nil {
//	        // Log security event and return not found
//	        tenant.LogCrossTenantAttempt(ctx, requestedTenant, r.tenant, "issue", id)
//	        return nil, ErrNotFound
//	    }
//	    return issue, nil
//	}
func EnsureTenantMatch(expected Tenant, pk string) error {
	actual, _, err := ParsePK(pk)
	if err != nil {
		return fmt.Errorf("invalid pk format: %w", err)
	}
	if actual != expected {
		return fmt.Errorf("tenant mismatch: got %s, expected %s", actual.String(), expected.String())
	}
	return nil
}
