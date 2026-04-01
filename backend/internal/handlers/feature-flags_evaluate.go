package handlers

import (
	"log/slog"
	"net/http"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/featureflags"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/hal"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// EvaluateFeatureFlags handles GET /api/feature-flags/evaluate
// Evaluates all flags for the current user context
func EvaluateFeatureFlags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	middleware.SetTxnLogDataType(ctx, "FeatureFlag")

	evaluator := featureflags.GetDefaultEvaluator()

	// Build evaluation context from request
	evalCtx := buildEvaluationContext(r)

	// Evaluate all flags
	results, err := evaluator.EvaluateAll(ctx, evalCtx)
	if err != nil {
		slog.Error("failed to evaluate flags", "error", err)
		writeFeatureFlagError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to evaluate flags")
		return
	}

	// Convert to simple map for frontend
	flags := make(map[string]bool, len(results))
	for _, result := range results {
		flags[result.Key] = result.Enabled
	}

	logger := middleware.LoggerWithCorrelation(ctx)
	logger.Debug("flags evaluated",
		"count", len(flags),
		"user_id", evalCtx.UserID,
		"tenant_id", evalCtx.TenantID,
	)

	response := hal.NewBuilder().
		Self("/api/feature-flags/evaluate").
		Data(map[string]interface{}{
			"flags": flags,
		}).
		Build()

	hal.WriteResource(w, http.StatusOK, response)
}

// buildEvaluationContext creates an evaluation context from the request
func buildEvaluationContext(r *http.Request) *featureflags.EvaluationContext {
	ctx := r.Context()
	evalCtx := &featureflags.EvaluationContext{}

	// Get user ID if authenticated
	// TODO: Integrate with auth package when available
	// if user := auth.GetUser(ctx); user != nil {
	// 	evalCtx.UserID = user.ID
	// }

	// Get session ID from cookie or header
	if sessionCookie, err := r.Cookie("ds_session"); err == nil {
		evalCtx.SessionID = sessionCookie.Value
	}

	// Get tenant ID from header or context
	// TODO: Integrate with tenant package when available
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		evalCtx.TenantID = tenantID
	}

	// Also check context for any injected values
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		evalCtx.UserID = userID
	}
	if tenantID, ok := ctx.Value("tenant_id").(string); ok && tenantID != "" {
		evalCtx.TenantID = tenantID
	}

	return evalCtx
}
