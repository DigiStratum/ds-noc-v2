// Example: Backfill data to a new field.
//
// Backfills iterate over existing items and add/transform data.
// Use rate limiting to avoid DynamoDB throttling.
//
// This example adds a `created_at` field to all products
// that don't have one.
package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/DigiStratum/GoTools/codegen/dbmigrate"
)

// M003_BackfillCreatedAt adds created_at to existing products.
type M003_BackfillCreatedAt struct{}

func (m *M003_BackfillCreatedAt) Version() string { return "003" }
func (m *M003_BackfillCreatedAt) Name() string    { return "backfill_created_at" }

func (m *M003_BackfillCreatedAt) Up(ctx context.Context, db dbmigrate.DynamoDBClient) error {
	config := &dbmigrate.BackfillConfig{
		BatchSize:        25,              // Items per scan batch
		Workers:          4,               // Parallel workers
		RateLimit:        100,             // Items/second (avoid throttling)
		ProgressInterval: 30 * time.Second,
		RetryAttempts:    3,
		RetryDelay:       1 * time.Second,
	}

	backfiller := dbmigrate.NewBackfiller(db, config)

	// Default timestamp for existing items without created_at
	defaultTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	processor := func(ctx context.Context, item map[string]interface{}) (map[string]interface{}, error) {
		// Skip if already has created_at
		if _, ok := item["created_at"]; ok {
			return nil, nil // Return nil to skip (no update needed)
		}

		// Add the field
		item["created_at"] = defaultTime
		return item, nil
	}

	result, err := backfiller.Run(ctx, "products", processor)
	if err != nil {
		return fmt.Errorf("backfill: %w", err)
	}

	if result.Failed > 0 {
		return fmt.Errorf("%d items failed", result.Failed)
	}

	fmt.Printf("Backfill complete: %d processed, %d updated, %d skipped in %v\n",
		result.Processed, result.Updated, result.Skipped, result.Duration)

	return nil
}

func (m *M003_BackfillCreatedAt) Down(ctx context.Context, db dbmigrate.DynamoDBClient) error {
	// Data backfills typically don't have clean rollbacks.
	// Options:
	// 1. Remove the field (may cause issues if app depends on it)
	// 2. Leave data as-is (most common)
	// 3. Restore from backup

	// This example leaves data as-is
	fmt.Println("⚠️  Rollback does not remove created_at field")
	fmt.Println("    Remove manually if needed, or restore from backup")
	return nil
}
