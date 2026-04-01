package featureflags

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
)

// Evaluator evaluates feature flags against a given context
type Evaluator struct {
	store *Store
}

// NewEvaluator creates a new Evaluator
func NewEvaluator(store *Store) *Evaluator {
	return &Evaluator{store: store}
}

// Evaluate checks if a flag is enabled for the given context
// Evaluation priority:
//  1. Disabled user override (explicit disable for user)
//  2. Enabled user override (explicit enable for user)
//  3. Disabled tenant override (explicit disable for tenant)
//  4. Enabled tenant override (explicit enable for tenant)
//  5. Percentage rollout (hash-based)
//  6. Global default
func (e *Evaluator) Evaluate(ctx context.Context, flagKey string, evalCtx *EvaluationContext) (*EvaluatedFlag, error) {
	flag, err := e.store.Get(ctx, flagKey)
	if err != nil {
		return nil, err
	}

	if flag == nil {
		// Flag doesn't exist - return disabled
		return &EvaluatedFlag{
			Key:     flagKey,
			Enabled: false,
			Reason:  "flag_not_found",
		}, nil
	}

	return e.evaluateFlag(flag, evalCtx), nil
}

// EvaluateAll evaluates all flags for the given context
func (e *Evaluator) EvaluateAll(ctx context.Context, evalCtx *EvaluationContext) ([]*EvaluatedFlag, error) {
	flags, err := e.store.List(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*EvaluatedFlag, 0, len(flags))
	for _, flag := range flags {
		results = append(results, e.evaluateFlag(flag, evalCtx))
	}

	return results, nil
}

// evaluateFlag performs the actual flag evaluation
func (e *Evaluator) evaluateFlag(flag *FeatureFlag, evalCtx *EvaluationContext) *EvaluatedFlag {
	result := &EvaluatedFlag{Key: flag.Key}

	// 1. Check disabled user override first (highest priority)
	if evalCtx.UserID != "" && containsString(flag.DisabledUsers, evalCtx.UserID) {
		result.Enabled = false
		result.Reason = "user_disabled"
		return result
	}

	// 2. Check enabled user override
	if evalCtx.UserID != "" && containsString(flag.Users, evalCtx.UserID) {
		result.Enabled = true
		result.Reason = "user_enabled"
		return result
	}

	// 3. Check disabled tenant override
	if evalCtx.TenantID != "" && containsString(flag.DisabledTenants, evalCtx.TenantID) {
		result.Enabled = false
		result.Reason = "tenant_disabled"
		return result
	}

	// 4. Check enabled tenant override
	if evalCtx.TenantID != "" && containsString(flag.Tenants, evalCtx.TenantID) {
		result.Enabled = true
		result.Reason = "tenant_enabled"
		return result
	}

	// 5. Check percentage rollout
	if flag.Percentage > 0 && flag.Percentage < 100 {
		// Use user ID if available, otherwise session ID
		hashInput := evalCtx.UserID
		if hashInput == "" {
			hashInput = evalCtx.SessionID
		}

		if hashInput != "" {
			bucket := calculateBucket(flag.Key, hashInput)
			if bucket < flag.Percentage {
				result.Enabled = true
				result.Reason = "percentage_rollout"
				return result
			}
		}
	}

	// 6. Fall back to global default
	result.Enabled = flag.Enabled
	if flag.Enabled {
		result.Reason = "global_enabled"
	} else {
		result.Reason = "global_disabled"
	}
	return result
}

// calculateBucket computes a deterministic bucket (0-99) for percentage rollout
// Uses hash(flag_key || user_id) % 100 for consistent assignment
func calculateBucket(flagKey, identifier string) int {
	h := sha256.New()
	h.Write([]byte(flagKey))
	h.Write([]byte(identifier))
	hash := h.Sum(nil)

	// Use first 8 bytes as uint64 and mod 100
	value := binary.BigEndian.Uint64(hash[:8])
	return int(value % 100)
}

// GetDefaultEvaluator returns an evaluator using the global store
func GetDefaultEvaluator() *Evaluator {
	return NewEvaluator(GetStore())
}
