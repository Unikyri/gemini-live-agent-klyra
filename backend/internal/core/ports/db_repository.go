package ports

import (
	"gorm.io/gorm"
)

// DBRepository defines the contract for database connection management.
// This abstraction enables switching between local PostgreSQL and Cloud SQL
// connections without changing business logic.
//
// The interface provides:
//   - GetDB(): Access to the underlying GORM instance for repositories
//   - Ping(): Health check for connection validation
//   - Close(): Resource cleanup
//   - RunMigrations(): Execution of SQL migration files
//
// Implementations:
//   - PostgreSQLRepository: Direct connection via DSN (development)
//   - CloudSQLRepository: Cloud SQL Proxy or Unix socket (production)
type DBRepository interface {
	// GetDB returns the underlying GORM database instance.
	// Repositories use this to execute queries against the connected database.
	GetDB() *gorm.DB

	// Ping verifies that the database connection is alive and responsive.
	// Returns an error if the connection cannot be reached.
	Ping() error

	// Close terminates the database connection and releases associated resources.
	// Should be called during application shutdown.
	Close() error

	// RunMigrations executes SQL migration files from the specified directory.
	// Migrations are applied in sequential order (000001, 000002, etc.).
	// Returns an error if any migration fails.
	RunMigrations(migrationsPath string) error
}