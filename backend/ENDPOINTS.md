# Backend API Endpoints Reference

## Authentication Endpoints

### POST /api/v1/auth/google
**Google OAuth Authentication** (Production)
- Request: `{ id_token: string }`
- Response: `{ access_token, refresh_token, user }`
- Status: 200 OK, 400 Bad Request, 500 Server Error
- Security: Validates Google JWT signature, extracts email/ID

### POST /api/v1/auth/google-mock  
**Guest/Mock Authentication** (Local Development Only)
- Request: `{ email?: string, name?: string }`
- Response: `{ access_token, refresh_token, user }`
- Status: 200 OK, 500 Server Error
- Warning: ⚠️ DEVELOPMENT ONLY - Requires ENV=local check before production
- Note: Skips Google verification, generates JWT for guest user

### POST /api/v1/auth/refresh
**Token Refresh**
- Request: `{ refresh_token: string }`
- Response: `{ access_token, expires_in }`
- Status: 200 OK, 401 Unauthorized, 500 Server Error
- Security: Validates refresh token, returns short-lived access token

## Course Management Endpoints

### POST /api/v1/courses
**Create Course**
- Request: `{ title, description?, owner_id, thumbnail? }`
- Response: `{ id, title, description, owner_id, created_at }`
- Status: 201 Created, 400 Bad Request, 500 Server Error
- Auth: Required (via access_token header)
- Validation: Title length > 3, owner_id UUID format

### GET /api/v1/courses/:id
**Get Course Details**
- Response: `{ id, title, description, topics: [...], materials: [...] }`
- Status: 200 OK, 404 Not Found, 500 Server Error
- Includes: All topics and materials in course

### GET /api/v1/users/:userId/courses
**List User's Courses**
- Response: `{ courses: [{ id, title, owner_id, created_at }] }`
- Status: 200 OK, 404 Not Found, 500 Server Error
- Pagination: ?limit=20&offset=0 (optional)

## Topic Management Endpoints

### POST /api/v1/courses/:courseId/topics
**Create Topic**
- Request: `{ title, sequence }`
- Response: `{ id, course_id, title, sequence }`
- Status: 201 Created, 400 Bad Request, 500 Server Error
- Validation: Unique title within course, order by sequence

### GET /api/v1/courses/:courseId/topics
**List Course Topics**
- Response: `{ topics: [{ id, title, sequence, material_count }] }`
- Status: 200 OK, 404 Not Found
- Ordered by: sequence ASC

## Material Management Endpoints

### POST /api/v1/topics/:topicId/materials
**Upload Material**
- Request: multipart/form-data { file, title }
- Response: `{ id, topic_id, title, storage_url, chunks_created }`
- Status: 201 Created, 400 Bad Request, 413 Payload Too Large, 500 Server Error
- File Size Limit: 50MB
- Supported Types: .pdf, .txt, .docx
- Process: Extract text → Generate embeddings → Create chunks

### GET /api/v1/materials/:id
**Get Material Details**
- Response: `{ id, topic_id, title, content, chunks: [...] }`
- Status: 200 OK, 404 Not Found, 500 Server Error

### GET /api/v1/topics/:topicId/materials
**List Topic Materials**
- Response: `{ materials: [{ id, title, file_type, chunk_count }] }`
- Status: 200 OK, 404 Not Found, 500 Server Error

## RAG (Vector Search) Endpoints

### POST /api/v1/topics/:topicId/process
**Process Material for RAG**
- Request: `{ material_id }`
- Response: `{ chunks_created, embeddings_generated }`
- Status: 202 Accepted, 400 Bad Request, 500 Server Error
- Process: Extract text → Chunk by sentences → Generate embeddings → Store in pgvector
- Async: Returns immediately, processing happens in background

### POST /api/v1/topics/:topicId/query
**Query Topic Context (Vector Search)**
- Request: `{ query, limit: 5 }`
- Response: `{ context: [{ content, similarity: 0.95, material_id }], total_results }`
- Status: 200 OK, 400 Bad Request, 500 Server Error
- Search: Cosine similarity on pgvector embeddings (<10ms for 10k+ documents with IVFFlat index)
- Ranking: By similarity score descending

## Error Response Format

```json
{
  "error": "Error title",
  "message": "Detailed error message",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Security Headers Required

| Header | Values | Purpose |
|--------|--------|----------|
| Authorization | Bearer {access_token} | OAuth token (except /auth/google) |
| Content-Type | application/json, multipart/form-data | Request format |
| X-Request-ID | UUID | Request tracing (optional) |

## Rate Limiting

- Auth endpoints: 10 req/sec per IP
- Course endpoints: 100 req/min per user
- File upload: 5 concurrent uploads per user
- Vector search: 1000 queries/day per user (production)

## Testing Notes

- **Mock auth**: POST /auth/google-mock with {email: "test@example.com"}
- **Local database**: PostgreSQL on localhost:5433
- **Vector dimension**: 384 (Vertex AI embeddings)
- **Chunk size**: 512 tokens with 100-token overlap
- **Test data**: See backend/migrations/fixtures/seed-pgvector-test-data.sql
