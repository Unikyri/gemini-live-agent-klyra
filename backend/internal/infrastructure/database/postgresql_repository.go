package database

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// postgresConfig holds PostgreSQL connection parameters
type postgresConfig struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	SSLMode  string
}

// PostgreSQLRepository implements ports.DBRepository for direct PostgreSQL connections
type PostgreSQLRepository struct {
	db     *gorm.DB
	config postgresConfig
}

// NewPostgreSQLRepository creates a new PostgreSQL connection using the provided configuration.
// This is the "local" mode implementation for development environments.
func NewPostgreSQLRepository(host, port, database, user, password, sslMode string) (ports.DBRepository, error) {
	config := postgresConfig{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
		SSLMode:  sslMode,
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return &PostgreSQLRepository{
		db:     db,
		config: config,
	}, nil
}

// GetDB returns the underlying GORM database instance
func (r *PostgreSQLRepository) GetDB() *gorm.DB {
	return r.db
}

// Ping verifies the database connection is alive
func (r *PostgreSQLRepository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.Ping()
}

// Close terminates the database connection
func (r *PostgreSQLRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection for closing: %w", err)
	}
	return sqlDB.Close()
}

// RunMigrations executes SQL migration files from the specified directory.
// Migrations must be named sequentially (000001_*.up.sql, 000002_*.up.sql, etc.)
// and will be executed in order.
func (r *PostgreSQLRepository) RunMigrations(migrationsPath string) error {
	if err := r.ensureMigrationsTable(); err != nil {
		return err
	}

	// Get all .up.sql files
	files, err := getMigrationFiles(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsPath)
	}

	// Execute each migration
	for _, file := range files {
		name := filepath.Base(file)
		applied, checkErr := r.isMigrationApplied(name)
		if checkErr != nil {
			return checkErr
		}
		if applied {
			continue
		}

		if err := r.executeMigration(file); err != nil {
			if !isAlreadyExistsMigrationError(err) {
				return fmt.Errorf("migration failed for %s: %w", filepath.Base(file), err)
			}
		}
		if err := r.markMigrationApplied(name); err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgreSQLRepository) ensureMigrationsTable() error {
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

func (r *PostgreSQLRepository) isMigrationApplied(name string) (bool, error) {
	var count int64
	if err := r.db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE name = ?", name).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check migration state for %s: %w", name, err)
	}
	return count > 0, nil
}

func (r *PostgreSQLRepository) markMigrationApplied(name string) error {
	if err := r.db.Exec("INSERT INTO schema_migrations(name) VALUES (?)", name).Error; err != nil {
		return fmt.Errorf("failed to register migration %s: %w", name, err)
	}
	return nil
}

func isAlreadyExistsMigrationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "sqlstate 42p07")
}

// getMigrationFiles returns a sorted list of .up.sql migration files
func getMigrationFiles(migrationsPath string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".up.sql") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files) // Ensures sequential execution (000001, 000002, ...)
	return files, nil
}

// executeMigration runs a single SQL migration file
func (r *PostgreSQLRepository) executeMigration(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if err := r.db.Exec(string(content)).Error; err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}
