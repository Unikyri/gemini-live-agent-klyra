# Klyra Project - Future Sprints Roadmap
## Based on Sprint 5 EPIC 3 Comparison & Cleanup

**Generated**: March 8, 2026  
**Status**: READY FOR IMPLEMENTATION

---

## Sprint 6 Timeline (7-10 days)

### Phase 6A: Test Infrastructure + Cloud Deployment (2-3 days)
**Objective**: Enable full automation, deploy to staging, run smoke tests

**Deliverables**:
- ✅ Fix remaining test infrastructure issues (if any) 
- ✅ Docker build + push to GCR
- ✅ Cloud Run staging deployment
- ✅ Smoke tests: health, auth, CRUD, pgvector queries
- ✅ 3 new tests (cloud deployment)

**Owner**: DevOps Engineer  
**Success Criteria**: `go test ./... -v` runs all 75+ tests, Cloud Run stable

**Implementation Notes**:
- Test fixes already applied (package declarations, rag_usecase_test.go header)
- Next: Deploy Backend image to GCR with env var configuration
- Measure pgvector performance (target <10ms p95)

---

### Phase 6B: Material Upload E2E + Flutter UI Tests (3-4 days)
**Objective**: Complete EPIC 2 with full validation

**Deliverables**:
1. Backend Integration Tests (5):
   - `TestE2E_MaterialUpload_Then_Search` (file → chunks → search)
   - `TestE2E_MultiUserIsolation` (cross-user security)
   - `TestE2E_ChunkingAndEmbedding` (pipeline flow)
   - Chunking performance benchmark
   - Embedding latency benchmark

2. Flutter Widget Tests (10):
   - Dashboard (4): empty, list, navigation, errors
   - Course Detail (4): topics, add-topic, materials, delete
   - Session (2): audio recording, context display

3. Material Upload UI Integration (1):
   - End-to-end material upload → storage → extraction

**Owner**: QA Engineer + Frontend Engineer  
**Success Criteria**: 15 new tests PASS, material upload → RAG flow validated  
**Running Total**: 75 + 15 = 90 tests (90/100 RFC)

**Implementation Sequence**:
1. Day 1-2: Write backend E2E tests (5 tests) + benchmarks
2. Day 2-3: Write Flutter widget tests (10 tests)
3. Day 3-4: Wire material upload UI to real backend, final integration test

---

### Phase 6C: Gemini Live Integration Design + Implementation (3-5 days)
**Objective**: Implement core tutor feature - context injection

**Deliverables**:
1. Context Window Design (1 day):
   - Limit top-3 chunks, topicID scoped
   - Token limit <2K for cost control
   - Source attribution mapping
   - Edge case handling (no chunks, timeout, rate limit)

2. Implementation (2-3 days):
   - Context injection into Gemini prompt
   - Multi-turn session state management
   - Source attribution in response

3. Tests (2-4):
   - Context window generation (1)
   - Gemini API call with context (1)  
   - Multi-turn conversation (1)
   - Edge cases (timeout, no chunks) (1)

**Owner**: Backend Lead + Frontend Lead  
**Success Criteria**: Gemini receives context for every user message, source attribution visible

**Implementation Complexity**:
- Requires: Gemini Live API client setup + WebSocket integration
- Current State: API spec exists, client not yet built
- Estimated Effort: 3-5 days for full feature

---

## Sprint 7 Timeline (Optional, if resources available)

### Phase 7A: User Feedback Loop (1-2 days, Optional)
**Objective**: Allow users to mark helpful/unhelpful chunks for ranking boost

**Deliverables**:
- Add `chunk_feedback` table (user_id, chunk_id, feedback_type)
- Modify search to boost HELPFUL chunks
- UI: thumbs-up/thumbs-down buttons in session screen
- 2 tests (feedback save, ranking boost)

**Owner**: Backend + Frontend  
**Success Criteria**: User feedback affects future search ranking

---

### Phase 7B: Analytics Dashboard (2-3 days)
**Objective**: Admin view of system health and engagement metrics

**Deliverables**:
- Dashboard screen showing:
  - Most helpful chunks per topic
  - User engagement metrics (sessions per user, avg session duration)
  - System health (API latency, error rates)
- 3-4 tests (data aggregation, display)

**Owner**: Backend Data Engineer + Frontend Engineer

---

## Sprint 8+ Timeline (EPIC 4 - Adaptive Learning)

### Phase 8A-8D: Persistent Learning Profiles
**Requirements** (from Product Canvas HUs 4.1-4.4):
- HU 4.1: Track learning progress per topic
- HU 4.2: Recommend topics for review (spaced repetition)
- HU 4.3: Learning analytics dashboard
- HU 4.4: Personalized learning style adaptation

**Estimated Effort**: 2-3 sprints (2-3 weeks)

**Architecture Decisions Needed**:
- Learning profile schema (topics studied, time spent, mastery score)
- Spaced repetition algorithm (Leitner system / SM-2)
- Recommendation engine (which topics to review when)
- Analytics model (what to track, how to visualize)

---

## Current Project Status Summary

