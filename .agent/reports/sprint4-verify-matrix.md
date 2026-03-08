# Sprint 4 RFC Spec Compliance Matrix

**Change**: sprint-4-planning  
**Verification Date**: 2026-03-08  
**Total RFC Scenarios**: 100  
**Tests with Passing Results**: 45  
**Test Gap**: 55 scenarios untested

---
## Test Execution Summary

| Component | Total Tests | Passed | Failed | Skipped | Build Status |
|-----------|------------|--------|--------|---------|--------------|
| Backend (Go) | 41 | 41 | 0 | 0 | ✅ Success |
| Mobile (Flutter) | 4 | 4 | 0 | 0 | ✅ Success |
| **TOTAL** | **45** | **45** | **0** | **0** | **✅ OK** |

---
## Full Test-to-RFC Traceability

| Test | Requirement ID | Status |
|------|----------------|--------|
| `backend/cmd/api/main_test.go::TestParseAllowedOrigins` | `REQ-CONFIG-001` | ✅ PASS |
| `backend/cmd/api/main_test.go::TestInitStorageService_LocalMode` | `REQ-STORAGE-001` | ✅ PASS |
| `backend/cmd/api/main_test.go::TestInitStorageService_GCSMode` | `REQ-STORAGE-002` | ✅ PASS |
| `backend/cmd/api/main_test.go::TestInitDBRepository_CloudMode_MissingConnection` | `REQ-DB-002` | ✅ PASS |
| `backend/cmd/api/main_test.go::TestInitDBRepository_LocalMode_ConnectionFailure` | `REQ-DB-001` | ✅ PASS |
| `backend/cmd/api/main_test.go::TestInitDBRepository_CloudMode_ConnectionFailure` | `REQ-DB-002` | ✅ PASS |
| `backend/internal/infrastructure/database/postgresql_repository_test.go::TestPostgreSQLRepository_RunMigrations_Idempotent` | `REQ-DB-MIGRATION-001` | ✅ PASS |
| `backend/internal/repositories/storage_service_test.go::TestLocalStorageService_UploadFile_WritesToDisk` | `REQ-STORAGE-001` | ✅ PASS |
| `backend/internal/repositories/storage_service_test.go::TestLocalStorageService_UploadFile_EmptyFile` | `REQ-MATERIAL-VALIDATION-003` | ✅ PASS |
| `backend/internal/repositories/storage_service_test.go::TestLocalStorageService_UploadFile_EmptyObjectName` | `REQ-STORAGE-VALIDATION-002` | ✅ PASS |
| `backend/internal/core/usecases/auth_usecase_test.go::TestAuthUseCase_GoogleSignIn_NewUser` | `REQ-AUTH-001` | ✅ PASS |
| `backend/internal/core/usecases/auth_usecase_test.go::TestAuthUseCase_GoogleSignIn_ExistingUser` | `REQ-AUTH-001` | ✅ PASS |
| `backend/internal/core/usecases/auth_usecase_test.go::TestAuthUseCase_GoogleSignIn_InvalidToken` | `REQ-AUTH-003` | ✅ PASS |
| `backend/internal/core/usecases/auth_usecase_test.go::TestAuthUseCase_GoogleSignIn_TokenGenerationError` | `REQ-AUTH-002` | ✅ PASS |
| `backend/internal/core/usecases/auth_usecase_test.go::TestAuthUseCase_GoogleSignIn_UserRepositoryError` | `REQ-AUTH-001` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_CreateCourse_NoImage` | `REQ-CRUD-001` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_CreateCourse_WithImage` | `REQ-CRUD-001` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_CreateCourse_InvalidUserID` | `REQ-CRUD-001` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_GetCoursesByUser` | `REQ-CRUD-003` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_GetCourseByID_Ownership_Valid` | `REQ-SECURITY-002` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_GetCourseByID_Ownership_Denied` | `REQ-SECURITY-003` | ✅ PASS |
| `backend/internal/core/usecases/course_usecase_test.go::TestCourseUseCase_CreateCourse_StorageUploadError` | `REQ-STORAGE-ERROR-001` | ✅ PASS |
| `backend/internal/handlers/http/auth_handler_test.go::TestAuthHandler_GoogleSignIn_Success` | `REQ-AUTH-001` | ✅ PASS |
| `backend/internal/handlers/http/auth_handler_test.go::TestAuthHandler_GoogleSignIn_MissingIDToken` | `REQ-AUTH-004` | ✅ PASS |
| `backend/internal/handlers/http/auth_handler_test.go::TestAuthHandler_GoogleSignIn_InvalidToken` | `REQ-AUTH-003` | ✅ PASS |
| `backend/internal/handlers/http/auth_handler_test.go::TestAuthHandler_GoogleSignIn_ReturnsUserData` | `REQ-AUTH-001` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_CreateCourse_Success` | `REQ-CRUD-001` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_CreateCourse_MissingName` | `REQ-VALIDATION-COURSE-001` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_ListCourses` | `REQ-CRUD-003` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_GetCourse_Success` | `REQ-CRUD-004` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_GetCourse_Ownership_Denied` | `REQ-SECURITY-003` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_AddTopic_Success` | `REQ-CRUD-002` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_AddTopic_MissingTitle` | `REQ-VALIDATION-TOPIC-001` | ✅ PASS |
| `backend/internal/handlers/http/course_handler_test.go::TestCourseHandler_AddTopic_Ownership_Denied` | `REQ-SECURITY-003` | ✅ PASS |
| `backend/internal/handlers/http/material_handler_test.go::TestMaterialHandler_UploadMaterial_WebBytes_PersistsToLocalStorage` | `REQ-MATERIAL-UPLOAD-001` | ✅ PASS |
| `backend/internal/handlers/http/material_handler_test.go::TestMaterialHandler_UploadMaterial_FilePathFlow_PersistsToLocalStorage` | `REQ-MATERIAL-UPLOAD-002` | ✅ PASS |
| `backend/internal/handlers/http/material_handler_test.go::TestMaterialHandler_UploadMaterial_OwnershipDenied_Returns403` | `REQ-SECURITY-004` | ✅ PASS |
| `backend/internal/handlers/http/material_handler_test.go::TestMaterialHandler_ListMaterials_OwnershipDenied_Returns403` | `REQ-SECURITY-004` | ✅ PASS |
| `backend/internal/handlers/http/material_handler_test.go::TestMaterialHandler_ListMaterials_Success` | `REQ-MATERIAL-UPLOAD-004` | ✅ PASS |
| `mobile/test/course/material_remote_datasource_test.dart::uploads web bytes (no path) successfully` | `REQ-FLUTTER-UPLOAD-001` | ✅ PASS |
| `mobile/test/course/material_remote_datasource_test.dart::uploads file path successfully` | `REQ-FLUTTER-UPLOAD-002` | ✅ PASS |
| `mobile/test/course/material_remote_datasource_test.dart::throws when file has no bytes and no path` | `REQ-FLUTTER-UPLOAD-003` | ✅ PASS |
| `mobile/test/widget_test.dart::Klyra app boots` | `REQ-FLUTTER-APP-001` | ✅ PASS |

---
## Remaining Coverage Snapshot

The detailed legacy per-area checklist was removed because it reflected pre-hardening counts.
Use this matrix as the single source of truth for executed tests and requirement links above.

### Current Gap Summary (55 scenarios)
- RAG pipeline end-to-end: chunking, embedding, similarity search.
2. Add focused integration tests for RAG and ownership/data-isolation edge cases.
3. Update this matrix with new test IDs and adjusted gap count.

---

**Report Generated**: 2026-03-08
