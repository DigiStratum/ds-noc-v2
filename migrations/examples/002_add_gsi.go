// Example: Add a Global Secondary Index (GSI).
//
// GSIs enable efficient queries on non-key attributes.
// This example adds a GSI to query products by category.
//
// Note: GSI creation can take several minutes in production.
// The WaitForTableActive call may need a longer timeout.
package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/DigiStratum/GoTools/codegen/dbmigrate"
)

// M002_AddCategoryGSI adds a GSI for querying by category.
type M002_AddCategoryGSI struct{}

func (m *M002_AddCategoryGSI) Version() string { return "002" }
func (m *M002_AddCategoryGSI) Name() string    { return "add_category_gsi" }

func (m *M002_AddCategoryGSI) Up(ctx context.Context, db dbmigrate.DynamoDBClient) error {
	input := &dbmigrate.UpdateTableInput{
		TableName: "products",
		AddGlobalSecondaryIndex: &dbmigrate.GSIInput{
			IndexName:        "ByCategory",
			PartitionKey:     "category",
			PartitionKeyType: "S",
			SortKey:          "created_at", // Optional: enables range queries
			SortKeyType:      "S",
			ProjectionType:   "ALL", // ALL, KEYS_ONLY, or INCLUDE
		},
	}

	if err := db.UpdateTable(ctx, input); err != nil {
		return fmt.Errorf("add GSI: %w", err)
	}

	// GSIs can take several minutes to create
	// Adjust timeout based on table size
	if err := db.WaitForTableActive(ctx, "products", 10*time.Minute); err != nil {
		return fmt.Errorf("wait for GSI: %w", err)
	}

	return nil
}

func (m *M002_AddCategoryGSI) Down(ctx context.Context, db dbmigrate.DynamoDBClient) error {
	input := &dbmigrate.UpdateTableInput{
		TableName:                  "products",
		DeleteGlobalSecondaryIndex: "ByCategory",
	}

	if err := db.UpdateTable(ctx, input); err != nil {
		return fmt.Errorf("delete GSI: %w", err)
	}

	// Wait for deletion to complete
	return db.WaitForTableActive(ctx, "products", 5*time.Minute)
}