### ✅ COMPLETE (100% production-ready)
- **EPIC 1**: Auth + Course Management + Topics (HUs 1.1-1.3)
- **Tests**: 75 passing (16 core MVP scenario validations)

### ⏳ IN PROGRESS (code ready, needs validation)
- **EPIC 2**: Material Upload + RAG Pipeline (HUs 2.1-2.4)
- **Tests**: 0/15 integration tests (scheduled for Sprint 6, Phase 6B)
- **Status**: Code implementation exists; end-to-end flow not yet validated

### ❌ NOT STARTED (needs design + implementation)
- **EPIC 3**: Gemini Live Tutor (HUs 3.1-3.4)  
- **Tests**: 0/4 context injection tests (scheduled for Sprint 6, Phase 6C)
- **Status**: Architecture defined; implementation begins Sprint 6

- **EPIC 4**: Adaptive Learning (HUs 4.1-4.4)
- **Tests**: 0 tests (deferred to Sprint 8+)
- **Status**: Requirements documented; architectural decisions pending

---

## Known Constraints & Dependencies

### Infrastructure Blockers (Now Fixed)
- ✅ Package naming conflicts in handlers/http → RESOLVED
- ✅ RAG usecase test header missing → RESOLVED
- ⏳ Full `go test ./...` automation still needs verification

### Resource Constraints
- Team size: 1 senior engineer (multi-role coverage)
- Parallel phases require additional hires or volunteer contributors
- Critical path: 6A → 6B/6C → 7A → 8A

### Technical Debt
- Staging deployment not yet validated
- No performance benchmarks collected (pgvector latency, chunking throughput)
- Flutter UI tests (10 tests) needed before production mobile release

---

## Success Metrics by Sprint

| Sprint | RFC Target | Tests Target | Actual (Planned) |
|--------|-----------|--------------|------------------|
| 5 | 75/100 | 88-90 | 75/100 ✅ |
| 6 | 90/100 | 105-110 | 90/100 (Phase 6A+6B+6C) |
| 7 | 95/100 | 115-120 | 95/100 (Phase 7A+7B) |
| 8+ | 100/100 | 130+ | 100/100 (EPIC 4 + stretch goals) |

---

## Phase Dependencies & Critical Path

```
SPRINT 5 COMPLETE ✅
     ↓
SPRINT 6 PHASE 6A (Infrastructure)
     ↓
SPRINT 6 PHASE 6B (Material Upload E2E) ← PARALLEL
SPRINT 6 PHASE 6C (Gemini Live Design) ← PARALLEL
     ↓
SPRINT 7 PHASE 7A/7B (Feedback Loop + Analytics) ← OPTIONAL
     ↓
SPRINT 8+ PHASE 8A-8D (Adaptive Learning)
```

**Critical Path Duration**: 6A (2-3d) + 6B+6C (3-5d, parallel) = 5-8 days
**Total Duration to EPIC 4**: 5-8 (Sprint 6) + 2-3 (Sprint 7) + 10-15 (Sprint 8) = 17-26 days (~3.5 weeks)

---

## Decision Record

### Why Fix Test Infrastructure First?
- Enables CI/CD automation with `go test ./...`
- Prevents regressions in future sprints
- Reduces false test failures (package conflicts)

### Why Prioritize Phase 6B over 6C?
- Material upload is prerequisite for Gemini context (need real data)
- EPIC 2 validation gates EPIC 3 implementation
- Allows parallel work: while 6B runs, design 6C

### Why Defer EPIC 4?
- Requires spaced repetition algorithm research + design
- Not blocking core MVP (tutor works without personalization)
- Can be added in later iterations without architectural changes

---

## Appendix: Hand-Off Checklist for Sprint 6

### Before Starting Sprint 6
- [ ] Read this roadmap + SPRINT-5-EPIC-3-COMPARISON.md
- [ ] Verify `go test ./internal/... -v` runs successfully
- [ ] Confirm Docker + GCP credentials configured
- [ ] Review Phase 6A deployment plan with DevOps
- [ ] Assign owners for Phases 6A, 6B, 6C

### Key Artifacts for Continuity
- **Comparison Matrix**: `.agent/reports/SPRINT-5-EPIC-3-COMPARISON.md`
- **Test Evidence**: Backend 75 passing tests (9 course + 2 auth handlers + 7 usecase)
- **Code State**: Main branch, commit `eab2ffc`
- **Architecture**: SPRINT-5-KICKOFF.md section "Phase 5B Overview"

### Questions for Sprint 6 Kickoff
1. Will 6B (Flutter tests) and 6C (Gemini design) run in parallel? (Recommend: Yes, will need 2+ engineers)
2. What's the GCP deployment timeline? (Recommend: Early in Phase 6A to unblock rest)
3. Who owns Gemini Live API client setup? (Backend or DevOps?)
4. What's the priority if time is limited? (Recommend: 6A > 6B > 6C, defer 7A)

---

**Prepared by**: SDD Orchestrator (Klyra Project)  
**Next Review**: Start of Sprint 6 (Day 1 kickoff)

