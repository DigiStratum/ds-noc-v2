package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/DigiStratum/GoLib/Cloud/aws/txnlog"
	"github.com/DigiStratum/ds-noc-v2/backend/pkg/tenant"
)

var (
	// resourceIDPattern matches numeric IDs in URL paths
	resourceIDPattern = regexp.MustCompile(`/(\d+)(?:/|$)`)
)

// TxnLogMiddleware adds transaction logging for CloudWatch ecosystem monitoring.
// Captures request/response details and writes to centralized log group.
//
// Required environment variables:
//   - TXNLOG_GROUP: CloudWatch log group (e.g., /ds/ecosystem/transactions)
//   - APP_ID: Application identifier (e.g., dskanban)
//   - APP_NAME: Display name (e.g., DS Projects)
//   - ENV: Environment (e.g., prod, dev)
//
// Gracefully degrades if log group is not configured (logs warning, does not crash).
func TxnLogMiddleware(next http.Handler) http.Handler {
	logGroup := os.Getenv("TXNLOG_GROUP")
	appID := os.Getenv("APP_ID")
	appName := os.Getenv("APP_NAME")
	env := os.Getenv("ENV")

	// Graceful degradation: if not configured, skip txnlog
	if logGroup == "" {
		slog.Warn("txnlog disabled: TXNLOG_GROUP not configured")
		return next
	}

	logger, err := txnlog.New(txnlog.Config{
		LogGroup: logGroup,
		AppID:    appID,
		AppName:  appName,
		Env:      env,
	})
	if err != nil {
		slog.Warn("txnlog disabled: failed to create logger", "error", err)
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Clone logger for this request
		txn := logger.Clone()
		txn.SetRequest(r.Method, r.URL.Path, extractResourceID(r.URL.Path))

		// Set correlation ID if available
		if correlationID := GetCorrelationID(r.Context()); correlationID != "" {
			txn.SetSession("", correlationID)
		}

		// Set source info (hashed for privacy)
		txn.SetSource(r.RemoteAddr, r.UserAgent(), r.Referer())

		// Add logger to context for handlers to use
		ctx := txnlog.WithLogger(r.Context(), txn)
		r = r.WithContext(ctx)

		// Wrap response writer to capture status/bytes
		wrapped := wrapResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		// Record tenant for audit if set (after handler runs, tenant may be set)
		if t := tenant.GetTenant(r.Context()); !t.IsZero() {
			txn.SetTenant(t.String())
		}

		// Record response metrics
		duration := time.Since(start)
		txn.SetResponse(wrapped.status, "", wrapped.size)

		// Record performance (duration only, cold start handled separately)
		txn.SetPerf(false, 0, 0, duration.Milliseconds())

		// Complete the transaction log
		txn.Complete()
	})
}

// extractResourceID extracts the primary resource ID from a URL path.
// Handles common patterns like /api/issues/123, /api/users/456/profile
func extractResourceID(path string) string {
	matches := resourceIDPattern.FindStringSubmatch(path)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// SetTxnLogDataType sets the data type for the current transaction.
// Call this from handlers after determining the response type.
//
// Example:
//
//	func (h *Handler) GetIssue(w http.ResponseWriter, r *http.Request) {
//	    // ... fetch issue ...
//	    middleware.SetTxnLogDataType(r.Context(), "Issue")
//	    json.NewEncoder(w).Encode(issue)
//	}
func SetTxnLogDataType(ctx context.Context, dataType string) {
	if logger := txnlog.FromContext(ctx); logger != nil {
		logger.SetResponse(0, dataType, 0) // status/bytes set by middleware
	}
}

// SetTxnLogTenant sets the tenant ID for multi-tenant isolation.
// Accepts both tenant.Tenant and string for flexibility.
func SetTxnLogTenant(ctx context.Context, tenantID string) {
	if logger := txnlog.FromContext(ctx); logger != nil {
		logger.SetTenant(tenantID)
	}
}

// SetTxnLogTenantFromContext extracts tenant from context and sets it on txnlog.
// Call this from handlers after tenant is set.
func SetTxnLogTenantFromContext(ctx context.Context) {
	if t := tenant.GetTenant(ctx); !t.IsZero() {
		SetTxnLogTenant(ctx, t.String())
	}
}

// SetTxnLogUser sets the hashed user ID for audit trails.
func SetTxnLogUser(ctx context.Context, userIDHash string) {
	if logger := txnlog.FromContext(ctx); logger != nil {
		logger.SetUser(userIDHash)
	}
}

// SetTxnLogEventType sets an event type for non-HTTP transactions (e.g., SQS, SNS).
func SetTxnLogEventType(ctx context.Context, eventType string) {
	if logger := txnlog.FromContext(ctx); logger != nil {
		logger.SetEventType(eventType)
	}
}
