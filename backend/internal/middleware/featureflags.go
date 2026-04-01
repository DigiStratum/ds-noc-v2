package middleware

import (
	"context"
	"net/http"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/featureflags"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
)

type ffContextKey string

const (
	evaluatorContextKey ffContextKey = "featureFlagEvaluator"
	evalCtxContextKey   ffContextKey = "featureFlagEvalContext"
)

// FeatureFlags creates a new feature flags middleware.
// This middleware injects the feature flag evaluator and evaluation context into the request context.
// It enables easy access to feature flags via IsEnabled(ctx, "flag-key").
func FeatureFlags() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Build evaluation context from request
			evalCtx := &featureflags.EvaluationContext{}

			// Get user ID if authenticated
			if user := auth.GetUser(ctx); user != nil {
				evalCtx.UserID = user.ID
			}

			// Get session ID
			if sess := session.GetSession(ctx); sess != nil {
				evalCtx.SessionID = sess.ID
			}

			// Get tenant ID
			evalCtx.TenantID = auth.GetTenantID(ctx)

			// Add evaluator and context to request
			evaluator := featureflags.GetDefaultEvaluator()
			ctx = context.WithValue(ctx, evaluatorContextKey, evaluator)
			ctx = context.WithValue(ctx, evalCtxContextKey, evalCtx)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IsEnabled is a helper to check if a feature flag is enabled for the current context.
// Use this in handlers for conditional feature logic:
//
//	if middleware.IsEnabled(ctx, "new-feature") {
//	    // New code path
//	} else {
//	    // Old code path
//	}
func IsEnabled(ctx context.Context, flagKey string) bool {
	evaluator, ok := ctx.Value(evaluatorContextKey).(*featureflags.Evaluator)
	if !ok || evaluator == nil {
		// Middleware not applied or evaluator missing - default to disabled
		return false
	}

	evalCtx, ok := ctx.Value(evalCtxContextKey).(*featureflags.EvaluationContext)
	if !ok || evalCtx == nil {
		evalCtx = &featureflags.EvaluationContext{}
	}

	result, err := evaluator.Evaluate(ctx, flagKey, evalCtx)
	if err != nil {
		// On error, default to disabled for safety
		return false
	}

	return result.Enabled
}

// IsEnabledForUser checks if a flag is enabled for a specific user/session context.
// Use this when you need to evaluate outside the request middleware chain.
func IsEnabledForUser(ctx context.Context, flagKey, userID, sessionID, tenantID string) bool {
	evaluator := featureflags.GetDefaultEvaluator()
	evalCtx := &featureflags.EvaluationContext{
		UserID:    userID,
		SessionID: sessionID,
		TenantID:  tenantID,
	}

	result, err := evaluator.Evaluate(ctx, flagKey, evalCtx)
	if err != nil {
		return false
	}

	return result.Enabled
}

// GetEvaluationContext retrieves the evaluation context from the request context.
// Useful for debugging or custom evaluation logic.
func GetEvaluationContext(ctx context.Context) *featureflags.EvaluationContext {
	evalCtx, _ := ctx.Value(evalCtxContextKey).(*featureflags.EvaluationContext)
	return evalCtx
}
