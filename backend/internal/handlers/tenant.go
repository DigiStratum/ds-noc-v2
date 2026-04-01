package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// TenantResponse represents the current tenant context
type TenantResponse struct {
	TenantID   string `json:"tenant_id"`
	IsPersonal bool   `json:"is_personal"`
}

// ListTenants handles GET /api/tenant
// Returns the current tenant context.
func ListTenants(w http.ResponseWriter, r *http.Request) {
	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(r.Context(), "Tenant")

	tenantID := auth.GetTenantID(r.Context())

	response := TenantResponse{
		TenantID:   tenantID,
		IsPersonal: tenantID == "",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
