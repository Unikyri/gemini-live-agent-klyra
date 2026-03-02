# Klyra Project Rules and Configurations

## Architecture (Backend - Go)
- **Clean Architecture:** All Go code MUST be strictly separated into Infrastructure Layers (Handlers, DB Repositories), Use Cases (Business Logic), and Entities.
- **No Infrastructure Leaks:** Zero dependencies between Use Cases and network elements (HTTP/gRPC/WSS).
- **Database:** Migrations MUST ALWAYS be reversible (write `UP` and `DOWN` scripts). Tables must include `created_at` and `updated_at`. Use UUIDs. Soft-delete is preferred.

## Architecture (Frontend - Flutter)
- **Visual Control:** Visual Design and Animation (skill: `frontend-engineer`) is of utmost priority ("WOW effect"). Do NOT use "basic" UI; prioritize fluid transitions, micro-animations, and high-end design.
- **Independent Lip-Sync:** Separate the native Audio logic ("Barge-in" Live API) from the presentation logic so as not to block the UI Event Loop.

## Methodology & Collaboration
- **XP (Extreme Programming):** TDD-driven development for solid business logic.
- Continuous Pair Programming. Small stories to avoid long "Merge Conflicts".
- **Language:** All repository code, comments, variables, and documentation (except personal chat communication) MUST be written in English.

## Mandatory Security (`security-engineer`)
- **ALWAYS** assume a Shift-Left Security approach.
- Every endpoint (REST or WSS) MUST first validate the JWT and ensure the `user_id` in the Token has authorization over the requested resource (`course_id`, `topic_id`, etc).
- "Refresh Tokens" in Flutter must ONLY reside within Secure Storage. Do NOT print tokens in logs under ANY CIRCUMSTANCES.
