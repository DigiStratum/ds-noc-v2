package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// ListMes handles GET /api/me
// Returns the authenticated user info. Returns 401 if not authenticated.
func ListMes(w http.ResponseWriter, r *http.Request) {
	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(r.Context(), "User")

	user := auth.GetUser(r.Context())
	if user == nil {
		writeErrorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "No authenticated user")
		return
	}

	logger := middleware.LoggerWithCorrelation(r.Context())
	logger.Info("user info requested", "user_id", user.ID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

// ErrorResponse represents a standard error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// writeErrorResponse writes a standard error response with correlation ID
func writeErrorResponse(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	correlationID := middleware.GetCorrelationID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: correlationID,
		},
	})
}
