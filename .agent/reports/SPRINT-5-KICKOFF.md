# Sprint 5 Kickoff Plan
**Klyra Project: Gemini Live AI Tutor**

**Status**: Ready to start  
**Entry Criteria**: All Phase 4B + 5A tests passing ✅  
**Test Gate**: 63/100 RFC scenarios validated  
**Deployment Gate**: Backend build clean, migrations idempotent, Cloud Run ready  

---

## Sprint 5 Phases Overview

### Phase 5B: Staging Deployment + UI Testing (Est. 5-7 days)

#### 5B.1: Cloud Deployment
**Objective**: Validate that abstraction layers work in production GCP environment

**Tasks**:
1. Build and push Docker image to Google Container Registry (GCR)
2. Deploy to Cloud Run with environment variables:
   - `DB_MODE=cloud`
   - `DB_INSTANCE_CONNECTION_NAME=PROJECT:REGION:INSTANCE`
   - `STORAGE_MODE=gcs`
   - `GCS_BUCKET=klyra-materials`
3. Configure Cloud SQL Auth Proxy for Unix socket connection
4. Smoke test against real Cloud SQL + GCS:
   - POST `/auth/google-signin` → JWT created
   - POST `/courses` → Course created (owner scoped)
   - POST `/courses/{courseId}/materials` → File uploaded to GCS, Material record created
   - GET `/courses/{courseId}/materials` → 200 OK (owned materials only)
   - GET `/courses/{otherId}/materials` → 403 Forbidden (not owned)
5. Validate RAG pipeline end-to-end:
   - Material uploaded → Text extracted (async)
   - ProcessMaterialChunks triggered → pgvector populated
   - GET `/courses/{courseId}/context?query=...` → Top-K chunks returned ranked by relevance

**Success Criteria**:
- ✅ All HTTP endpoints return correct status codes (200, 403, 404, 500)
- ✅ Cloud SQL connectivity stable (no connection timeouts)
- ✅ GCS operations idempotent (re-upload same file → no duplicates)
- ✅ pgvector queries <10ms latency (p95)
- ✅ topicID isolation verified (cross-topic query returns 403)

**Owner**: DevOps Engineer  
**Duration**: 1-2 days

---

#### 5B.2: Flutter Widget Tests (Dashboard + Course Detail + Session)
**Objective**: Validate mobile UI state transitions and error handling

**Tasks**:
1. **Dashboard Screen** (4 tests)
   - Test empty course list (no materials uploaded)
   - Test course list with 3+ materials
   - Test tap course → navigation to detail screen
   - Test error state when fetch fails (show SnackBar with retry)

2. **Course Detail Screen** (4 tests)
   - Test topic list display
   - Test add-topic form (validation, success)
   - Test material list with expandable summaries
   - Test material delete with confirmation dialog

3. **Session Screen** (2 tests, basic)
   - Test audio recording state (idle → recording → paused)
   - Test context display (renders chunks with similarity scores)

**Test Utilities**:
- `WidgetTester.pumpWidget()` for rendering
- `mockito` for stubbed repository responses
- Custom matchers for error message validation

**Success Criteria**:
- ✅ All forms validate input correctly (required fields, email format)
- ✅ Error states display user-friendly messages
- ✅ Navigation works (push/pop consistent with Material spec)
- ✅ Async operations show loading spinner while pending

**Owner**: Frontend Engineer  
**Duration**: 3-4 days

---

#### 5B.3: Backend Integration Tests
**Objective**: Validate complete workflows (material upload → search → context retrieved)

**Tasks**:
1. Write 3 end-to-end test scenarios using real repositories (mocked only external APIs):
   ```go
   TestE2E_MaterialUpload_Then_Search
   TestE2E_MultiUserIsolation
   TestE2E_ChunkingAndEmbedding
   ```

2. Each test covers:
   - Create material (mock file upload to GCS)
   - Trigger text extraction (mocked, returns canned text)
   - Verify chunks saved to pgvector
   - Query with relevant topic/query
   - Validate top-K results ranked and scoped

3. Add performance benchmarks:
   - Chunking throughput: millions of runes per second
   - Embedding latency: p95 <500ms (including network)

