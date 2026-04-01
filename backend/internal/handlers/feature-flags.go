package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/featureflags"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/hal"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// ListFeatureFlags handles GET /api/feature-flags
// Returns all feature flags (admin only)
func ListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	middleware.SetTxnLogDataType(ctx, "FeatureFlag")

	// TODO: Add admin check when auth is integrated
	// if !isAdmin(r) {
	// 	writeFeatureFlagError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
	// 	return
	// }

	store := featureflags.GetStore()
	flags, err := store.List(ctx)
	if err != nil {
		slog.Error("failed to list flags", "error", err)
		writeFeatureFlagError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list flags")
		return
	}

	response := hal.NewBuilder().
		Self("/api/feature-flags").
		Data(map[string]interface{}{
			"flags": flags,
			"count": len(flags),
		}).
		Build()

	hal.WriteResource(w, http.StatusOK, response)
}

// PatchFeatureFlag handles PATCH /api/feature-flags/{key}
// Updates or creates a feature flag
func PatchFeatureFlag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	middleware.SetTxnLogDataType(ctx, "FeatureFlag")

	// TODO: Add admin check when auth is integrated

	// Extract flag key from path
	flagKey := extractFlagKey(r.URL.Path, "/api/feature-flags/")
	if flagKey == "" {
		writeFeatureFlagError(w, http.StatusBadRequest, "INVALID_KEY", "Flag key required")
		return
	}

	// Parse request body
	var update featureflags.FlagUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeFeatureFlagError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	store := featureflags.GetStore()

	// Get existing flag or create new one
	flag, err := store.Get(ctx, flagKey)
	if err != nil {
		slog.Error("failed to get flag", "key", flagKey, "error", err)
		writeFeatureFlagError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get flag")
		return
	}

	if flag == nil {
		// Create new flag
		flag = featureflags.NewFeatureFlag(flagKey, update.Description, update.Enabled)
	}

	// Apply updates
	if update.Description != "" {
		flag.Description = update.Description
	}
	flag.Enabled = update.Enabled
	if update.Tenants != nil {
		flag.Tenants = update.Tenants
	}
	if update.Users != nil {
		flag.Users = update.Users
	}
	if update.DisabledTenants != nil {
		flag.DisabledTenants = update.DisabledTenants
	}
	if update.DisabledUsers != nil {
		flag.DisabledUsers = update.DisabledUsers
	}
	if update.Percentage >= 0 && update.Percentage <= 100 {
		flag.Percentage = update.Percentage
	}
	flag.UpdatedAt = time.Now().UTC()

	// Save
	if err := store.Save(ctx, flag); err != nil {
		slog.Error("failed to save flag", "key", flagKey, "error", err)
		writeFeatureFlagError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save flag")
		return
	}

	logger := middleware.LoggerWithCorrelation(ctx)
	logger.Info("flag updated",
		"key", flagKey,
		"enabled", flag.Enabled,
		"percentage", flag.Percentage,
	)

	response := hal.NewBuilder().
		Self("/api/feature-flags/" + flagKey).
		Data(flag).
		Build()

	hal.WriteResource(w, http.StatusOK, response)
}

// DeleteFeatureFlag handles DELETE /api/feature-flags/{key}
// Deletes a feature flag
func DeleteFeatureFlag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	middleware.SetTxnLogDataType(ctx, "FeatureFlag")

	// TODO: Add admin check when auth is integrated

	// Extract flag key from path
	flagKey := extractFlagKey(r.URL.Path, "/api/feature-flags/")
	if flagKey == "" {
		writeFeatureFlagError(w, http.StatusBadRequest, "INVALID_KEY", "Flag key required")
		return
	}

	store := featureflags.GetStore()
	if err := store.Delete(ctx, flagKey); err != nil {
		slog.Error("failed to delete flag", "key", flagKey, "error", err)
		writeFeatureFlagError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete flag")
		return
	}

	logger := middleware.LoggerWithCorrelation(ctx)
	logger.Info("flag deleted", "key", flagKey)

	w.WriteHeader(http.StatusNoContent)
}

// extractFlagKey extracts the flag key from the URL path
func extractFlagKey(path, prefix string) string {
	key := strings.TrimPrefix(path, prefix)
	if key == "" || key == path {
		return ""
	}
	// Remove any trailing slashes
	return strings.TrimSuffix(key, "/")
}

// writeFeatureFlagError writes an error response in HAL format
func writeFeatureFlagError(w http.ResponseWriter, status int, code, message string) {
	response := hal.NewBuilder().
		Data(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    code,
				"message": message,
			},
		}).
		Build()
	hal.WriteResource(w, status, response)
}
