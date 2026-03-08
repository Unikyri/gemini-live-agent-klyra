package main

import (
	"strings"
	"testing"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/repositories"
)

func TestParseAllowedOrigins(t *testing.T) {
	origins := parseAllowedOrigins("http://localhost:3000, http://localhost:8080")
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if origins[0] != "http://localhost:3000" || origins[1] != "http://localhost:8080" {
		t.Fatalf("unexpected origins: %#v", origins)
	}
}

func TestInitStorageService_LocalMode(t *testing.T) {
	t.Setenv("STORAGE_MODE", "local")
	t.Setenv("STORAGE_PATH", "./test-storage")

	svc := initStorageService()
	if _, ok := svc.(*repositories.LocalStorageService); !ok {
		t.Fatalf("expected LocalStorageService, got %T", svc)
	}
}

func TestInitStorageService_GCSMode(t *testing.T) {
	t.Setenv("STORAGE_MODE", "gcs")

	svc := initStorageService()
	if _, ok := svc.(*repositories.GCSStorageService); !ok {
		t.Fatalf("expected GCSStorageService, got %T", svc)
	}
}

func TestInitDBRepository_CloudMode_MissingConnection(t *testing.T) {
	t.Setenv("DB_MODE", "cloud")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_INSTANCE_CONNECTION_NAME", "")
	t.Setenv("INSTANCE_CONNECTION_NAME", "")
	t.Setenv("DB_NAME", "klyra_db")
	t.Setenv("DB_USER", "klyra_user")
	t.Setenv("DB_PASSWORD", "klyra_pass")
	t.Setenv("DB_SSL_MODE", "disable")

	_, err := initDBRepository()
	if err == nil {
		t.Fatal("expected error when cloud connection data is missing")
	}
	if !strings.Contains(err.Error(), "DB_MODE=cloud requires DB_INSTANCE_CONNECTION_NAME or DB_HOST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitDBRepository_LocalMode_ConnectionFailure(t *testing.T) {
	t.Setenv("DB_MODE", "local")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "1")
	t.Setenv("DB_NAME", "klyra_db")
	t.Setenv("DB_USER", "klyra_user")
	t.Setenv("DB_PASSWORD", "klyra_pass")
	t.Setenv("DB_SSL_MODE", "disable")

	_, err := initDBRepository()
	if err == nil {
		t.Fatal("expected connection failure in local mode")
	}
	if !strings.Contains(err.Error(), "failed to connect to PostgreSQL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitDBRepository_CloudMode_ConnectionFailure(t *testing.T) {
	t.Setenv("DB_MODE", "cloud")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "1")
	t.Setenv("DB_INSTANCE_CONNECTION_NAME", "")
	t.Setenv("DB_NAME", "klyra_db")
	t.Setenv("DB_USER", "klyra_user")
	t.Setenv("DB_PASSWORD", "klyra_pass")
	t.Setenv("DB_SSL_MODE", "disable")

	_, err := initDBRepository()
	if err == nil {
		t.Fatal("expected connection failure in cloud mode")
	}
	if !strings.Contains(err.Error(), "failed to connect to Cloud SQL") {
		t.Fatalf("unexpected error: %v", err)
	}
}