**Success Criteria**:
- ✅ Material upload → chunks in pgvector (full flow)
- ✅ Cross-user isolation (User A chunks don't appear in User B searches)
- ✅ Chunking idempotent (re-upload same material → same chunks)

**Owner**: QA Engineer  
**Duration**: 2-3 days  
**Can run in parallel**: Yes, with 5B.2 (Flutter tests)

---

### Phase 5C: Session Management & Gemini Live Integration (Est. 4-5 days)

#### 5C.1: Gemini Live Context Injection
**Objective**: Integrate RAG context into Gemini Live chat flow

**Tasks**:
1. Design context window for Gemini Live:
   - Limit context to top-3 most relevant chunks (cost + latency)
   - Prioritize chunks from current topic
   - De-prioritize chunks >7 days old
   - Inject as user instruction: "Based on course material: [context], answer:'query'"

2. Implement context rotation:
   - User provides query (chat turn N)
   - System retrieves relevant chunks (topicID scoped)
   - Chunks injected into system prompt
   - Gemini response includes source attribution (which chunk was helpful)

3. Handle edge cases:
   - No chunks found → use generic system prompt
   - Max tokens exceeded → truncate context, log warning
   - Embedding API timeout → return generic context

**Success Criteria**:
- ✅ Gemini receives context for each user message
- ✅ Context limited to <2K tokens (cost control)
- ✅ Source attribution included in response

**Owner**: Backend Lead (Go) + Frontend Lead (UI display)  
**Duration**: 2 days

---

#### 5C.2: User Feedback Loop (Optional Enhancement)
**Objective**: Allow users to mark chunks as helpful/irrelevant for future ranking

**Tasks** (if time permits):
1. Add feedback table: `chunk_feedback(user_id, chunk_id, feedback_type: HELPFUL|IRRELEVANT)`
2. Modify SearchSimilar to boost score for HELPFUL-marked chunks
3. Add UI button: thumbs-up/thumbs-down in session screen
4. Collect metrics: most helpful chunks per topic

**Success Criteria**:
- ✅ Feedback persisted to database
- ✅ Subsequent searches for same topic reflect user preference
- ✅ Admin dashboard shows feedback statistics

**Owner**: Optional (backlog if time limited)  
**Duration**: 1-2 days (backlog)

---

## Test Addition Targets

### Phase 5B
- **Backend Integration**: +5 tests (E2E, multi-user, idempotence)
- **Flutter UI**: +10 tests (dashboard, detail, session)
- **Cloud Deployment**: +3 smoke tests (health, auth, RAG flow)
- **Subtotal**: 18 new tests

### Phase 5C
- **Gemini Context**: +4 tests (window size, rotation, edge cases)
- **User Feedback**: +2 tests (save, boost ranking) — optional
- **Session Management**: +3 tests (context state, multi-turn consistency)
- **Subtotal**: 7-9 new tests

**Running Total**: 63 (Sprint 4) + 25-27 (Sprint 5) = **88-90/100 RFC scenarios** by end of Sprint 5

---

## Resource Allocation

| Phase | Owner | Role | Effort | Parallel? |
|-------|-------|------|--------|----------|
| 5B.1 | DevOps | Cloud Run, Cloud SQL setup, smoke tests | 1-2d | Independent |
| 5B.2 | Frontend | Flutter widget tests | 3-4d | Yes, with 5B.3 |
| 5B.3 | QA | Backend integration tests | 2-3d | Yes, with 5B.2 |
| 5C.1 | Backend + Frontend | RAG context injection, UI display | 2d | After 5B.1 + 5B.3 |
| 5C.2 | Backend | User feedback loop | 1-2d | Optional, backlog |

**Critical Path**: 5B.1 → 5B.2/5B.3 (parallel) → 5C.1 → 5C.2 (optional)  
**Estimated Duration**: 5-7 days for Phase 5B, +2-3 days Phase 5C = **7-10 days total**

---

## Known Risks & Mitigation

| Risk | Likelihood | Mitigation |
|------|-----------|----------|
| Cloud SQL connectivity issues | Medium | Test Cloud SQL Proxy locally before staging (Docker container simulating Unix socket) |
| pgvector query timeout | Low | Profile queries against 10k+ real chunks; benchmark IVFFlat vs HNSW if needed |
| Flutter widget test flakiness | Medium | Mock all async operations (Network, DB), avoid real HTTP in tests |
| Gemini API rate limits | Medium | Implement exponential backoff, queue context injection requests if needed |
| topicID scoping bypassed | Low | Add security audit test: verify cross-topic queries return 403, not 200 with empty results |

---

## Definition of Done (Per Phase)

### Phase 5B.1 (Staging Deployment)
- [ ] Docker build succeeds, image pushed to GCR
- [ ] Cloud Run deployment succeeds with all env vars set
- [ ] `/health` returns 200 within 2 seconds
- [ ] POST `/auth/google-signin` returns JWT (test with mock token)
- [ ] POST `/courses` creates course, scoped to authenticated user
- [ ] GET `/courses/{otherId}/...` returns 403 Forbidden (not owned)
- [ ] pgvector query latency <10ms (p95)
- [ ] All smoke tests PASS

### Phase 5B.2 (Flutter UI Tests)
- [ ] Dashboard widget renders course list
- [ ] Tap course navigates to detail screen
- [ ] Add-topic form validates input
- [ ] Error states display SnackBar (no crash on network error)
- [ ] All 10 widget tests PASS

### Phase 5B.3 (Integration Tests)
- [ ] Material upload → text extraction → chunking → pgvector save flow completes
- [ ] Cross-user isolation verified (User A query doesn't return User B chunks)
- [ ] Chunking idempotence verified (re-run produces identical chunks)
- [ ] Similarity search ranking validated (relevant chunks ranked higher)
- [ ] All 5 integration tests PASS

### Phase 5C.1 (Gemini Context)
- [ ] Context window limited to top-3 chunks, topicID-scoped
- [ ] Context concatenated into system prompt pre-injection to Gemini
- [ ] Edge cases handled (no chunks, token limit, API timeout)
- [ ] All 4 context tests PASS

---

## Success Metrics

### By End of Phase 5B
- ✅ 63 + 18 = 81 tests passing
- ✅ Backend deployed to Cloud Run (staging)
- ✅ Flutter dashboard + course detail screens working
- ✅ E2E material upload → search flow validated
- ✅ topicID isolation verified end-to-end

### By End of Phase 5C
- ✅ 81 + 9 = 90 tests passing (10% RFC gap remaining)
- ✅ Gemini Live receiving RAG context for every user query
- ✅ Source attribution visible in session screen
- ✅ User feedback loop storing helpful/irrelevant marks
- ✅ System coherence validated (multi-turn session maintains context)

---

## Transition Plan from Sprint 4

### Before Starting Sprint 5
1. **Code Review**: Run through [SPRINT-4-CLOSURE.md] and [sdd-verify-sprint4-planning.md]
2. **Build Validation**: `go test ./...` → confirm 63/100 tests still passing
3. **Docker Setup**: `docker-compose up -d postgres` → test local development flow
4. **GCP Credentials**: Set up `GOOGLE_APPLICATION_CREDENTIALS` for Cloud SQL Proxy testing
5. **Kick-off Meeting**: Review Phase 5B timeline, assign owners, confirm priorities

### Development Environment Checklist
- ✅ Go 1.22+ installed (`go version`)
- ✅ PostgreSQL + pgvector running locally (`docker-compose up`)
- ✅ Flutter 3.x with latest dependencies (`flutter pub get`)
- ✅ All backend tests passing (`go test ./...`)
- ✅ All Flutter tests passing (`flutter test`)
- ✅ GCP project configured (Cloud Run, Cloud SQL, GCS buckets)

---

## Appendix: Command Reference

### Local Development
```bash
# Start PostgreSQL + pgvector
docker-compose up -d postgres

# Run all backend tests
cd backend && go test ./... -v

# Run only integration tests
go test -tags=integration ./... -v

# Run integration tests for one package
go test -tags=integration ./internal/repositories -v -run TestChunkRepository

# Start local API server
go run ./cmd/api/main.go
```

### Cloud Deployment (Sample)
```bash
# Build Docker image
docker build -t gcr.io/my-project/klyra-backend:latest .

# Push to GCR
docker push gcr.io/my-project/klyra-backend:latest

# Deploy to Cloud Run
gcloud run deploy klyra-backend \
  --image=gcr.io/my-project/klyra-backend:latest \
  --region=us-central1 \
  --platform=managed \
  --set-env-vars="DB_MODE=cloud,DB_INSTANCE_CONNECTION_NAME=my-project:us-central1:klyra-db"

# View logs
gcloud run logs read klyra-backend --region=us-central1
```

### Flutter Testing
```bash
# Run all Flutter tests
flutter test

# Run single test file
flutter test test/features/course/presentation/dashboard_test.dart

# Run with verbose output
flutter test -v

# Run integration tests (device required)
cd mobile && flutter drive --target=test_driver/app.dart
```

---

**Document Version**: 1.0  
**Last Updated**: 2026-03-08  
**Prepared By**: Orchestrator (SDD Agent)  
**Next Review**: Start of Phase 5B (Staging Deployment)