package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPostgreSQLRepository_RunMigrations_Idempotent(t *testing.T) {
	repo, err := NewPostgreSQLRepository("localhost", "5433", "klyra_db", "klyra_user", "klyra_pass", "disable")
	if err != nil {
		t.Skipf("skipping integration migration test, local db unavailable: %v", err)
	}

	dbRepo, ok := repo.(*PostgreSQLRepository)
	if !ok {
		t.Fatalf("expected *PostgreSQLRepository, got %T", repo)
	}
	defer func() {
		_ = dbRepo.Close()
	}()

	tmpDir := t.TempDir()
	migrationFile := filepath.Join(tmpDir, "000001_create_verify_table.up.sql")
	migrationSQL := `CREATE TABLE IF NOT EXISTS verify_migration_test (id SERIAL PRIMARY KEY);`
	if writeErr := os.WriteFile(migrationFile, []byte(migrationSQL), 0o644); writeErr != nil {
		t.Fatalf("failed to write temp migration: %v", writeErr)
	}

	if err := dbRepo.RunMigrations(tmpDir); err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}

	if err := dbRepo.RunMigrations(tmpDir); err != nil {
		t.Fatalf("second migration run should be idempotent, got: %v", err)
	}
}
