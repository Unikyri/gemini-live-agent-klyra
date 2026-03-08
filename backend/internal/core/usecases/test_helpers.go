//go:build test_helpers
// +build test_helpers

package usecases

// Exported mock constructors for use in integration tests across packages.
// These allow handler tests to use the same mocks as use case tests.

// Exported type aliases for mocks (makes them importable)
type ExportedMockUserRepository = MockUserRepository
type ExportedMockTokenService = MockTokenService
type ExportedMockGoogleTokenVerifier = MockGoogleTokenVerifier
type ExportedMockCourseRepository = MockCourseRepository
type ExportedMockStorageService = MockStorageService
type ExportedMockAvatarGenerator = MockAvatarGenerator
type ExportedMockTopicRepository = MockTopicRepository
