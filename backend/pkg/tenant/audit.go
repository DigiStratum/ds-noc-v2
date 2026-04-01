// Package tenant provides multi-tenant isolation primitives.
//
// This file contains audit logging helpers for tenant-aware logging.
package tenant

import (
	"context"
	"log/slog"
)

// LogAttrs returns slog attributes for the tenant.
// Use this to add tenant info to structured logs.
//
// Example:
//
//	slog.Info("issue created", tenant.LogAttrs(t)...)
func LogAttrs(t Tenant) []any {
	if t.IsZero() {
		return []any{}
	}
	return []any{
		slog.String("tenant_type", string(t.Type)),
		slog.String("tenant_id", t.ID),
		slog.String("tenant", t.String()),
	}
}

// LogAttrsFromContext extracts tenant from context and returns log attributes.
// Returns empty slice if no tenant in context.
//
// Example:
//
//	slog.Info("operation completed", tenant.LogAttrsFromContext(ctx)...)
func LogAttrsFromContext(ctx context.Context) []any {
	return LogAttrs(GetTenant(ctx))
}

// Logger returns a slog.Logger with tenant attributes pre-attached.
// Falls back to default logger if no tenant in context.
//
// Example:
//
//	logger := tenant.Logger(ctx)
//	logger.Info("processing request")
//	// Logs: {"msg":"processing request","tenant":"user:123","tenant_type":"user","tenant_id":"123"}
func Logger(ctx context.Context) *slog.Logger {
	t := GetTenant(ctx)
	if t.IsZero() {
		return slog.Default()
	}
	return slog.Default().With(LogAttrs(t)...)
}

// LogWith returns a logger with tenant and additional attributes.
// Useful for handlers that want both tenant and correlation ID.
//
// Example:
//
//	logger := tenant.LogWith(ctx, "correlation_id", corrID, "user_id", userID)
func LogWith(ctx context.Context, args ...any) *slog.Logger {
	attrs := append(LogAttrsFromContext(ctx), args...)
	return slog.Default().With(attrs...)
}

// AuditEvent represents a structured audit log entry.
// All audit events include tenant context for compliance and debugging.
type AuditEvent struct {
	// Action is what happened (e.g., "create", "update", "delete", "access")
	Action string

	// Resource is what was affected (e.g., "issue", "project", "user")
	Resource string

	// ResourceID is the specific resource identifier
	ResourceID string

	// Actor is who performed the action (user ID, system, etc.)
	Actor string

	// Tenant is the tenant context (automatically included)
	Tenant Tenant

	// Details contains additional action-specific data
	Details map[string]any
}

// LogAudit logs an audit event with full tenant context.
// This is the primary method for audit trail logging.
//
// Example:
//
//	tenant.LogAudit(ctx, tenant.AuditEvent{
//	    Action:     "create",
//	    Resource:   "issue",
//	    ResourceID: "123",
//	    Actor:      userID,
//	    Details:    map[string]any{"title": issue.Title},
//	})
func LogAudit(ctx context.Context, event AuditEvent) {
	t := GetTenant(ctx)
	event.Tenant = t

	attrs := []any{
		slog.String("audit_action", event.Action),
		slog.String("audit_resource", event.Resource),
		slog.String("audit_resource_id", event.ResourceID),
		slog.String("audit_actor", event.Actor),
	}

	// Add tenant attributes
	attrs = append(attrs, LogAttrs(t)...)

	// Add details as a group
	if len(event.Details) > 0 {
		detailAttrs := make([]any, 0, len(event.Details)*2)
		for k, v := range event.Details {
			detailAttrs = append(detailAttrs, slog.Any(k, v))
		}
		attrs = append(attrs, slog.Group("details", detailAttrs...))
	}

	slog.Info("audit", attrs...)
}

// LogAccess logs a data access event for compliance.
// Use this when reading sensitive data.
func LogAccess(ctx context.Context, resource, resourceID, reason string) {
	t := GetTenant(ctx)
	slog.Info("data_access",
		slog.String("resource", resource),
		slog.String("resource_id", resourceID),
		slog.String("reason", reason),
		slog.String("tenant", t.String()),
		slog.String("tenant_type", string(t.Type)),
		slog.String("tenant_id", t.ID),
	)
}

// LogCrossTenantAttempt logs a potential cross-tenant access attempt.
// This should NEVER happen in normal operation - it indicates a bug or attack.
func LogCrossTenantAttempt(ctx context.Context, requestedTenant, actualTenant Tenant, resource, resourceID string) {
	slog.Error("SECURITY: cross-tenant access attempt",
		slog.String("requested_tenant", requestedTenant.String()),
		slog.String("actual_tenant", actualTenant.String()),
		slog.String("resource", resource),
		slog.String("resource_id", resourceID),
	)
}
