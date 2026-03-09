package repositories

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorageService_UploadFile_WritesToDisk(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewLocalStorageService(tmpDir, "")

	url, err := svc.UploadFile(context.Background(), "", "materials/test.txt", []byte("hello"), "text/plain")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(url, "/static/materials/test.txt") {
		t.Fatalf("unexpected URL: %s", url)
	}

	storedPath := filepath.Join(tmpDir, "materials", "test.txt")
	content, readErr := os.ReadFile(storedPath)
	if readErr != nil {
		t.Fatalf("expected file to exist: %v", readErr)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected file content: %s", string(content))
	}
}

func TestLocalStorageService_UploadFile_EmptyFile(t *testing.T) {
	svc := NewLocalStorageService(t.TempDir(), "")

	_, err := svc.UploadFile(context.Background(), "", "materials/empty.txt", []byte{}, "text/plain")
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "empty file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocalStorageService_UploadFile_EmptyObjectName(t *testing.T) {
	svc := NewLocalStorageService(t.TempDir(), "")

	_, err := svc.UploadFile(context.Background(), "", "", []byte("x"), "text/plain")
	if err == nil {
		t.Fatal("expected error for empty objectName")
	}
	if !strings.Contains(err.Error(), "objectName is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocalStorageService_CustomBaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	customBaseURL := "http://10.0.2.2:8080/static"
	svc := NewLocalStorageService(tmpDir, customBaseURL)

	url, err := svc.UploadFile(context.Background(), "", "avatars/123/avatar.png", []byte("fake-image"), "image/png")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(url, "http://10.0.2.2:8080/static") {
		t.Fatalf("expected URL to contain custom base URL, got: %s", url)
	}
	if !strings.Contains(url, "avatars/123/avatar.png") {
		t.Fatalf("expected URL to contain object path, got: %s", url)
	}
}

func TestLocalStorageService_DefaultBaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewLocalStorageService(tmpDir, "") // Empty string triggers fallback

	url, err := svc.UploadFile(context.Background(), "", "avatars/456/avatar.png", []byte("fake-image"), "image/png")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(url, "http://localhost:8080/static") {
		t.Fatalf("expected URL to contain default localhost base URL, got: %s", url)
	}
	if !strings.Contains(url, "avatars/456/avatar.png") {
		t.Fatalf("expected URL to contain object path, got: %s", url)
	}
}
