# Change 11 Workstream - Doc/Reality Reconciliation

Date: 2026-06-10
Status: Planned
Scope: Bring the planning documents back in line with the implemented system, correct overstated workstream statuses, record the actual stack as an ADR, and clean repo hygiene. No production code behavior changes.

## Objective

The planning docs have drifted from the code. `Technical_Stack.md` still lists GORM, Asynq, Redis, and sqlite-vec — none are in `go.mod`. Change 9 (Wails) and Change 10 (GBUS) are marked Complete but are, respectively, a half-wired scaffold and an untrained placeholder. Drift is the highest-leverage threat: every future decision (human or AI agent) reads the plans and inherits a wrong architecture. This change makes the documents tell the truth. It is docs-only and carries zero runtime risk.

## Guiding Constraints

- Documentation only — no `internal/`, `cmd/`, or `frontend/src` behavior changes (except `.gitignore`).
- Do not delete historical ADRs; supersede with new records (preserve decision history per ADR process).
- Status changes must reflect verified code state, not aspiration.
- Keep the ADR numbering sequential (next is 0019).

## Workstream 1 — Technical Stack Reconciliation

Objective:
Make `Technical_Stack.md` describe the system that exists.

Key tasks:
- [ ] Replace GORM references with the actual `database/sql` + hand-rolled repository adapters.
- [ ] Remove/mark-superseded Asynq + Redis; document the in-process goroutine queue (`deep_processor`) as the actual job mechanism.
- [ ] Replace sqlite-vec with the actual pure-Go brute-force cosine vector search (`internal/repository/sqlite/vector_repository.go`).
- [ ] Update AI model references (gpt-4o / claude-3-5-sonnet) to current config-driven model names; note models are config-driven, not hardcoded.
- [ ] Add a "Superseded sections" banner pointing to ADR 0019.

Deliverables:
- [ ] Updated `Plans/Technical_Stack.md`.

Done criteria:
- [ ] No technology named in `Technical_Stack.md` is absent from `go.mod` / `package.json` without an explicit "planned, not implemented" tag.

## Workstream 2 — Actual-Stack ADR

Objective:
Record the implemented stack as a durable decision so the reconciliation is not re-litigated.

Key tasks:
- [ ] Create `Plans/ADR/0019-actual-stack-vs-planned-stack.md` documenting: database/sql over GORM, in-process queue over Asynq/Redis, brute-force cosine over sqlite-vec, and the reasoning for each divergence.
- [ ] Add ADR 0019 to the ADR index in `Plans/ADR/README.md`.

Deliverables:
- [ ] `Plans/ADR/0019-actual-stack-vs-planned-stack.md`.
- [ ] Updated `Plans/ADR/README.md` index.

Done criteria:
- [ ] ADR 0019 is Accepted and indexed.

## Workstream 3 — Workstream Status Correction

Objective:
Stop the "Complete" overstatement on Changes 9 and 10.

Key tasks:
- [ ] `Change_9_Workstream.md`: status → `In Progress`. Untick WS2 IPC done-criteria (bindings not generated), WS3 native-feature criteria, and any milestone not code-verified.
- [ ] `Change_10_Workstream.md`: status → `Scaffold (model not trained)`. Untick WS3 ("trained", ">5% lift") and WS4/WS5 claims that depend on a real model. Keep the signal/aggregator/feature-store/inference-fallback items that genuinely exist.
- [ ] `Changes.md`: add honest "What we did" entries for Changes 8–10 reflecting verified state.
- [ ] `Changes_log.md`: append the reconciliation session entry.

Deliverables:
- [ ] Updated `Change_9_Workstream.md`, `Change_10_Workstream.md`, `Changes.md`, `Changes_log.md`.

Done criteria:
- [ ] No workstream is marked Complete unless its done-criteria are verifiable in code.

## Workstream 4 — Repo Hygiene

Objective:
Stop tracking build artifacts.

Key tasks:
- [ ] Add `*.exe`, `/server.exe`, `/desktop.exe`, `bin/`, `dist/` to `.gitignore`.
- [ ] Remove any committed binaries from the working tree (git rm --cached only; do not delete local files needed for testing).

Deliverables:
- [ ] Updated `.gitignore`.

Done criteria:
- [ ] `git status` shows no tracked build artifacts.

## Planned Milestones

- [ ] Milestone 11A: Technical_Stack.md reconciled (WS1 complete).
- [ ] Milestone 11B: ADR 0019 recorded and indexed (WS2 complete).
- [ ] Milestone 11C: Change 9/10 statuses corrected (WS3 complete).
- [ ] Milestone 11D: Repo hygiene applied (WS4 complete).

## Change 11 Definition of Done

- [ ] Technical_Stack.md names only technologies that exist (or tags planned ones).
- [ ] ADR 0019 records the actual stack and is indexed.
- [ ] Changes 9 and 10 statuses reflect verified code state, not aspiration.
- [ ] Build artifacts are gitignored.
- [ ] No code behavior changed; `go test ./...` still passes.
