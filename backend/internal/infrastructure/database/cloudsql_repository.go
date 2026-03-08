package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// CloudSQLRepository implements ports.DBRepository for GCP Cloud SQL connections.
// It supports both unix socket (/cloudsql/INSTANCE_CONNECTION_NAME) and TCP host fallback.
type CloudSQLRepository struct {
	db *gorm.DB
}

// NewCloudSQLRepository creates a Cloud SQL repository.
// instanceConnectionName can be empty if DB_HOST already points to the Cloud SQL proxy endpoint.
func NewCloudSQLRepository(instanceConnectionName, database, user, password, sslMode string) (ports.DBRepository, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		if instanceConnectionName == "" {
			return nil, fmt.Errorf("DB_MODE=cloud requires DB_INSTANCE_CONNECTION_NAME or DB_HOST")
		}
		host = filepath.ToSlash(filepath.Join("/cloudsql", instanceConnectionName))
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		password,
		database,
		sslMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Cloud SQL: %w", err)
	}

	return &CloudSQLRepository{db: db}, nil
}

// GetDB returns the underlying GORM instance.
func (r *CloudSQLRepository) GetDB() *gorm.DB {
	return r.db
}

// Ping verifies database connectivity.
func (r *CloudSQLRepository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get cloud database connection: %w", err)
	}
	return sqlDB.Ping()
}

// Close closes the cloud SQL connection.
func (r *CloudSQLRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get cloud database connection for close: %w", err)
	}
	return sqlDB.Close()
}

// RunMigrations executes .up.sql files in lexical order.
func (r *CloudSQLRepository) RunMigrations(migrationsPath string) error {
	if err := r.ensureMigrationsTable(); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, filepath.Join(migrationsPath, name))
		}
	}

	sort.Strings(files)
	for _, file := range files {
		name := filepath.Base(file)
		applied, checkErr := r.isMigrationApplied(name)
		if checkErr != nil {
			return checkErr
		}
		if applied {
			continue
		}

		content, readErr := os.ReadFile(file)
		if readErr != nil {
			return fmt.Errorf("failed to read migration %s: %w", filepath.Base(file), readErr)
		}
		if execErr := r.db.Exec(string(content)).Error; execErr != nil {
			if !isAlreadyExistsMigrationError(execErr) {
				return fmt.Errorf("failed to execute migration %s: %w", filepath.Base(file), execErr)
			}
		}
		if err := r.markMigrationApplied(name); err != nil {
			return err
		}
	}

	return nil
}

func (r *CloudSQLRepository) ensureMigrationsTable() error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	if err := r.db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}
	return nil
}

func (r *CloudSQLRepository) isMigrationApplied(name string) (bool, error) {
	var count int64
	if err := r.db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE name = ?", name).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check migration state for %s: %w", name, err)
	}
	return count > 0, nil
}

func (r *CloudSQLRepository) markMigrationApplied(name string) error {
	if err := r.db.Exec("INSERT INTO schema_migrations(name) VALUES (?)", name).Error; err != nil {
		return fmt.Errorf("failed to register migration %s: %w", name, err)
	}
	return nil
}