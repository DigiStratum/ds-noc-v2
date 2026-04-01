// Package migrations contains DynamoDB migrations for this app.
//
// To add a new migration:
// 1. Create a new file: NNN_description.go
// 2. Implement the Migration interface
// 3. Register it in registry.go
//
// See examples/ for common patterns.
package migrations

import "github.com/DigiStratum/GoTools/codegen/dbmigrate"

// All returns all migrations in order.
// Add new migrations here after creating them.
func All() []dbmigrate.Migration {
	return []dbmigrate.Migration{
		// Add your migrations here, e.g.:
		// &M001_CreateProducts{},
		// &M002_AddCategoryGSI{},
	}
}
