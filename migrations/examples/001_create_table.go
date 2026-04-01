// Example: Create a DynamoDB table.
//
// This example shows the basic structure of a table creation migration.
// Copy and adapt for your app's tables.
package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/DigiStratum/GoTools/codegen/dbmigrate"
)

// M001_CreateProducts creates the products table.
type M001_CreateProducts struct{}

func (m *M001_CreateProducts) Version() string { return "001" }
func (m *M001_CreateProducts) Name() string    { return "create_products_table" }

func (m *M001_CreateProducts) Up(ctx context.Context, db dbmigrate.DynamoDBClient) error {
	input := &dbmigrate.CreateTableInput{
		TableName:        "products",
		PartitionKey:     "product_id",
		PartitionKeyType: "S", // "S" = String, "N" = Number, "B" = Binary
		BillingMode:      "PAY_PER_REQUEST",
	}

	if err := db.CreateTable(ctx, input); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Wait for table to become active
	if err := db.WaitForTableActive(ctx, "products", 2*time.Minute); err != nil {
		return fmt.Errorf("wait for table: %w", err)
	}

	return nil
}

func (m *M001_CreateProducts) Down(ctx context.Context, db dbmigrate.DynamoDBClient) error {
	// ⚠️ This deletes all data in the table!
	return db.DeleteTable(ctx, "products")
}
